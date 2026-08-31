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
const config = readProjectConfig(path.join(root, "dvv.config.json"));
const tmuxShortcut = shortcutToZshSequences(
  process.env.DVV_TMUX_SESSION_SHORTCUT ||
    process.env.DEVT_TMUX_SESSION_SHORTCUT ||
    config?.tmux?.session?.shortcut ||
    "ctrl+f",
);
const tmuxHomeShortcut = shortcutToZshSequences(
  process.env.DVV_TMUX_HOME_SHORTCUT ||
    process.env.DEVT_TMUX_HOME_SHORTCUT ||
    config?.tmux?.home?.shortcut ||
    "ctrl+shift+f",
);

const bindings = [];
addBindings(bindings, tmuxShortcut, "dvv tmux:session");
addBindings(bindings, tmuxHomeShortcut, "dvv tmux:home");
bindings.push(`bindkey -s "\\es" "dvv ssh\\n"`);

if (bindings.length === 0) {
  process.exit(0);
}

const begin = "# >>> dvv shell shortcuts >>>";
const end = "# <<< dvv shell shortcuts <<<";
const block = [
  begin,
  "# Managed by dvv. Edit dvv.config.json or set DVV_SKIP_SHELL_INTEGRATION=1 before install.",
  `[[ -d "$HOME/.zfunc" && ":${"$"}{fpath[*]}:" != *":$HOME/.zfunc:"* ]] && fpath=("$HOME/.zfunc" $fpath)`,
  ...bindings,
  end,
].join("\n");

let content = "";
try {
  content = fs.readFileSync(zshrc, "utf8");
} catch (error) {
  if (error.code !== "ENOENT") {
    if (process.env.DVV_VERBOSE_POSTINSTALL === "1") {
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
  if (process.env.DVV_VERBOSE_POSTINSTALL === "1") {
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

function addBindings(bindings, sequences, command) {
  for (const sequence of sequences) {
    bindings.push(`bindkey -s "${sequence}" "${command}\\n"`);
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
    'bindkey -s "\\es" "devv ssh\\n"',
    'bindkey -s "\\es" "dvv ssh\\n"',
    'bindkey -s "\\es" "devv ssh:connect\\n"',
  ].includes(value);
}
