# 09 — Marketplace

The marketplace is a **Git repository** (`github.com/opendray/marketplace`) that anyone can fork, and whose `main` branch is the canonical registry. Plugin artifacts are hosted on the publisher's infrastructure (GitHub Releases, a CDN, etc.); the registry only holds metadata + integrity hashes.

> **Locked:** Git-repo-as-registry with PR-based approval. No central server to run, no vendor lock-in, audit trail free.

## Repo layout

```
opendray/marketplace/
  index.json                         # root registry — stable URL
  plugins/
    acme/
      hello/
        meta.json                    # publisher + description + latest version
        1.0.0.json                   # per-version metadata (hash, url, manifest)
        1.1.0.json
  publishers/
    acme.json                        # signed publisher record
  revocations.json                   # killswitch entries
  CODEOWNERS                         # review routing
```

## `index.json` (root) schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "OpenDray Marketplace Index",
  "type": "object",
  "required": ["version", "generatedAt", "plugins"],
  "properties": {
    "version":     { "const": 1 },
    "generatedAt": { "type": "string", "format": "date-time" },
    "plugins": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "publisher", "latest", "path"],
        "properties": {
          "name":        { "type": "string" },
          "publisher":   { "type": "string" },
          "displayName": { "type": "string" },
          "description": { "type": "string", "maxLength": 280 },
          "icon":        { "type": "string" },
          "categories":  { "type": "array", "items": { "type": "string" } },
          "keywords":    { "type": "array", "items": { "type": "string" } },
          "latest":      { "type": "string", "description": "semver" },
          "path":        { "type": "string", "description": "relative path to plugin dir" },
          "trust":       { "enum": ["official","verified","community"] },
          "downloads":   { "type": "integer" },
          "stars":       { "type": "integer" }
        }
      }
    }
  }
}
```

A CI job in the marketplace repo regenerates `index.json` on every push to `main`. Clients fetch **exactly one** URL to populate the browse screen.

## Per-version file (`plugins/<publisher>/<name>/<version>.json`)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["name","publisher","version","artifact","sha256","manifest"],
  "properties": {
    "name":      { "type": "string" },
    "publisher": { "type": "string" },
    "version":   { "type": "string" },
    "releaseNotes": { "type": "string" },
    "artifact": {
      "type": "object",
      "required": ["url","size"],
      "properties": {
        "url":  { "type": "string", "format": "uri" },
        "size": { "type": "integer", "description": "bytes" },
        "mirrors": { "type": "array", "items": { "type": "string", "format": "uri" } }
      }
    },
    "sha256":    { "type": "string", "pattern": "^[a-f0-9]{64}$" },
    "signature": {
      "type": "object",
      "properties": {
        "alg":       { "enum": ["minisign","ed25519","sigstore"] },
        "publicKey": { "type": "string" },
        "value":     { "type": "string" }
      }
    },
    "manifest":  { "$ref": "https://opendray.dev/schemas/plugin-manifest-v1.json" },
    "engines":   { "$ref": "#/$defs/engines" },
    "platforms": { "type": "array",
                   "items": { "enum": ["linux-x64","linux-arm64",
                                       "darwin-x64","darwin-arm64",
                                       "windows-x64","any"] } }
  }
}
```

**Rule:** The `manifest` field is a full copy of the bundle's `manifest.json`. Clients use this for consent-screen rendering without having to download the zip first.

## Publisher record (`publishers/<publisher>.json`)

```json
{
  "name": "acme",
  "displayName": "Acme Inc.",
  "homepage": "https://acme.example",
  "trust": "verified",
  "keys": [
    { "alg": "ed25519", "publicKey": "base64...", "addedAt": "2025-...", "expiresAt": "2027-..." }
  ],
  "domainVerification": { "method": "dns-txt", "record": "opendray-verify=..." }
}
```

## Plugin release artifact (the zip on the publisher's server)

