#!/usr/bin/env node

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

if (process.env.DVV_SKIP_SHELL_INTEGRATION === "1" || process.platform === "win32") {
  process.exit(0);
}

const root = path.resolve(__dirname, "..");
const home = os.homedir();

if (!home) {
  process.exit(0);
}

const zshrc = path.join(home, ".zshrc");
const configHome = process.env.XDG_CONFIG_HOME || path.join(home, ".config");
const shellShortcutsFile = path.join(configHome, "devv", "shell-shortcuts.zsh");
const config = readProjectConfig(path.join(root, "dvv.config.json"));
const mainHubShortcut = shortcutToZshSequences(
  shortcutValue("DVV_SHELL_MAIN_SHORTCUT", undefined, config?.shell?.shortcuts?.mainHub, "alt+g"),
);
const workspaceShortcut = shortcutToZshSequences(
  shortcutValue("DVV_SHELL_WORKSPACE_SHORTCUT", undefined, config?.shell?.shortcuts?.workspace, "alt+w"),
);
const shellTmuxShortcut = shortcutToZshSequences(
  shortcutValue("DVV_SHELL_TMUX_SHORTCUT", undefined, config?.shell?.shortcuts?.tmux, "alt+t"),
);
const sshShortcut = shortcutToZshSequences(
  shortcutValue("DVV_SHELL_SSH_SHORTCUT", undefined, config?.shell?.shortcuts?.ssh, "alt+s"),
);
const tmuxShortcut = shortcutToZshSequences(
  shortcutValue("DVV_TMUX_SESSION_SHORTCUT", "DEVT_TMUX_SESSION_SHORTCUT", config?.tmux?.session?.shortcut, "alt+p"),
);
const tmuxHomeShortcut = shortcutToZshSequences(
  shortcutValue("DVV_TMUX_HOME_SHORTCUT", "DEVT_TMUX_HOME_SHORTCUT", config?.tmux?.home?.shortcut, "alt+f"),
);
const tmuxResetShortcut = shortcutToZshSequences(
  shortcutValue("DVV_TMUX_RESET_SHORTCUT", "DEVT_TMUX_RESET_SHORTCUT", config?.tmux?.reset?.shortcut, "alt+r"),
);

const bindings = [];
addBindings(bindings, mainHubShortcut, "dvv");
addBindings(bindings, workspaceShortcut, "dvv workspace");
addBindings(bindings, shellTmuxShortcut, "dvv tmux");
addBindings(bindings, sshShortcut, "dvv ssh");
addBindings(bindings, tmuxShortcut, "dvv tmux:session");
addBindings(bindings, tmuxHomeShortcut, "dvv tmux:home");
addBindings(bindings, tmuxResetShortcut, "dvv tmux:reset-api");

if (bindings.length === 0) {
  process.exit(0);
}

const begin = "# >>> dvv shell shortcuts >>>";
const end = "# <<< dvv shell shortcuts <<<";
writeShellShortcutsFile(shellShortcutsFile, bindings);
const block = [
  begin,
  "# Managed by dvv. Edit dvv.config.json or set DVV_SKIP_SHELL_INTEGRATION=1 before setup.",
  `__dvv_reload_shell_shortcuts() {`,
  `  local file="\${XDG_CONFIG_HOME:-$HOME/.config}/devv/shell-shortcuts.zsh"`,
  `  [[ -r "$file" ]] && source "$file"`,
  `}`,
  `__dvv_reload_shell_shortcuts`,
  `[[ -d "$HOME/.zfunc" && ":${"$"}{fpath[*]}:" != *":$HOME/.zfunc:"* ]] && fpath=("$HOME/.zfunc" $fpath)`,
  ...bindings,
  `dvv() {`,
  `  command dvv "$@"`,
  `  local __dvv_status=$?`,
  `  if [[ $__dvv_status -eq 0 ]]; then`,
  `    case "$1" in`,
  `      config|setup) __dvv_reload_shell_shortcuts ;;`,
  `    esac`,
  `  fi`,
  `  return $__dvv_status`,
  `}`,
  end,
].join("\n");

let content = "";
try {
  content = fs.readFileSync(zshrc, "utf8");
} catch (error) {
  if (error.code !== "ENOENT") {
    if (process.env.DVV_VERBOSE_SETUP === "1") {
      console.error(`dvv shell shortcuts skipped: ${error.message}`);
    }
    process.exit(0);
  }
}

content = removeManagedBlock(content, begin, end);
content = removeLegacyShortcutLines(content);
const next = `${content.trimEnd()}\n\n${block}\n`;

