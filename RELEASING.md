# Releasing OpenDray

The release ceremony for the gateway binary. Roughly 10 minutes when
everything goes right, double that the first time you do it after a
gap. Captures hard-won knowledge from prior releases so the next
person doesn't have to re-discover the gotchas.

For *what* the version number means (major-as-generation, not strict
SemVer), see [`VERSIONING.md`](VERSIONING.md). For the release notes
themselves, see [`CHANGELOG.md`](CHANGELOG.md).

## The chain

```
   PR with CHANGELOG.md entry  ─►  merge to main
                                          │
                                          ▼
                          git tag v<X.Y.Z> on main
                          git push origin v<X.Y.Z>
                                          │
                                          ▼
              .github/workflows/release.yml fires
              (trigger: push tags 'v*')
                                          │
                       ┌──────────────────┴──────────────────┐
                       ▼                                     ▼
                goreleaser builds                   cosign signs SHA256SUMS
                cross-platform binaries             SBOM generation (Anchore)
                       │                                     │
                       └──────────────────┬──────────────────┘
                                          ▼
                       DRAFT GitHub release with assets
                                          │
                                          ▼
                         human reviews + publishes
                          (release.draft: true in
                           .goreleaser.yml is the
                           safety check by design)
                                          │
                                          ▼
              notify-site-on-release.yml fires
              (trigger: release: types: [published])
                                          │
                                          ▼
              repository_dispatch → Opendray/opendray.dev
                                          │
                                          ▼
                opendray.dev auto-rebuilds (~30s)
                /changelog/ shows the new version
                                          │
                                          ▼
              `opendray update` on every host now
              advertises the new version
```

## The order-of-operations gotcha

**Land the CHANGELOG.md PR on main BEFORE pushing the tag.**

Goreleaser snapshots the entire repo at the tagged commit when it
builds the release. If the tag is at a commit that doesn't yet have
the v<X.Y.Z> CHANGELOG section, the release body comes out as
"See CHANGELOG.md for the full release notes" — useless to the people
who land on the release page, useless to the `opendray.dev` rebuild
that pulls from GitHub Releases.

Recovery if you tagged first: PATCH the release body via the GitHub
API with the changelog section after the fact. Annoying, single-use
mistake — better to do it in the right order.

## Repository secrets (one-time setup)

The release pipeline reads two repository secrets. `GITHUB_TOKEN` is
provided automatically by Actions — you don't manage it. The other:

- **`NPM_TOKEN`** — npm Automation token with publish permission for
  `opendray`, `opendray-{linux,darwin}-{x64,arm64}`, and
  `@opendray/sdk`. Used by the `Publish to npm` job in `release.yml`.

  Generate at https://www.npmjs.com/settings/~/tokens → "Generate New
  Token" → Classic → **Automation** (not Read-only, not Publish).
  Automation tokens bypass 2FA challenges in CI.

  Add at
  https://github.com/Opendray/opendray/settings/secrets/actions/new
  with the name `NPM_TOKEN`.

  If the secret is missing, `scripts/publish-npm.mjs` skips cleanly
  with a clear log message — the GitHub release tarballs still ship,
  just without the npm channel. To recover after adding the secret:
  re-run the failed `Publish to npm` job on the existing workflow
  run, or trigger `workflow_dispatch` on `release.yml` with the
  existing tag.

  Operational note: never paste an npm token into a chat transcript,
  PR comment, issue, or any persisted log. If you do, treat it as
  compromised — revoke immediately at the npm tokens page above and
  rotate the secret value.

## Picking the version number

The rules from [`VERSIONING.md`](VERSIONING.md):

- **Major** (`v2.x.x` → `v3.0.0`) — new product generation. Architectural
  rewrite, breaking changes the operator must consciously plan for,
  documented migration path. Rare.
- **Minor** (`v2.3.x` → `v2.4.0`) — new feature, possibly with new
  endpoints / new config / new schema migration. Backwards compatible.