### Required contents
```
<root>/
  manifest.json                # MUST match the marketplace per-version record
  LICENSE
  ui/                          # when form=webview
  bin/                         # when form=host
  README.md                    # strongly recommended
  CHANGELOG.md                 # strongly recommended
```

### Rules
- Zip, no tar. Cross-platform tooling friendliness.
- Max 20 MB for webview, 200 MB for host.
- No executables with setuid/setgid bits.
- No symlinks outside root.
- Root manifest hash MUST equal the registry-declared manifest hash; mismatch aborts install.

## Publish flow

1. Plugin author runs `opendray plugin publish` from SDK.
2. CLI:
   - Validates manifest against v1 schema.
   - Builds zip, computes sha256, signs if key configured.
   - Uploads artifact to a user-configured endpoint (GitHub Release default).
   - Forks the marketplace repo if needed.
   - Creates a branch, adds `<version>.json` and updates `meta.json`.
   - Opens a PR with a templated body (changelog, caps diff, screenshots).
3. Marketplace CI runs:
   - Schema validation.
   - SHA check against artifact URL.
   - Sandbox scan of the bundle (static checks for forbidden file types / suspicious binaries).
   - Capability-diff vs. previous version.
4. CODEOWNERS review.
5. Merge → `index.json` regenerated → clients see the plugin within the poll window (1 h default; users can force-refresh).

## Review criteria (gate for merging PRs)

- Manifest valid against v1 schema.
- Artifact reachable from mirrors and sha256 verifies.
- No hard-coded credentials in bundle (secret scanner).
- Bundle size below form-specific cap.
- Icon present and renders.
- New capabilities since last version are justified in PR description.
- For `verified` publisher: signature verifies against a registered publisher key.
- For `official` publisher: only OpenDray maintainers can add to `publishers/opendray.json`.

## Trust levels

| Level | Means | Granted by |
|-------|-------|-----------|
| `official` | Maintained by OpenDray core team | manual edit to `publishers/opendray.json` |
| `verified` | Publisher passed domain / identity verification + key registered | Marketplace maintainer adds `trust: "verified"` to `publishers/<name>.json` |
| `community` | Any merged PR author | default |
| `sideloaded` | Installed from a URL / local path, not via marketplace | client-side label |

Installer surfaces the trust level in the consent sheet.

## Kill-switch / revocation

`revocations.json`:
```json
{
  "version": 1,
  "entries": [
    { "name":"acme/evil",
      "versions":"<=1.2.3",
      "reason":"credential exfiltration",
      "recordedAt":"2025-...",
      "action":"uninstall" }     // "uninstall" | "disable" | "warn"
  ]
}
```

Clients poll this file every 6 h (and on app launch). On match:
- `uninstall` — auto-uninstall with banner.
- `disable` — flip enabled=false.
- `warn` — red banner, plugin keeps working until user acts.

> **Locked:** Revocation is advisory. Airgapped installs won't auto-act, but they'll see the warning on next network contact. We do not ship a "plugin keeps reporting home to mothership" feature.

## Mirror support

`index.json` and per-version JSON files must be publicly reachable from at least two URLs (configured in client settings). Default mirrors: `https://raw.githubusercontent.com/opendray/marketplace/main/…` and a Cloudflare-fronted copy. Clients fall back round-robin.

## Settings the user sees

Settings → Marketplace:
- **Registry URL** (default to official).
- **Auto-update plugins** (default: off; prompts on capability-broadening updates).
- **Allow community plugins** (default: on; toggle off for locked-down installs).
- **Refresh cache now** button.

## Go package ownership

- `plugin/market/` (new) — fetch/cache `index.json`, resolve URLs, verify sha256/signatures.
- `gateway/plugins_market.go` (new) — REST routes for browse/search/install.

## Out of scope for v1

- In-marketplace ratings / reviews.
- Paid plugins / billing.
- Private marketplaces (can work today by pointing registry URL at a private repo, but no explicit features).
