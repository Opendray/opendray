# npm distribution

This directory contains the npm packages that mirror the official GoReleaser
output. Users get them via `npm install -g opendray`.

## Layout

| Package                 | Purpose                                                  |
| ----------------------- | -------------------------------------------------------- |
| `opendray/`             | The package users install. Tiny Node shim in `bin/opendray.js` that resolves to the matching platform package via `optionalDependencies`. |
| `opendray-linux-x64/`   | Linux amd64 binary (`os: linux`, `cpu: x64`).            |
| `opendray-linux-arm64/` | Linux arm64 binary (`os: linux`, `cpu: arm64`).          |
| `opendray-darwin-x64/`  | macOS x64 binary (`os: darwin`, `cpu: x64`).             |
| `opendray-darwin-arm64/`| macOS arm64 binary (`os: darwin`, `cpu: arm64`).         |

The pattern is the same one used by esbuild, Biome, swc, and Bun: no
`postinstall` script, no network call at install time. npm itself picks
the right platform package using the `os`/`cpu` constraints in each
`package.json`, and the main shim does `require.resolve(...)` to find it.

## Publishing

[`scripts/publish-npm.mjs`](../scripts/publish-npm.mjs) is invoked by the
`publish-npm` job in `.github/workflows/release.yml` after GoReleaser
finishes. It downloads each platform tarball from the GitHub release,
verifies the SHA-256 against `SHA256SUMS`, drops the binary into the
matching package, then publishes all five packages with
`npm publish --provenance --access public`.

### One-time bootstrap

Before the first run, an operator needs to:

1. Create an npm account (or use an existing one) and reserve the
   following package names by publishing version `0.0.0` from a clean
   checkout of this directory:
   - `opendray`
   - `opendray-linux-x64`
   - `opendray-linux-arm64`
   - `opendray-darwin-x64`
   - `opendray-darwin-arm64`
2. Create an npm automation token scoped to those five packages.
3. Add the token to GitHub Actions secrets as `NPM_TOKEN` on the
   `Opendray/opendray` repository.

After that, every tag push (`v*`) automatically publishes the matching
npm version. Pre-release tags containing `-` (e.g. `v2.5.0-rc.1`) are
skipped by the workflow.

## Manual smoke test

```sh
cd npm/opendray
npm pack --dry-run
```

Lists what `npm publish` would ship. Use it to verify the file set
before cutting a release.
