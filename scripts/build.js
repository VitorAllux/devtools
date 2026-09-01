#!/usr/bin/env node

const childProcess = require("node:child_process");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");

const root = path.resolve(__dirname, "..");
const dist = path.join(root, "dist");
const binaryName = process.platform === "win32" ? "dvv.exe" : "dvv";
const output = path.join(dist, binaryName);

fs.mkdirSync(dist, { recursive: true });

const env = { ...process.env };
if (!env.GOCACHE) {
  env.GOCACHE = path.join(os.tmpdir(), "dvv-go-build-cache");
}

const result = childProcess.spawnSync("go", ["build", "-buildvcs=false", "-o", output, "./cmd/dvv"], {
  cwd: root,
  env,
  stdio: "inherit",
  windowsHide: false,
});

if (result.error) {
  console.error("dvv build failed. Go is required to build the local binary.");
  console.error(result.error.message);
  process.exit(127);
}

if ((result.status ?? 1) !== 0) {
  process.exit(result.status ?? 1);
}

console.error(`dvv built at ${output}`);
process.exit(0);