- **Patch** (`v2.3.4` → `v2.3.5`) — bug fix, polish, documentation,
  CI/dependency bump. No new feature surface.

When in doubt, **minor**. Operators who care can read the changelog;
operators who don't will run `opendray update` and get the new thing.

## Step-by-step

Each step listed individually so you can copy-paste cleanly.

### 1. Update CHANGELOG.md on a feature branch

```bash
git checkout main
git pull origin main --ff-only
git checkout -b docs/changelog-v<X.Y.Z>
$EDITOR CHANGELOG.md
```

Add a new section between `## [Unreleased]` and the previous version's
section. Follow the Keep-a-Changelog format the file already uses —
group by `### Added` / `### Changed` / `### Fixed` / `### Security` /
`### API` / `### Config`.

Keep the lead paragraph short: one sentence on the headline capability
+ one sentence on flavor.

```bash
git add CHANGELOG.md
git commit -m "docs(changelog): add v<X.Y.Z> entry

<short paragraph mirroring the lead in the changelog>"
git push -u origin docs/changelog-v<X.Y.Z>
```

### 2. Bump README status line

If the release adds a notable capability (new minor or major), update
the `## Status` block in both `README.md` and `README.zh.md` to point
at the new version. Fold those edits into the same PR as the changelog
— or a separate one if you prefer.

### 3. Open the changelog PR and merge it

Open the PR, wait for CI green, merge. **Do not push the tag yet.**

### 4. Pull main, tag, push tag

```bash
git checkout main
git pull origin main --ff-only

# Annotated tag — the description shows up on the release page.
git tag -a v<X.Y.Z> -m "v<X.Y.Z> — <one-line headline>

See CHANGELOG.md for the full release notes."

git push origin v<X.Y.Z>
```

The release workflow fires immediately. Watch it:

```
https://github.com/Opendray/opendray/actions/workflows/release.yml
```

Takes ~2 minutes — goreleaser cross-compiles Linux/macOS × amd64/arm64,
cosign signs SHA256SUMS keylessly via Sigstore, Anchore generates an
SPDX SBOM. All artifacts uploaded to the draft release.

### 5. Review and publish the draft release

Open https://github.com/Opendray/opendray/releases. The new entry
appears at the top with a "Draft" badge.

- Click **Edit**.
- Skim the auto-populated body. If it's the right CHANGELOG section,
  good. If it's just "See CHANGELOG.md for the full release notes"
  you missed step 1 — fix via API after publish (see Recovery below)
  or cancel and start over.
- Confirm all 8 assets are attached:
  - `opendray_<X.Y.Z>_linux_amd64.tar.gz`
  - `opendray_<X.Y.Z>_linux_arm64.tar.gz`
  - `opendray_<X.Y.Z>_darwin_amd64.tar.gz`
  - `opendray_<X.Y.Z>_darwin_arm64.tar.gz`
  - `SHA256SUMS`
  - `SHA256SUMS.sig`  (cosign signature)
  - `SHA256SUMS.pem`  (cosign cert)
  - `sbom-opendray.spdx.json`
- Click **Publish release**.

### 6. Verify the downstream cascade fires

Within ~30 seconds:

```
https://github.com/Opendray/opendray/actions/workflows/notify-site-on-release.yml
```

Should show a run triggered by event=`release`, status=success. That
workflow POSTs a `repository_dispatch` event to `Opendray/opendray.dev`
which kicks the site rebuild:

```
https://github.com/Opendray/opendray.dev/actions
```

Look for `event=repository_dispatch`. Pages deploy completes in ~2
minutes; `https://opendray.dev/changelog/` then shows the new version.

If the notify workflow doesn't fire or `opendray.dev` doesn't rebuild:
- Check the workflow secret `ODSITE_DISPATCH_PAT` still exists on the
  opendray repo (Settings → Secrets and variables → Actions).
