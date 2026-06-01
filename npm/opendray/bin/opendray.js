#!/usr/bin/env node
// Resolves the matching `opendray-<os>-<arch>` package installed via
// npm's optionalDependencies (selected by `os`/`cpu` in each package's
// own package.json) and execs the bundled Go binary. No network call,
// no postinstall — works under `--ignore-scripts`.

const { spawnSync } = require("node:child_process");
const path = require("node:path");
const fs = require("node:fs");

const PLATFORM_MAP = {
  "darwin-arm64": "opendray-darwin-arm64",
  "darwin-x64": "opendray-darwin-x64",
  "linux-arm64": "opendray-linux-arm64",
  "linux-x64": "opendray-linux-x64",
};

const key = `${process.platform}-${process.arch}`;
const pkgName = PLATFORM_MAP[key];

if (!pkgName) {
  console.error(
    `opendray: unsupported platform ${key}.\n` +
      `Supported: ${Object.keys(PLATFORM_MAP).join(", ")}.\n` +
      `See https://github.com/Opendray/opendray/releases for source builds.`,
  );
  process.exit(1);
}

let binaryPath;
try {
  binaryPath = require.resolve(`${pkgName}/bin/opendray`);
} catch {
  console.error(
    `opendray: the matching platform package "${pkgName}" was not installed.\n` +
      `This usually means npm was run with --no-optional, or the install was\n` +
      `interrupted. Re-run:  npm install -g opendray`,
  );
  process.exit(1);
}

try {
  fs.chmodSync(binaryPath, 0o755);
} catch {
  // best-effort — the tarball already has exec bits; chmod may fail on
  // read-only filesystems but that's the user's responsibility there.
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
  shell: false,
});

if (result.error) {
  console.error(`opendray: failed to exec ${binaryPath}: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status ?? 0);
