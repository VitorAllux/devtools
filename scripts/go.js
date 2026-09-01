#!/usr/bin/env node

const childProcess = require("node:child_process");
const os = require("node:os");
const path = require("node:path");

const root = path.resolve(__dirname, "..");
const args = process.argv.slice(2);

if (args.length === 0) {
  console.error("Usage: node ./scripts/go.js <go-args>");
  process.exit(2);
}

const env = { ...process.env };
if (!env.GOCACHE) {
  env.GOCACHE = path.join(os.tmpdir(), "dvv-go-build-cache");
}

const result = childProcess.spawnSync("go", args, {
  cwd: root,
  env,
  stdio: "inherit",
  windowsHide: false,
});

if (result.error) {
  console.error("dvv Go command failed. Go is required for local development checks.");
  console.error(result.error.message);
  process.exit(127);
}

process.exit(result.status ?? 1);
