#!/usr/bin/env node

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const childProcess = require("node:child_process");

if (process.env.DVV_SKIP_TMUX_INTEGRATION === "1" || process.platform === "win32") {
  process.exit(0);
}

const root = path.resolve(__dirname, "..");
const home = os.homedir();

if (!home) {
  process.exit(0);
}

const tmuxConf = path.join(home, ".tmux.conf");
const config = readProjectConfig(path.join(root, "dvv.config.json"));
const resetKey = shortcutToTmuxKey(
  process.env.DVV_TMUX_RESET_SHORTCUT ||
    process.env.DEVT_TMUX_RESET_SHORTCUT ||
    config?.tmux?.reset?.shortcut ||
    "alt+r",
);

const begin = "# >>> dvv tmux shortcuts >>>";
const end = "# <<< dvv tmux shortcuts <<<";

let content = "";
try {
  content = fs.readFileSync(tmuxConf, "utf8");
} catch (error) {
  if (error.code !== "ENOENT") {
    if (process.env.DVV_VERBOSE_SETUP === "1") {
      console.error(`dvv tmux shortcuts skipped: ${error.message}`);
    }
    process.exit(0);
  }
}

content = removeManagedBlock(content, begin, end);

if (!resetKey) {
  writeTmuxConf(tmuxConf, content.trimEnd() ? `${content.trimEnd()}\n` : "");
  sourceTmuxConf(tmuxConf);
  console.error(`dvv tmux shortcuts disabled in ${tmuxConf}`);
  process.exit(0);
}

const block = [
  begin,
  "# Managed by dvv. Edit dvv.config.json or set DVV_SKIP_TMUX_INTEGRATION=1 before setup.",
  `unbind-key -n ${resetKey}`,
  `bind-key -n ${resetKey} run-shell -b ${tmuxQuote(resetCommand(root))}`,
  end,
].join("\n");

const next = `${content.trimEnd()}\n\n${block}\n`;
writeTmuxConf(tmuxConf, next);
sourceTmuxConf(tmuxConf);
console.error(`dvv tmux shortcuts installed in ${tmuxConf}`);

function readProjectConfig(file) {
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch {
    return {};
  }
}

function shortcutToTmuxKey(value) {
  const normalized = String(value || "")
    .trim()
    .toLowerCase()
    .replace(/\s+/g, "")
    .replace(/_/g, "-")
    .replace(/\+/g, "-");

  if (!normalized || ["none", "off", "disabled"].includes(normalized)) {
    return "";
  }

  const alt = normalized.match(/^alt-(.)$/);
  if (alt) {
    return `M-${alt[1]}`;
  }

  const ctrl = normalized.match(/^ctrl-(.)$/);
  if (ctrl) {
    return `C-${ctrl[1]}`;
  }

  const shift = normalized.match(/^shift-(.)$/);
  if (shift) {
    return shift[1].toUpperCase();
  }

  if (normalized.length === 1) {
    return normalized;
  }

  return "";
}

function resetCommand(root) {
  const binary = dvvCommand(root);
  const command = `NO_COLOR=1 ${shellWord(binary)} tmux:reset-api --session "#{session_name}" --window "#{window_name}" --fallback-global`;
  return `log_dir="\${XDG_CACHE_HOME:-$HOME/.cache}/devv"; log_file="$log_dir/tmux-reset.log"; mkdir -p "$log_dir"; ${command} >"$log_file" 2>&1; status=$?; if [ "$status" -ne 0 ]; then message="$(tail -n 1 "$log_file" 2>/dev/null)"; [ -n "$message" ] || message="dvv reset failed; see $log_file"; tmux display-message -d 5000 -t "#{session_name}:#{window_name}" "$message"; fi`;
}

function dvvCommand(root) {
  const binaryName = process.platform === "win32" ? "dvv.exe" : "dvv";
  const binaryPath = path.join(root, "dist", binaryName);
  if (fs.existsSync(binaryPath)) {
    return binaryPath;
  }
  return "dvv";
}

function shellWord(value) {
  if (/^[A-Za-z0-9_/:.+-]+$/.test(value)) {
    return value;
  }
  return shellQuote(value);
}

function shellQuote(value) {
  return `'${String(value).replace(/'/g, "'\\''")}'`;
}

function tmuxQuote(value) {
  return shellQuote(value);
}

function removeManagedBlock(content, begin, end) {
  const start = content.indexOf(begin);
  const finish = content.indexOf(end);
  if (start === -1 || finish === -1 || finish < start) {
    return content;
  }
  return content.slice(0, start).trimEnd() + "\n" + content.slice(finish + end.length).trimStart();
}

function writeTmuxConf(file, content) {
  try {
    fs.writeFileSync(file, content, "utf8");
  } catch (error) {
    if (process.env.DVV_VERBOSE_SETUP === "1") {
      console.error(`dvv tmux shortcuts skipped: ${error.message}`);
    }
  }
}

function sourceTmuxConf(file) {
  if (process.env.DVV_SKIP_TMUX_SOURCE === "1") {
    return;
  }
  childProcess.spawnSync("tmux", ["source-file", file], {
    stdio: "ignore",
    windowsHide: false,
  });
}