- Check the PAT hasn't expired (fine-grained PATs default to ~1y).
- The site has a daily cron as a safety net — it'll pick the release
  up within 24 h regardless.

### 7. Verify on a host

```bash
# On any host running opendray:
opendray update --check
```

Should report the new version as available. Then:

```bash
sudo opendray update --restart
opendray version
```

Should print the new `vX.Y.Z`.

## Recovery: changelog landed AFTER the tag

If you tagged first and the release body is the empty "See
CHANGELOG.md" line:

```bash
# Extract the v<X.Y.Z> section from CHANGELOG.md (drop the heading
# itself — GitHub shows the version separately).
python3 -c "
import re
body = open('CHANGELOG.md').read()
m = re.search(r'(## \[v<X.Y.Z>\][\s\S]+?)(?=^## \[)', body, re.MULTILINE)
notes = re.sub(r'^## \[v<X.Y.Z>\][^\n]*\n+', '', m.group(1).rstrip(), count=1)
print(notes)
" > /tmp/notes.md

# Look up the release id:
gh api repos/Opendray/opendray/releases | jq '.[] | select(.tag_name == "v<X.Y.Z>") | .id'

# PATCH the release body:
gh api repos/Opendray/opendray/releases/<id> \
    --method PATCH \
    --field body=@/tmp/notes.md
```

(`gh` CLI shown for clarity. Direct `curl -X PATCH .../releases/<id>`
with an auth token works the same.)

## Recovery: bad release / pulled back

GitHub releases are reversible. The atomic-cosign-signed binaries are
not — once `cosign verify-blob` has signed your SHA256SUMS for a tag,
that signature stays in the Sigstore transparency log forever. If a
release is genuinely broken:

1. **Mark the release as draft again** (PATCH `draft: true`) — pulls
   it from `releases/latest` so `opendray update` no longer offers it.
2. **Delete the tag** locally and on origin:
   `git tag -d v<X.Y.Z> && git push origin :refs/tags/v<X.Y.Z>`.
3. **Cut a new patch** with the fix. Use `v<X.Y.Z+1>` — never reuse a
   tag, the transparency log makes that ambiguous forever.

## Local pre-release checklist (the day-of)

Before tagging:

- [ ] `git status` shows clean working tree on `main`
- [ ] `git log -3` confirms the changelog commit is in
- [ ] CI on `main` is green (look at the badge in the README or hit
      `gh run list --workflow=ci.yml --limit=1`)
- [ ] `go test ./...` passes locally in a clean env (the two
      `internal/config` tests are env-pollution sensitive — pre-
      existing, not a blocker)
- [ ] The release matches your intent: features in the changelog
      under the right groups, no surprise commits between the last
      tag and this one (`git log v<prev>..main --oneline`)

After publishing:

- [ ] `https://github.com/Opendray/opendray/releases/latest` redirects
      to the new tag
- [ ] Auto-rebuild for `opendray.dev` fired (see step 6)
- [ ] `https://opendray.dev/changelog/` shows the new entry within
      ~2 minutes
- [ ] `opendray update --check` on a host reports the new version
- [ ] `npm view opendray version` returns the new version (and the
      four `opendray-{linux,darwin}-{x64,arm64}` packages match —
      one-liner:
      `for p in opendray opendray-linux-x64 opendray-linux-arm64 opendray-darwin-x64 opendray-darwin-arm64 @opendray/sdk; do npm view "$p" version; done`)

## Skipped today, on the roadmap

- **`release-please` automation.** The repo has a
  `.github/workflows/release-please.yml` file but the bot isn't
  currently configured to auto-open release PRs. Migrating to it
  (Conventional Commits everywhere + a release-please config) would
  make steps 1–4 above mostly automatic. Worth doing when the team
  grows past one maintainer.
- **Pre-release channels.** `v<X.Y.Z>-rc.N` tags would let us cut
  release candidates and validate them on staging before promoting.
  Goreleaser supports it; we just haven't formalized when to use it.
