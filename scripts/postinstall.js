#!/usr/bin/env node

const childProcess = require("node:child_process");
const path = require("node:path");

if (process.env.DVV_SKIP_POSTINSTALL_BUILD === "1") {
  process.exit(0);
}

const root = path.resolve(__dirname, "..");
const result = childProcess.spawnSync(process.execPath, [path.join(root, "scripts", "build.js")], {
  cwd: root,
  stdio: "inherit",
  windowsHide: false,
});

if ((result.status ?? 1) !== 0) {
  process.exit(result.status ?? 1);
}

console.error("Run `dvv setup` to install zsh completion and shell shortcuts.");
process.exit(0);
