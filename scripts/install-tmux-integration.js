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

const shortcutsBegin = "# >>> dvv tmux shortcuts >>>";
const shortcutsEnd = "# <<< dvv tmux shortcuts <<<";
const themeBegin = "# >>> dvv tmux theme >>>";
const themeEnd = "# <<< dvv tmux theme <<<";

let content = "";
try {
  content = fs.readFileSync(tmuxConf, "utf8");
} catch (error) {
  if (error.code !== "ENOENT") {
    if (process.env.DVV_VERBOSE_SETUP === "1") {
      console.error(`dvv tmux integration skipped: ${error.message}`);
    }
    process.exit(0);
  }
}

content = removeManagedBlock(content, shortcutsBegin, shortcutsEnd);
content = removeManagedBlock(content, themeBegin, themeEnd);

const blocks = [];
if (resetKey) {
  blocks.push([
    shortcutsBegin,
    "# Managed by dvv. Edit dvv.config.json or set DVV_SKIP_TMUX_INTEGRATION=1 before setup.",
    `unbind-key -n ${resetKey}`,
    `bind-key -n ${resetKey} run-shell -b ${tmuxQuote(resetCommand(root))}`,
    shortcutsEnd,
  ].join("\n"));
}

const themeBlock = tmuxThemeBlock(root);
if (themeBlock) {
  blocks.push(themeBlock);
}

if (blocks.length === 0) {
  writeTmuxConf(tmuxConf, content.trimEnd() ? `${content.trimEnd()}\n` : "");
  sourceTmuxConf(tmuxConf);
  console.error(`dvv tmux integration disabled in ${tmuxConf}`);
  process.exit(0);
}

const prefix = content.trimEnd();
const next = `${prefix ? `${prefix}\n\n` : ""}${blocks.join("\n\n")}\n`;
writeTmuxConf(tmuxConf, next);
sourceTmuxConf(tmuxConf);
console.error(`dvv tmux integration installed in ${tmuxConf}`);

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

function tmuxThemeBlock(root) {
  if (process.env.DVV_SKIP_TMUX_THEME === "1") {
    return "";
  }
  const env = {
    ...process.env,
    DVV_DIR: root,
  };
  if (!env.GOCACHE) {
    env.GOCACHE = path.join(os.tmpdir(), "dvv-go-build-cache");
  }

  const binaryName = process.platform === "win32" ? "dvv.exe" : "dvv";
  const binaryPath = path.join(root, "dist", binaryName);
  let command = binaryPath;
  let args = ["__tmux:theme-block"];
  if (!fs.existsSync(binaryPath)) {
    command = "go";
    args = ["run", "-buildvcs=false", "./cmd/dvv", "__tmux:theme-block"];
  }

  const result = childProcess.spawnSync(command, args, {
    cwd: root,
    env,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
    windowsHide: false,
  });
  if (result.error || (result.status ?? 1) !== 0) {
    if (process.env.DVV_VERBOSE_SETUP === "1") {
      const detail = result.error ? result.error.message : result.stderr || `exit ${result.status ?? 1}`;
      console.error(`dvv tmux theme skipped: ${String(detail).trim()}`);
    }
    return "";
  }
  return String(result.stdout || "").trim();
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
      console.error(`dvv tmux integration skipped: ${error.message}`);
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
