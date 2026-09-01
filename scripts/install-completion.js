#!/usr/bin/env node

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

if (process.env.DVV_SKIP_COMPLETION_INSTALL === "1" || process.platform === "win32") {
  process.exit(0);
}

const root = path.resolve(__dirname, "..");
const source = path.join(root, "completions", "_dvv");

if (!fs.existsSync(source)) {
  process.exit(0);
}

const completionDirs = candidateCompletionDirs();
for (const dir of completionDirs) {
  if (!dir.explicit && !fs.existsSync(dir.path)) {
    continue;
  }

  try {
    fs.mkdirSync(dir.path, { recursive: true });
    const target = path.join(dir.path, "_dvv");
    fs.copyFileSync(source, target);
    removeBrokenLegacyCompletion(dir.path);
    console.error(`dvv zsh completion installed at ${target}`);
    process.exit(0);
  } catch (error) {
    if (process.env.DVV_VERBOSE_SETUP === "1") {
      console.error(`dvv zsh completion skipped for ${dir.path}: ${error.message}`);
    }
  }
}

process.exit(0);

function candidateCompletionDirs() {
  const dirs = [];
  if (process.env.DVV_ZSH_COMPLETION_DIR) {
    dirs.push({ path: path.resolve(expandHome(process.env.DVV_ZSH_COMPLETION_DIR)), explicit: true });
  }

  const home = os.homedir();
  if (home) {
    dirs.push({ path: path.join(home, ".zfunc"), explicit: true });
    const xdgDataHome = process.env.XDG_DATA_HOME || path.join(home, ".local", "share");
    dirs.push({ path: path.join(xdgDataHome, "zsh", "site-functions"), explicit: false });
  }

  return dirs;
}

function expandHome(value) {
  if (value === "~") {
    return os.homedir();
  }
  if (value.startsWith("~/")) {
    return path.join(os.homedir(), value.slice(2));
  }
  return value;
}

function removeBrokenLegacyCompletion(dir) {
  const legacy = path.join(dir, "_devv");
  try {
    const stat = fs.lstatSync(legacy);
    if (!stat.isSymbolicLink()) {
      return;
    }
    const target = fs.readlinkSync(legacy);
    if (target.endsWith(path.join("completions", "_devv")) || target.includes("/completions/_devv")) {
      fs.unlinkSync(legacy);
    }
  } catch {
  }
}
