#!/usr/bin/env node
// Publishes the npm release for a tagged version. Called by the
// Release workflow after GoReleaser has uploaded the platform tarballs
// to the GitHub release.
//
// For each of the four supported platforms it:
//   1. downloads opendray_<VERSION>_<os>_<arch>.tar.gz from the release,
//   2. verifies the SHA-256 against SHA256SUMS (also from the release),
//   3. extracts the `opendray` binary into npm/opendray-<os>-<arch>/bin/,
//   4. bumps the matching package.json version to <VERSION>,
//   5. runs `npm publish --provenance --access public`.
//
// Once all four platform packages are published it bumps the main
// `opendray` package (matching `optionalDependencies` to the new version)
// and publishes it. Order matters: the main package's
// optionalDependencies must resolve at install time.
//
// Finally builds and publishes @opendray/sdk at the same version, so
// the TS SDK always ships in lockstep with the gateway.
//
// Required env:
//   NODE_AUTH_TOKEN  — npm automation token with publish for `opendray`
//                      and `opendray-{linux,darwin}-{x64,arm64}`.
//   GITHUB_TOKEN     — used to call the GitHub API (anonymous works for
//                      public repos but rate limits aggressively).
// Required arg:
//   --tag vX.Y.Z

import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { existsSync, mkdirSync, readFileSync, writeFileSync, chmodSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = resolve(__dirname, "..");
const NPM_ROOT = join(REPO_ROOT, "npm");

const PLATFORMS = [
  { pkg: "opendray-linux-x64", os: "linux", arch: "amd64" },
  { pkg: "opendray-linux-arm64", os: "linux", arch: "arm64" },
  { pkg: "opendray-darwin-x64", os: "darwin", arch: "amd64" },
  { pkg: "opendray-darwin-arm64", os: "darwin", arch: "arm64" },
];

function die(msg) {
  console.error(`publish-npm: ${msg}`);
  process.exit(1);
}

function parseArgs() {
  const tagFlag = process.argv.indexOf("--tag");
  if (tagFlag === -1 || !process.argv[tagFlag + 1]) {
    die("missing --tag vX.Y.Z");
  }
  const tag = process.argv[tagFlag + 1];
  if (!/^v\d+\.\d+\.\d+/.test(tag)) {
    die(`tag must look like vX.Y.Z, got: ${tag}`);
  }
  return { tag, version: tag.replace(/^v/, "") };
}

async function fetchBuffer(url, token) {
  const headers = { "User-Agent": "opendray-publish-npm" };
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(url, { headers, redirect: "follow" });
  if (!res.ok) die(`fetch ${url} -> ${res.status} ${res.statusText}`);
  return Buffer.from(await res.arrayBuffer());
}

function sha256(buf) {
  return createHash("sha256").update(buf).digest("hex");
}

function parseSums(sumsText) {
  const map = new Map();
  for (const line of sumsText.split(/\r?\n/)) {
    const m = line.match(/^([0-9a-f]{64})\s+\*?(.+)$/i);
    if (m) map.set(m[2].trim(), m[1].toLowerCase());
  }
  return map;
}

function run(cmd, args, opts = {}) {
  const r = spawnSync(cmd, args, { stdio: "inherit", ...opts });
  if (r.status !== 0) die(`${cmd} ${args.join(" ")} -> exit ${r.status}`);
}

function captureRun(cmd, args, opts = {}) {
  const r = spawnSync(cmd, args, { encoding: "utf-8", ...opts });
  if (r.status !== 0) die(`${cmd} ${args.join(" ")} -> exit ${r.status}\n${r.stderr}`);
  return r.stdout;
}

function setPkgVersion(pkgDir, version, extra = {}) {
  const file = join(pkgDir, "package.json");
  const pkg = JSON.parse(readFileSync(file, "utf-8"));
  pkg.version = version;
  Object.assign(pkg, extra);
  writeFileSync(file, JSON.stringify(pkg, null, 2) + "\n");
}

function copyLicenseInto(pkgDir) {
  const src = join(REPO_ROOT, "LICENSE");
  if (!existsSync(src)) die("repo LICENSE missing — refusing to publish");
  writeFileSync(join(pkgDir, "LICENSE"), readFileSync(src));
}

async function preparePlatform(platform, tag, version, sums, ghToken) {
  const { pkg, os, arch } = platform;
  const tarballName = `opendray_${version}_${os}_${arch}.tar.gz`;
  const tarballUrl = `https://github.com/Opendray/opendray/releases/download/${tag}/${tarballName}`;
  console.log(`\n=== ${pkg}: ${tarballName} ===`);

  const buf = await fetchBuffer(tarballUrl, ghToken);
  const got = sha256(buf);
  const want = sums.get(tarballName);
  if (!want) die(`${tarballName} not present in SHA256SUMS`);
  if (got !== want) die(`SHA mismatch for ${tarballName}: got ${got} want ${want}`);
  console.log(`  sha256 verified (${got})`);

  const tmp = join(tmpdir(), `opendray-publish-${pkg}-${process.pid}`);
  mkdirSync(tmp, { recursive: true });
  const tarPath = join(tmp, tarballName);
  writeFileSync(tarPath, buf);
  run("tar", ["-xzf", tarPath, "-C", tmp]);

  const pkgDir = join(NPM_ROOT, pkg);
  const binDest = join(pkgDir, "bin", "opendray");
  const extractedBinary = join(tmp, "opendray");
  if (!existsSync(extractedBinary)) {
    die(`extracted tarball missing opendray binary at ${extractedBinary}`);
  }
  writeFileSync(binDest, readFileSync(extractedBinary));
  chmodSync(binDest, 0o755);

  setPkgVersion(pkgDir, version);
  copyLicenseInto(pkgDir);
  console.log(`  staged ${pkgDir} at ${version}`);
  return pkgDir;
}

function npmPublish(pkgDir) {
  console.log(`\n=== publishing ${pkgDir} ===`);
  run("npm", ["publish", "--provenance", "--access", "public"], { cwd: pkgDir });
}

function prepareSdk(version) {
  const sdkDir = join(NPM_ROOT, "sdk");
  console.log(`\n=== @opendray/sdk: building ===`);
  run("npm", ["install", "--no-audit", "--no-fund"], { cwd: sdkDir });
  run("npm", ["run", "build"], { cwd: sdkDir });
  setPkgVersion(sdkDir, version);
  copyLicenseInto(sdkDir);
  console.log(`  staged ${sdkDir} at ${version}`);
  return sdkDir;
}

async function main() {
  const { tag, version } = parseArgs();
  const ghToken = process.env.GITHUB_TOKEN || "";

  if (!process.env.NODE_AUTH_TOKEN) {
    console.log("publish-npm: SKIPPED — NODE_AUTH_TOKEN is empty.");
    console.log("");
    console.log("The NPM_TOKEN repository secret is not configured, so the");
    console.log("npm distribution channel is skipped for this release. The");
    console.log("GitHub release tarballs are unaffected.");
    console.log("");
    console.log("To enable npm publishing:");
    console.log("  1. Generate an npm Automation token at");
    console.log("     https://www.npmjs.com/settings/~/tokens");
    console.log("  2. Add it as the NPM_TOKEN repository secret at");
    console.log("     https://github.com/Opendray/opendray/settings/secrets/actions/new");
    console.log("  3. Re-run this workflow (workflow_dispatch against the");
    console.log(`     existing tag ${tag}) — npm publish will pick up the secret`);
    console.log("     on the next attempt.");
    console.log("");
    console.log("See RELEASING.md for details.");
    process.exit(0);
  }

  console.log(`Publishing opendray @ ${version} (tag ${tag})`);

  const sumsBuf = await fetchBuffer(
    `https://github.com/Opendray/opendray/releases/download/${tag}/SHA256SUMS`,
    ghToken,
  );
  const sums = parseSums(sumsBuf.toString("utf-8"));
  if (sums.size === 0) die("SHA256SUMS empty / unparseable");

  const platformDirs = [];
  for (const platform of PLATFORMS) {
    platformDirs.push(await preparePlatform(platform, tag, version, sums, ghToken));
  }

  for (const pkgDir of platformDirs) {
    npmPublish(pkgDir);
  }

  const mainDir = join(NPM_ROOT, "opendray");
  const optDeps = {};
  for (const { pkg } of PLATFORMS) optDeps[pkg] = version;
  setPkgVersion(mainDir, version, { optionalDependencies: optDeps });
  copyLicenseInto(mainDir);
  npmPublish(mainDir);

  const sdkDir = prepareSdk(version);
  npmPublish(sdkDir);

  console.log(`\nopendray @ ${version} + @opendray/sdk @ ${version} published.`);
  console.log(`Verify: npm view opendray version && npm view @opendray/sdk version`);
}

main().catch((err) => die(err.stack || err.message));
