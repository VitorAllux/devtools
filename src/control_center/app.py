#!/usr/bin/env python3

import os
import re
import shutil
import subprocess
import sys
from pathlib import Path
from tkinter import END, BOTH, LEFT, RIGHT, VERTICAL, X, Y, messagebox, simpledialog
from tkinter import ttk
import tkinter as tk


class DevvControlCenter:
    def __init__(self, root: tk.Tk) -> None:
        self.root = root
        self.root.title("devv Control Center - Phase 1")
        self.root.geometry("980x680")

        self.devv_bin = os.environ.get("DEVT_DEVV_BIN")
        if not self.devv_bin:
            self.devv_bin = str(Path(__file__).resolve().parents[2] / "bin" / "devv")

        if not Path(self.devv_bin).exists():
            messagebox.showerror("Error", f"devv binary not found: {self.devv_bin}")
            self.root.destroy()
            return

        self._build_ui()
        self.refresh_all()

    def _build_ui(self) -> None:
        style = ttk.Style()
        if "clam" in style.theme_names():
            style.theme_use("clam")

        notebook = ttk.Notebook(self.root)
        notebook.pack(fill=BOTH, expand=True, padx=8, pady=8)

        self.overview_tab = ttk.Frame(notebook)
        self.wsl_tab = ttk.Frame(notebook)
        self.ssh_tab = ttk.Frame(notebook)
        self.config_tab = ttk.Frame(notebook)

        notebook.add(self.overview_tab, text="Overview")
        notebook.add(self.wsl_tab, text="WSL")
        notebook.add(self.ssh_tab, text="SSH")
        notebook.add(self.config_tab, text="Config")

        self._build_overview_tab()
        self._build_wsl_tab()
        self._build_ssh_tab()
        self._build_config_tab()

    def _build_overview_tab(self) -> None:
        top = ttk.Frame(self.overview_tab)
        top.pack(fill=X, padx=8, pady=8)

        self.overview_wsl_label = ttk.Label(top, text="WSL Distros: -")
        self.overview_wsl_label.pack(side=LEFT, padx=(0, 12))

        self.overview_ssh_label = ttk.Label(top, text="SSH Entries: -")
        self.overview_ssh_label.pack(side=LEFT, padx=(0, 12))

        ttk.Button(top, text="Refresh All", command=self.refresh_all).pack(side=RIGHT)

        actions = ttk.LabelFrame(self.overview_tab, text="Quick Actions")
        actions.pack(fill=X, padx=8, pady=(0, 8))

        ttk.Button(actions, text="WSL Status", command=self.wsl_status).pack(side=LEFT, padx=6, pady=6)
        ttk.Button(actions, text="WSL Shutdown", command=self.wsl_shutdown).pack(side=LEFT, padx=6, pady=6)
        ttk.Button(actions, text="List SSH", command=self.refresh_ssh_entries).pack(side=LEFT, padx=6, pady=6)

        logs_frame = ttk.LabelFrame(self.overview_tab, text="Activity")
        logs_frame.pack(fill=BOTH, expand=True, padx=8, pady=(0, 8))

        self.log_text = tk.Text(logs_frame, wrap="word", height=16)
        self.log_text.pack(side=LEFT, fill=BOTH, expand=True)
        scroll = ttk.Scrollbar(logs_frame, orient=VERTICAL, command=self.log_text.yview)
        scroll.pack(side=RIGHT, fill=Y)
        self.log_text.configure(yscrollcommand=scroll.set)
        self.log_text.configure(state="disabled")

    def _build_wsl_tab(self) -> None:
        controls = ttk.Frame(self.wsl_tab)
        controls.pack(fill=X, padx=8, pady=8)

        ttk.Button(controls, text="Refresh", command=self.refresh_wsl_list).pack(side=LEFT, padx=4)
        ttk.Button(controls, text="Start", command=self.wsl_start_selected).pack(side=LEFT, padx=4)
        ttk.Button(controls, text="Stop", command=self.wsl_stop_selected).pack(side=LEFT, padx=4)
        ttk.Button(controls, text="Shutdown All", command=self.wsl_shutdown).pack(side=LEFT, padx=4)
        ttk.Button(controls, text="Status", command=self.wsl_status).pack(side=LEFT, padx=4)

        table_frame = ttk.Frame(self.wsl_tab)
        table_frame.pack(fill=BOTH, expand=True, padx=8, pady=(0, 8))

        cols = ("name", "state", "version")
        self.wsl_tree = ttk.Treeview(table_frame, columns=cols, show="headings", selectmode="browse")
        self.wsl_tree.heading("name", text="Distro")
        self.wsl_tree.heading("state", text="State")
        self.wsl_tree.heading("version", text="Version")
        self.wsl_tree.column("name", width=320)
        self.wsl_tree.column("state", width=120, anchor="center")
        self.wsl_tree.column("version", width=80, anchor="center")
        self.wsl_tree.pack(side=LEFT, fill=BOTH, expand=True)

        scroll = ttk.Scrollbar(table_frame, orient=VERTICAL, command=self.wsl_tree.yview)
        scroll.pack(side=RIGHT, fill=Y)
        self.wsl_tree.configure(yscrollcommand=scroll.set)

    def _build_ssh_tab(self) -> None:
        controls = ttk.Frame(self.ssh_tab)
        controls.pack(fill=X, padx=8, pady=8)

        ttk.Button(controls, text="Refresh", command=self.refresh_ssh_entries).pack(side=LEFT, padx=4)
        ttk.Button(controls, text="Add", command=self.ssh_add_entry).pack(side=LEFT, padx=4)
        ttk.Button(controls, text="Remove", command=self.ssh_remove_selected).pack(side=LEFT, padx=4)
        ttk.Button(controls, text="Connect", command=self.ssh_connect_selected).pack(side=LEFT, padx=4)

        table_frame = ttk.Frame(self.ssh_tab)
        table_frame.pack(fill=BOTH, expand=True, padx=8, pady=(0, 8))

        cols = ("name", "target")
        self.ssh_tree = ttk.Treeview(table_frame, columns=cols, show="headings", selectmode="browse")
        self.ssh_tree.heading("name", text="Name")
        self.ssh_tree.heading("target", text="Target")
        self.ssh_tree.column("name", width=220)
        self.ssh_tree.column("target", width=420)
        self.ssh_tree.pack(side=LEFT, fill=BOTH, expand=True)

        scroll = ttk.Scrollbar(table_frame, orient=VERTICAL, command=self.ssh_tree.yview)
        scroll.pack(side=RIGHT, fill=Y)
        self.ssh_tree.configure(yscrollcommand=scroll.set)

    def _build_config_tab(self) -> None:
        top = ttk.Frame(self.config_tab)
        top.pack(fill=X, padx=8, pady=8)

        ttk.Button(top, text="Refresh", command=self.refresh_config).pack(side=LEFT, padx=4)
        ttk.Button(top, text="Set Key", command=self.set_config_key).pack(side=LEFT, padx=4)

        frame = ttk.LabelFrame(self.config_tab, text="Current Configuration")
        frame.pack(fill=BOTH, expand=True, padx=8, pady=(0, 8))

        self.config_text = tk.Text(frame, wrap="none")
        self.config_text.pack(side=LEFT, fill=BOTH, expand=True)
        scroll = ttk.Scrollbar(frame, orient=VERTICAL, command=self.config_text.yview)
        scroll.pack(side=RIGHT, fill=Y)
        self.config_text.configure(yscrollcommand=scroll.set)
        self.config_text.configure(state="disabled")

    def run_devv(self, args, timeout=25):
        cmd = [self.devv_bin] + args
        proc = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
        output = (proc.stdout or "") + (proc.stderr or "")
        output = output.replace("\x00", "")
        return proc.returncode, output.strip()

    def append_log(self, text: str) -> None:
        self.log_text.configure(state="normal")
        self.log_text.insert(END, text + "\n")
        self.log_text.see(END)
        self.log_text.configure(state="disabled")

    def refresh_all(self) -> None:
        self.refresh_wsl_list()
        self.refresh_ssh_entries()
        self.refresh_config()

    def refresh_wsl_list(self) -> None:
        rc, out = self.run_devv(["wsl:list"])
        self.wsl_tree.delete(*self.wsl_tree.get_children())

        if rc != 0:
            self.append_log("[WSL] Failed to list distros")
            if out:
                self.append_log(out)
            self.overview_wsl_label.configure(text="WSL Distros: error")
            return

        count = 0
        for line in out.splitlines():
            line = line.strip()
            if not line:
                continue
            lower = line.lower()
            if lower.startswith("name") or "windows subsystem" in lower:
                continue

            match = re.match(r"^\*?\s*(.+?)\s{2,}(\S+)\s+(\d+)\s*$", line)
            if not match:
                continue

            name, state, version = match.groups()
            self.wsl_tree.insert("", END, values=(name, state, version))
            count += 1

        self.overview_wsl_label.configure(text=f"WSL Distros: {count}")
        self.append_log("[WSL] Distros refreshed")

    def selected_wsl_distro(self):
        selected = self.wsl_tree.focus()
        if not selected:
            messagebox.showwarning("WSL", "Select a distro first.")
            return None
        values = self.wsl_tree.item(selected, "values")
        if not values:
            return None
        return values[0]

    def wsl_start_selected(self) -> None:
        distro = self.selected_wsl_distro()
        if not distro:
            return
        rc, out = self.run_devv(["wsl:start", distro])
        if rc == 0:
            self.append_log(f"[WSL] Started: {distro}")
        else:
            self.append_log(f"[WSL] Failed to start {distro}")
            if out:
                self.append_log(out)
        self.refresh_wsl_list()

    def wsl_stop_selected(self) -> None:
        distro = self.selected_wsl_distro()
        if not distro:
            return
        rc, out = self.run_devv(["wsl:stop", distro])
        if rc == 0:
            self.append_log(f"[WSL] Stopped: {distro}")
        else:
            self.append_log(f"[WSL] Failed to stop {distro}")
            if out:
                self.append_log(out)
        self.refresh_wsl_list()

    def wsl_shutdown(self) -> None:
        if not messagebox.askyesno("WSL", "Shutdown all WSL distros?"):
            return
        rc, out = self.run_devv(["wsl:shutdown"])
        if rc == 0:
            self.append_log("[WSL] Shutdown requested")
        else:
            self.append_log("[WSL] Failed to shutdown")
            if out:
                self.append_log(out)
        self.refresh_wsl_list()

    def wsl_status(self) -> None:
        rc, out = self.run_devv(["wsl:status"])
        if rc == 0:
            self.append_log("[WSL] Status")
            if out:
                self.append_log(out)
        else:
            self.append_log("[WSL] Failed to get status")
            if out:
                self.append_log(out)

    def refresh_ssh_entries(self) -> None:
        rc, out = self.run_devv(["ssh:list"])
        self.ssh_tree.delete(*self.ssh_tree.get_children())

        if rc != 0:
            self.append_log("[SSH] Failed to list entries")
            if out:
                self.append_log(out)
            self.overview_ssh_label.configure(text="SSH Entries: error")
            return

        count = 0
        for line in out.splitlines():
            clean = line.strip()
            if not clean:
                continue
            parts = clean.split(None, 1)
            if len(parts) < 2:
                continue
            name, target = parts[0], parts[1]
            self.ssh_tree.insert("", END, values=(name, target))
            count += 1

        self.overview_ssh_label.configure(text=f"SSH Entries: {count}")
        self.append_log("[SSH] Entries refreshed")

    def selected_ssh_entry(self):
        selected = self.ssh_tree.focus()
        if not selected:
            messagebox.showwarning("SSH", "Select an SSH entry first.")
            return None
        values = self.ssh_tree.item(selected, "values")
        if not values:
            return None
        return values[0], values[1]

    def ssh_add_entry(self) -> None:
        name = simpledialog.askstring("Add SSH", "Server name (ex: production-web):")
        if not name:
            return

        target = simpledialog.askstring("Add SSH", "Target (ex: root@192.168.1.10):")
        if not target:
            return

        rc, out = self.run_devv(["ssh:add", "--name", name.strip(), "--conn", target.strip()])
        if rc == 0:
            self.append_log(f"[SSH] Added: {name}")
        else:
            self.append_log(f"[SSH] Failed to add: {name}")
            if out:
                self.append_log(out)
        self.refresh_ssh_entries()

    def ssh_remove_selected(self) -> None:
        selected = self.selected_ssh_entry()
        if not selected:
            return

        name, _target = selected
        if not messagebox.askyesno("SSH", f"Remove '{name}'?"):
            return

        rc, out = self.run_devv(["ssh:remove", "--name", name])
        if rc == 0:
            self.append_log(f"[SSH] Removed: {name}")
        else:
            self.append_log(f"[SSH] Failed to remove: {name}")
            if out:
                self.append_log(out)
        self.refresh_ssh_entries()

    def ssh_connect_selected(self) -> None:
        selected = self.selected_ssh_entry()
        if not selected:
            return

        _name, target = selected
        if self._open_terminal_for_ssh(target):
            self.append_log(f"[SSH] Opening terminal for {target}")
            return

        self.root.clipboard_clear()
        self.root.clipboard_append(f"ssh {target}")
        self.root.update()
        messagebox.showinfo(
            "SSH",
            "Could not auto-open a terminal.\n"
            f"Command copied to clipboard:\nssh {target}",
        )
        self.append_log(f"[SSH] Could not launch terminal automatically for {target}")

    def _open_terminal_for_ssh(self, target: str) -> bool:
        terminal_variants = []

        if shutil.which("x-terminal-emulator"):
            terminal_variants.append(["x-terminal-emulator", "-e", "ssh", target])
        if shutil.which("gnome-terminal"):
            terminal_variants.append(["gnome-terminal", "--", "ssh", target])
        if shutil.which("konsole"):
            terminal_variants.append(["konsole", "-e", "ssh", target])
        if shutil.which("xfce4-terminal"):
            terminal_variants.append(["xfce4-terminal", "-e", f"ssh {target}"])
        if shutil.which("wt.exe"):
            terminal_variants.append(["wt.exe", "wsl.exe", "-e", "ssh", target])

        for cmd in terminal_variants:
            try:
                subprocess.Popen(cmd)
                return True
            except OSError:
                continue

        return False

    def refresh_config(self) -> None:
        rc, out = self.run_devv(["config:list"])
        self.config_text.configure(state="normal")
        self.config_text.delete("1.0", END)
        if rc == 0:
            self.config_text.insert(END, out + "\n")
        else:
            self.config_text.insert(END, out + "\n")
            self.append_log("[Config] Failed to load config")
        self.config_text.configure(state="disabled")

    def set_config_key(self) -> None:
        key = simpledialog.askstring("Config", "Key:")
        if not key:
            return
        value = simpledialog.askstring("Config", "Value:")
        if value is None:
            return

        rc, out = self.run_devv(["config:set", key.strip(), value])
        if rc == 0:
            self.append_log(f"[Config] Set {key}")
            self.refresh_config()
        else:
            self.append_log(f"[Config] Failed to set {key}")
            if out:
                self.append_log(out)


def main() -> int:
    root = tk.Tk()
    app = DevvControlCenter(root)
    if not app.root.winfo_exists():
        return 1
    root.mainloop()
    return 0


if __name__ == "__main__":
    sys.exit(main())
