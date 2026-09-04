#!/usr/bin/env node

const childProcess = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const root = path.resolve(__dirname, "..");
const bin = path.join(root, "bin", "dvv");
const tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), "dvv-smoke-"));
const home = path.join(tempRoot, "home");
const xdgConfigHome = path.join(tempRoot, "config");
const workspaceRoot = path.join(tempRoot, "workspace");
const dumpsDir = path.join(tempRoot, "dumps");
const completionDir = path.join(home, ".zfunc");

fs.mkdirSync(home, { recursive: true });
fs.mkdirSync(xdgConfigHome, { recursive: true });
fs.mkdirSync(workspaceRoot, { recursive: true });
fs.mkdirSync(dumpsDir, { recursive: true });
fs.mkdirSync(path.join(home, ".config", "devv"), { recursive: true });
fs.writeFileSync(path.join(home, ".config", "devv", "servers.list"), "local root@127.0.0.1\n", "utf8");

const env = {
  ...process.env,
  HOME: home,
  XDG_CONFIG_HOME: xdgConfigHome,
  DVV_AUTO_BUILD: "0",
  DVV_DIR: root,
  DVV_NO_LOADER: "1",
  DVV_SKIP_TMUX_SOURCE: "1",
  DVV_THEME: "tokyo-night",
  DVV_ZSH_COMPLETION_DIR: completionDir,
  DVV_WORKSPACES_DIR: workspaceRoot,
  DVV_DUMPS_DIR: dumpsDir,
  NO_COLOR: "1",
};

run("build script", process.execPath, [path.join(root, "scripts", "build.js")], { cwd: root });
run("dvv build from another directory", bin, ["build"], { cwd: home });
run("dvv check help", bin, ["check", "help"], { cwd: home });
run("root help", bin, ["help"]);
run("ssh help", bin, ["ssh", "help"]);
run("workspace help", bin, ["workspace", "help"]);
run("tmux help", bin, ["tmux", "help"]);
run("tmux session help", bin, ["tmux:session", "help"]);
run("tmux home help", bin, ["tmux:home", "help"]);
run("db help", bin, ["db", "help"]);
run("resources help", bin, ["resources", "help"]);
run("secrets help", bin, ["secrets", "help"]);
run("config help", bin, ["config", "help"]);
run("bootstrap help", bin, ["bootstrap", "help"]);
run("doctor", bin, ["doctor"]);
run("setup in temporary HOME", bin, ["setup"]);
run("script-friendly ssh list", bin, ["ssh:list"]);
run("script-friendly workspace list", bin, ["workspace:list"]);

assertFile(path.join(completionDir, "_dvv"), "zsh completion");
const zshrc = readFile(path.join(home, ".zshrc"));
assertIncludes(zshrc, "dvv\\n", "managed Alt+G main hub shortcut");
assertIncludes(zshrc, "\\eg", "managed Alt+G sequence");
assertIncludes(zshrc, "dvv workspace\\n", "managed Alt+W workspace shortcut");
assertIncludes(zshrc, "\\ew", "managed Alt+W sequence");
assertIncludes(zshrc, "dvv tmux\\n", "managed Alt+T tmux shortcut");
assertIncludes(zshrc, "\\et", "managed Alt+T sequence");
assertIncludes(zshrc, "dvv tmux:session\\n", "managed Alt+P directory picker shortcut");
assertIncludes(zshrc, "\\ep", "managed Alt+P sequence");
assertIncludes(zshrc, "dvv tmux:home\\n", "managed Alt+F shortcut");
assertIncludes(zshrc, "\\ef", "managed Alt+F sequence");
assertIncludes(zshrc, "dvv tmux:reset-api\\n", "managed Alt+R shell fallback");
assertIncludes(zshrc, "\\er", "managed Alt+R sequence");
assertIncludes(zshrc, "dvv ssh\\n", "managed Alt+S shortcut");
assertExcludes(zshrc, "devv ", "legacy devv shortcut");

const tmuxConf = readFile(path.join(home, ".tmux.conf"));
assertIncludes(tmuxConf, "dvv tmux:reset-api", "managed tmux reset shortcut");
assertIncludes(tmuxConf, "unbind-key -n M-r", "managed Alt+R stale unbind");
assertIncludes(tmuxConf, "bind-key -n M-r", "managed Alt+R tmux sequence");
assertIncludes(tmuxConf, "--fallback-global", "managed Alt+R global fallback");
assertIncludes(tmuxConf, "tmux display-message", "managed Alt+R failure feedback");
assertIncludes(tmuxConf, "tmux-reset.log", "managed Alt+R silent log");
assertIncludes(tmuxConf, "# >>> dvv tmux theme >>>", "managed tmux theme block");
assertIncludes(tmuxConf, "Theme: tokyo-night", "managed tmux theme follows CLI theme");
assertIncludes(tmuxConf, "status-style \"bg=#e0af68,fg=#1a1b26\"", "managed Tokyo Night visible status bar");
assertIncludes(tmuxConf, "window-status-current-style \"bg=#bb9af7,fg=#ffffff,bold\"", "managed Tokyo Night active tmux window style");
assertIncludes(tmuxConf, "pane-active-border-style \"fg=#bb9af7\"", "managed Tokyo Night active pane border");

const completion = readFile(path.join(root, "completions", "_dvv"));
assertIncludes(completion, "DVV_COMPLETE_COMPAT", "compatibility completion gate");
assertIncludes(completion, "ssh:Open the SSH hub", "hub-first SSH completion");

console.error("dvv smoke ok");

function run(label, command, args, options = {}) {
  const result = childProcess.spawnSync(command, args, {
    cwd: options.cwd || root,
    env,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
    windowsHide: false,
  });

  if (result.error) {
    fail(`${label} failed to start: ${result.error.message}`);
  }

  if ((result.status ?? 1) !== 0) {
    if (result.stdout) {
      process.stderr.write(result.stdout);
    }
    if (result.stderr) {
      process.stderr.write(result.stderr);
    }
    fail(`${label} exited with status ${result.status ?? 1}`);
  }
}

function assertFile(file, label) {
  if (!fs.existsSync(file)) {
    fail(`${label} was not created at ${file}`);
  }
}

function readFile(file) {
  try {
    return fs.readFileSync(file, "utf8");
  } catch (error) {
    fail(`could not read ${file}: ${error.message}`);
  }
}

function assertIncludes(value, expected, label) {
  if (!value.includes(expected)) {
    fail(`${label} missing ${JSON.stringify(expected)}`);
  }
}

function assertExcludes(value, unexpected, label) {
  if (value.includes(unexpected)) {
    fail(`${label} contains ${JSON.stringify(unexpected)}`);
  }
}

function fail(message) {
  console.error(`dvv smoke failed: ${message}`);
  process.exit(1);
}