try {
  fs.writeFileSync(zshrc, next, "utf8");
  console.error(`dvv zsh shortcuts installed in ${zshrc}`);
} catch (error) {
  if (process.env.DVV_VERBOSE_SETUP === "1") {
    console.error(`dvv shell shortcuts skipped: ${error.message}`);
  }
}

function readProjectConfig(file) {
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch {
    return {};
  }
}

function shortcutValue(primaryEnv, legacyEnv, configured, fallback) {
  if (process.env[primaryEnv]) {
    return process.env[primaryEnv];
  }
  if (legacyEnv && process.env[legacyEnv]) {
    return process.env[legacyEnv];
  }
  return configured || fallback;
}

function addBindings(bindings, sequences, command) {
  for (const sequence of sequences) {
    bindings.push(`bindkey -s "${sequence}" "${command}\\n"`);
  }
}

function writeShellShortcutsFile(file, bindings) {
  const sequences = bindings
    .map((binding) => binding.match(/^bindkey -s "(.+)" "/)?.[1])
    .filter(Boolean);
  const lines = [
    "# Managed by dvv. Source this file from ~/.zshrc; do not edit.",
    `if (( \${+__dvv_managed_shortcut_sequences} )); then`,
    `  for __dvv_sequence in "\${__dvv_managed_shortcut_sequences[@]}"; do`,
    `    bindkey -r "$__dvv_sequence" 2>/dev/null || true`,
    `  done`,
    `fi`,
    `typeset -ga __dvv_managed_shortcut_sequences`,
    `__dvv_managed_shortcut_sequences=(${sequences.map((sequence) => `"${sequence}"`).join(" ")})`,
    `unset __dvv_sequence`,
    `[[ -d "$HOME/.zfunc" && ":${"$"}{fpath[*]}:" != *":$HOME/.zfunc:"* ]] && fpath=("$HOME/.zfunc" $fpath)`,
    ...bindings,
    "",
  ];
  try {
    fs.mkdirSync(path.dirname(file), { recursive: true, mode: 0o700 });
    fs.writeFileSync(file, lines.join("\n"), { encoding: "utf8", mode: 0o600 });
  } catch (error) {
    if (process.env.DVV_VERBOSE_SETUP === "1") {
      console.error(`dvv shell shortcut source skipped: ${error.message}`);
    }
  }
}

function shortcutToZshSequences(value) {
  const normalized = String(value || "")
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "")
    .replace(/-/g, "+");

  if (!normalized || ["none", "off", "disabled"].includes(normalized)) {
    return [];
  }

  const ctrlShift = normalized.match(/^ctrl\+shift\+(.+)$/);
  if (ctrlShift && ctrlShift[1].length === 1) {
    const codepoint = ctrlShift[1].toUpperCase().codePointAt(0);
    return [`\\e[${codepoint};6u`];
  }

  const ctrl = normalized.match(/^ctrl\+(.+)$/);
  if (ctrl && ctrl[1].length === 1) {
    return [`^${ctrl[1].toUpperCase()}`];
  }

  const alt = normalized.match(/^alt\+(.+)$/);
  if (alt && alt[1].length === 1) {
    return [`\\e${alt[1]}`];
  }

  return [];
}

function removeManagedBlock(content, begin, end) {
  const start = content.indexOf(begin);
  const finish = content.indexOf(end);
  if (start === -1 || finish === -1 || finish < start) {
    return content;
  }
  return content.slice(0, start).trimEnd() + "\n" + content.slice(finish + end.length).trimStart();
}

function removeLegacyShortcutLines(content) {
  return content
    .split("\n")
    .filter((line) => !isLegacyShortcutLine(line))
    .join("\n");
}

function isLegacyShortcutLine(line) {
  const value = line.trim();
  return [
    'bindkey -s "^F" "devv tmux:session\\n"',
    'bindkey -s "^F" "dvv tmux:session\\n"',
    'bindkey -s "\\e[70;6u" "dvv tmux:home\\n"',
    'bindkey -s "\\eg" "dvv\\n"',
    'bindkey -s "\\ew" "dvv workspace\\n"',
    'bindkey -s "\\et" "dvv tmux\\n"',
    'bindkey -s "\\ep" "dvv tmux:session\\n"',
    'bindkey -s "\\ef" "dvv tmux:home\\n"',
    'bindkey -s "\\er" "dvv tmux:reset-api\\n"',
    'bindkey -s "\\es" "devv ssh\\n"',
    'bindkey -s "\\es" "dvv ssh\\n"',
    'bindkey -s "\\es" "devv ssh:connect\\n"',
  ].includes(value);
}
