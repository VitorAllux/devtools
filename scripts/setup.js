#!/usr/bin/env node

const childProcess = require("node:child_process");
const path = require("node:path");

const root = path.resolve(__dirname, "..");

run("completion", "install-completion.js");
run("shell integration", "install-shell-integration.js");

console.error("dvv setup complete.");

function run(label, script) {
  const result = childProcess.spawnSync(process.execPath, [path.join(root, "scripts", script)], {
    cwd: root,
    stdio: "inherit",
    windowsHide: false,
  });

  if (result.error) {
    console.error(`dvv ${label} setup failed.`);
    console.error(result.error.message);
    process.exit(127);
  }

  if ((result.status ?? 1) !== 0) {
    process.exit(result.status ?? 1);
  }
}
