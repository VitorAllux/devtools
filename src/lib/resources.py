#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import re
import shlex
import shutil
import subprocess
import sys
from dataclasses import dataclass, field


ACTIONS = ("start", "stop", "restart")
SERVICE_DEFINITIONS = (
    ("redis-server", "Redis"),
    ("postgresql", "PostgreSQL"),
    ("mysql", "MySQL"),
    ("mariadb", "MariaDB"),
    ("docker", "Docker daemon"),
)


@dataclass(frozen=True)
class CommandResult:
    returncode: int
    stdout: str = ""
    stderr: str = ""

    @property
    def output(self) -> str:
        return clean_text(f"{self.stdout}\n{self.stderr}")


@dataclass(frozen=True)
class Resource:
    id: str
    kind: str
    name: str
    state: str
    available: bool
    manager: str
    details: str
    actions: tuple[str, ...] = field(default_factory=tuple)
    requires_sudo: bool = False
    target: str = ""
    compose_files: tuple[str, ...] = field(default_factory=tuple)


@dataclass(frozen=True)
class ActionResult:
    success: bool
    message: str
    output: str = ""
    requires_terminal: bool = False
    command: tuple[str, ...] = field(default_factory=tuple)


def clean_text(text: str) -> str:
    return text.replace("\x00", "").replace("\r", "").strip()


def first_line(text: str, fallback: str = "") -> str:
    for line in clean_text(text).splitlines():
        line = line.strip()
        if line:
            return line[:180]
    return fallback


def run_command(cmd: list[str], timeout: int = 5) -> CommandResult:
    try:
        proc = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
    except FileNotFoundError as exc:
        return CommandResult(127, "", str(exc))
    except PermissionError as exc:
        return CommandResult(126, "", str(exc))
    except subprocess.TimeoutExpired as exc:
        stdout = exc.stdout if isinstance(exc.stdout, str) else ""
        stderr = exc.stderr if isinstance(exc.stderr, str) else ""
        return CommandResult(124, stdout, stderr or "Command timed out")

    return CommandResult(proc.returncode, proc.stdout or "", proc.stderr or "")


def current_user_needs_sudo() -> bool:
    return hasattr(os, "geteuid") and os.geteuid() != 0


def service_status_entries() -> dict[str, str]:
    if not shutil.which("service"):
        return {}

    result = run_command(["service", "--status-all"], timeout=8)
    entries: dict[str, str] = {}
    for line in result.output.splitlines():
        match = re.match(r"^\s*\[\s*([+\-?])\s*\]\s+(.+?)\s*$", line)
        if not match:
            continue
        marker, name = match.groups()
        entries[name.strip()] = marker
    return entries


def systemctl_is_usable() -> bool:
    if not shutil.which("systemctl"):
        return False

    result = run_command(["systemctl", "list-units", "--type=service", "--no-pager", "--plain"], timeout=3)
    return result.returncode == 0


def systemctl_service_exists(service_name: str) -> bool:
    unit = f"{service_name}.service"
    result = run_command(["systemctl", "list-unit-files", unit, "--no-pager", "--plain"], timeout=3)
    return result.returncode == 0 and unit in result.output


def detect_service(service_name: str, label: str, service_entries: dict[str, str], has_systemctl: bool) -> Resource | None:
    needs_sudo = current_user_needs_sudo()

    if has_systemctl and systemctl_service_exists(service_name):
        unit = f"{service_name}.service"
        active = run_command(["systemctl", "is-active", unit], timeout=3)
        state = first_line(active.stdout, "unknown")
        details = f"systemd unit: {unit}"
        return Resource(
            id=f"service:{service_name}",
            kind="Service",
            name=label,
            state=state,
            available=True,
            manager="systemctl",
            details=details,
            actions=ACTIONS,
            requires_sudo=needs_sudo,
            target=service_name,
        )

    if service_name not in service_entries:
        return None

    marker = service_entries[service_name]
    status = run_command(["service", service_name, "status"], timeout=5)
    if marker == "+" or status.returncode == 0:
        state = "running"
    elif marker == "-":
        state = "stopped"
    else:
        state = "unknown"

    status_detail = first_line(status.output)
    if "failed to connect to bus" in status_detail.lower():
        details = f"service entry: {service_name}"
    else:
        details = status_detail or f"service entry: {service_name}"
    return Resource(
        id=f"service:{service_name}",
        kind="Service",
        name=label,
        state=state,
        available=True,
        manager="service",
        details=details,
        actions=ACTIONS,
        requires_sudo=needs_sudo,
        target=service_name,
    )


def docker_unavailable_detail(output: str) -> str:
    lower = output.lower()
    if "wsl 2 distro" in lower or "wsl integration" in lower:
        return "Docker CLI found, but Docker Desktop WSL integration is not enabled for this distro."
    if "permission denied" in lower:
        return "Docker CLI found, but the current user cannot access the Docker daemon."
    if "cannot connect" in lower or "is the docker daemon running" in lower:
        return "Docker CLI found, but the Docker daemon is not reachable."
    return first_line(output, "Docker CLI found, but Docker is unavailable.")


def detect_docker_resources() -> list[Resource]:
    docker_bin = shutil.which("docker")
    if not docker_bin:
        return []

    version = run_command(["docker", "version", "--format", "{{.Server.Version}}"], timeout=6)
    if version.returncode != 0:
        return [
            Resource(
                id="docker:daemon",
                kind="Docker",
                name="Docker daemon",
                state="unavailable",
                available=False,
                manager="docker",
                details=docker_unavailable_detail(version.output),
                target="docker",
            )
        ]

    resources = [
        Resource(
            id="docker:daemon",
            kind="Docker",
            name="Docker daemon",
            state="running",
            available=True,
            manager="docker",
            details=f"Docker server {first_line(version.stdout, 'available')}",
            target="docker",
        )
    ]
    resources.extend(detect_docker_containers())
    resources.extend(detect_compose_projects())
    return resources


def docker_state_from_status(status: str) -> str:
    lower = status.lower()
    if lower.startswith("up"):
        return "running"
    if "exited" in lower or "created" in lower:
        return "stopped"
    if lower:
        return lower.split()[0]
    return "unknown"


def safe_id_part(value: str) -> str:
    return re.sub(r"[^A-Za-z0-9_.-]+", "_", value).strip("_") or "unknown"


def detect_docker_containers() -> list[Resource]:
    result = run_command(
        ["docker", "ps", "-a", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}"],
        timeout=8,
    )
    if result.returncode != 0:
        return []

    resources: list[Resource] = []
    for line in result.stdout.splitlines():
        parts = line.split("\t", 3)
        if len(parts) != 4:
            continue
        container_id, name, image, status = [part.strip() for part in parts]
        if not name:
            continue
        resources.append(
            Resource(
                id=f"container:{safe_id_part(name)}",
                kind="Container",
                name=name,
                state=docker_state_from_status(status),
                available=True,
                manager="docker",
                details=f"{image} | {status}",
                actions=ACTIONS,
                target=name or container_id,
            )
        )
    return resources


def compose_command_base() -> list[str] | None:
    if shutil.which("docker"):
        version = run_command(["docker", "compose", "version"], timeout=5)
        if version.returncode == 0:
            return ["docker", "compose"]
    if shutil.which("docker-compose"):
        return ["docker-compose"]
    return None


def detect_compose_projects() -> list[Resource]:
    base = compose_command_base()
    if not base:
        return []

    result = run_command(base + ["ls", "--format", "json"], timeout=8)
    if result.returncode != 0:
        return []

    try:
        projects = json.loads(result.stdout or "[]")
    except json.JSONDecodeError:
        return []

    if isinstance(projects, dict):
        projects = [projects]

    resources: list[Resource] = []
    for project in projects:
        if not isinstance(project, dict):
            continue
        name = str(project.get("Name") or project.get("name") or "").strip()
        if not name:
            continue
        status = str(project.get("Status") or project.get("status") or "unknown").strip()
        config = str(project.get("ConfigFiles") or project.get("configFiles") or "").strip()
        files = tuple(part.strip() for part in re.split(r"[,;]", config) if part.strip())
        resources.append(
            Resource(
                id=f"compose:{safe_id_part(name)}",
                kind="Compose",
                name=name,
                state=status or "unknown",
                available=True,
                manager=" ".join(base),
                details=config or "Docker Compose project",
                actions=ACTIONS,
                target=name,
                compose_files=files,
            )
        )
    return resources


def list_resources() -> list[Resource]:
    service_entries = service_status_entries()
    has_systemctl = systemctl_is_usable()
    resources: list[Resource] = []

    for service_name, label in SERVICE_DEFINITIONS:
        resource = detect_service(service_name, label, service_entries, has_systemctl)
        if resource:
            resources.append(resource)

    service_ids = {resource.id for resource in resources}
    for docker_resource in detect_docker_resources():
        if docker_resource.id == "docker:daemon" and "service:docker" in service_ids:
            continue
        resources.append(docker_resource)

    return resources


def find_resource(resource_id: str) -> Resource | None:
    for resource in list_resources():
        if resource.id == resource_id:
            return resource
    return None


def build_action_command(resource: Resource, action: str, use_sudo: bool = False) -> tuple[str, ...]:
    if action not in ACTIONS:
        return ()

    sudo = ["sudo"] if use_sudo and current_user_needs_sudo() else []

    if resource.kind == "Service":
        if resource.manager == "systemctl":
            return tuple(sudo + ["systemctl", action, f"{resource.target}.service"])
        return tuple(sudo + ["service", resource.target, action])

    if resource.kind == "Container":
        return tuple(["docker", action, resource.target])

    if resource.kind == "Compose":
        base = compose_command_base()
        if not base:
            return ()
        command = [*base]
        for config_file in resource.compose_files:
            command.extend(["-f", config_file])
        command.extend(["-p", resource.target, action])
        return tuple(command)

    return ()


def run_resource_action(resource_id: str, action: str, use_sudo: bool = False) -> ActionResult:
    resource = find_resource(resource_id)
    if not resource:
        return ActionResult(False, f"Resource not found: {resource_id}")

    if action not in resource.actions:
        return ActionResult(False, f"{resource.name} does not support {action}.")

    if not resource.available:
        return ActionResult(False, f"{resource.name} is unavailable: {resource.details}")

    command = build_action_command(resource, action, use_sudo=use_sudo)
    if not command:
        return ActionResult(False, f"No command available to {action} {resource.name}.")

    if resource.requires_sudo and current_user_needs_sudo() and not use_sudo:
        sudo_command = build_action_command(resource, action, use_sudo=True)
        return ActionResult(
            False,
            f"{action.title()} for {resource.name} requires sudo.",
            requires_terminal=True,
            command=sudo_command,
        )

    result = run_command(list(command), timeout=90)
    if result.returncode == 0:
        return ActionResult(True, f"{action.title()} completed: {resource.name}", result.output, command=command)

    output = result.output
    if not use_sudo and "permission denied" in output.lower():
        sudo_command = ("sudo", *command)
        return ActionResult(
            False,
            f"{action.title()} for {resource.name} requires elevated permissions.",
            output,
            requires_terminal=True,
            command=sudo_command,
        )

    return ActionResult(False, f"Failed to {action} {resource.name}.", output, command=command)


def format_shell_command(command: tuple[str, ...] | list[str]) -> str:
    return " ".join(shlex.quote(part) for part in command)


def format_resources_table(resources: list[Resource]) -> str:
    rows = [["KIND", "NAME", "STATE", "MANAGER", "DETAILS"]]
    for resource in resources:
        rows.append([resource.kind, resource.name, resource.state, resource.manager, resource.details])

    widths = [0, 0, 0, 0, 0]
    for row in rows:
        for index, value in enumerate(row):
            widths[index] = max(widths[index], len(value))

    lines = []
    for index, row in enumerate(rows):
        lines.append(
            "  ".join(
                value.ljust(widths[column_index])
                for column_index, value in enumerate(row)
            ).rstrip()
        )
        if index == 0:
            lines.append(
                "  ".join("-" * width for width in widths).rstrip()
            )
    return "\n".join(lines)


def format_fzf_rows(resources: list[Resource]) -> str:
    rows = []
    for resource in resources:
        fields = [
            resource.kind,
            resource.name,
            resource.state,
            resource.manager,
            resource.details,
            resource.id,
        ]
        rows.append(
            "\t".join(field.replace("\t", " ") for field in fields)
        )
    return "\n".join(rows)


def describe_resource(resource: Resource) -> str:
    actions = ", ".join(resource.actions) if resource.actions else "none"
    sudo = "yes" if resource.requires_sudo else "no"
    return "\n".join(
        [
            f"ID: {resource.id}",
            f"Kind: {resource.kind}",
            f"Name: {resource.name}",
            f"State: {resource.state}",
            f"Available: {'yes' if resource.available else 'no'}",
            f"Manager: {resource.manager}",
            f"Actions: {actions}",
            f"Requires sudo: {sudo}",
            f"Details: {resource.details}",
        ]
    )


def cli_list(_args: argparse.Namespace) -> int:
    resources = list_resources()
    if not resources:
        print("No resources detected.")
        return 0
    print(format_resources_table(resources))
    return 0


def cli_rows(_args: argparse.Namespace) -> int:
    print(format_fzf_rows(list_resources()))
    return 0


def cli_details(args: argparse.Namespace) -> int:
    resource = find_resource(args.resource_id)
    if not resource:
        print(f"Resource not found: {args.resource_id}", file=sys.stderr)
        return 1
    print(describe_resource(resource))
    return 0


def cli_action(args: argparse.Namespace) -> int:
    result = run_resource_action(args.resource_id, args.action, use_sudo=args.sudo)
    print(result.message)
    if result.requires_terminal and result.command:
        print(f"Run: {format_shell_command(result.command)}")
    if result.output:
        print(result.output)
    return 0 if result.success else 1


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Internal devv resources helper")
    subparsers = parser.add_subparsers(dest="command", required=True)

    list_parser = subparsers.add_parser("list", help="List resources")
    list_parser.set_defaults(func=cli_list)

    rows_parser = subparsers.add_parser("rows", help="Print fzf rows")
    rows_parser.set_defaults(func=cli_rows)

    details_parser = subparsers.add_parser("details", help="Show resource details")
    details_parser.add_argument("resource_id")
    details_parser.set_defaults(func=cli_details)

    action_parser = subparsers.add_parser("action", help="Run a resource action")
    action_parser.add_argument("--sudo", action="store_true", help="Use sudo for actions that need it")
    action_parser.add_argument("action", choices=ACTIONS)
    action_parser.add_argument("resource_id")
    action_parser.set_defaults(func=cli_action)

    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
