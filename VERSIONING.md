# Versioning

This document explains how opendray version numbers work. It is
intentionally short — but the policy it captures is deliberate, so
read it before opening a PR that touches versioning, tags, or
release-pipeline behaviour.

## TL;DR

- **Major version = product generation.** v1 is the legacy
  `Opendray/opendray` codebase (archived). v2 is this codebase
  (`Opendray/opendray_v2`). v3 is reserved for a future cross-
  architecture rewrite.
- **Minor version = feature iteration.** New channels, new providers,
  new memory subsystems, new API endpoints — anything additive within
  the current generation.
- **Patch version = fix / polish / docs.** Anything not user-facing
  and anything that fixes a regression without changing the contract.
- We **don't** strictly follow SemVer-by-the-book. The major version
  is a generation marker, not a "breaking-change-since-the-last-major"
  flag.

## Why not strict SemVer?

SemVer's strict reading — "major bump on any backwards-incompatible
change to the public API" — works well for libraries, where the
public API is the only contract. opendray is a deployed system: the
public surface is the operator-facing config schema, the database
migrations, the channel webhooks, and the integration API. Some of
those move on different cadences, and a strict reading would force
either:

- Constant major bumps (e.g. "we renamed an env var, bump v2 → v3"),
  which dilutes what a major version *means*; or
- Frozen contracts within a major version, which would stop us from
  fixing legitimate design mistakes during a generation's lifetime.

The major-as-generation approach is widely used in the JavaScript
and infrastructure worlds — see Vue (2 → 3), React (16 → 17 → 18),
Angular (2 → 4 → 17), Webpack (4 → 5), Vite (4 → 5 → 6), MySQL (5.7
→ 8.0), Node.js (16 → 18 → 20), PHP (5 → 7 → 8 — skipping 6).
opendray follows the same convention.

## What a major bump means here

A v3 will ship when **any** of the following is true:

1. **Architectural rewrite** comparable to v1 → v2 (Flutter → Go +
   React, hand-rolled DB → pgx + pgvector, etc.).
2. **Forced cross-generation migration path** — operators cannot
   in-place upgrade and must run a one-shot conversion tool or
   re-deploy from scratch.
3. **Replacement of the core deployment contract** — different
   binary layout, different process model, different default
   on-disk paths that previous versions wouldn't recognise.

None of these is on the roadmap today. v2 is expected to absorb
many iterations of additive / corrective work before v3 is
considered.

## What a minor bump means here

- Adding a new channel kind (e.g. a new Bridge platform)
- Adding a new provider (e.g. a new AI CLI alongside Claude /
  Codex / Gemini)
- Adding a new API endpoint or event topic
- Adding a new top-level admin page
- Adding a new plugin / skill / MCP host capability
- Any feature that's net-additive to operators

## What a patch bump means here

- Bug fixes
- Documentation
- Internal refactors invisible to operators
- Build / CI / release-pipeline tweaks
- Dependency bumps that don't change behaviour

## Handling small operator-facing changes within a generation

Within a generation, some operator-facing changes will still happen
(env var renames, DB migrations, config schema tweaks). These do
**not** force a major bump. Instead:

- They land in a **minor** release if additive or strictly opt-in
  (e.g. a new keyfile path that an existing config keeps working
  without).
- They land in a **minor** release marked **BREAKING** in the
  release notes if existing operators need to do something
  (re-run `migrate`, move a file, set a new env var) to keep their
  deploy working.
- The release notes call out exactly what needs to change and
  provide the smallest possible migration path.

This is the same pattern Vue, React, and Webpack use within a major
version (e.g. React 17 → 18's `createRoot` migration). Operators
should always read the release notes between minor versions.

## Pre-release identifiers

For testing significant new functionality before a final release:

- `vX.Y.Z-rc1`, `vX.Y.Z-rc2`, … — release candidates against an
  upcoming `vX.Y.Z`. Promote the highest-numbered RC to `vX.Y.Z`
  once it stabilises (no force-push — both tags coexist forever).
- `vX.Y.Z-beta.N`, `vX.Y.Z-alpha.N` — earlier-stage previews. Used
  rarely; most pre-release testing happens via `-dev` builds from
  `main` HEAD instead.

## `main` HEAD builds vs release builds

Anything built from `main` HEAD (or a feature branch) is **not** a
release. The version banner injected at build time reflects this:

```
opendray vX.Y.Z-dev.<short-sha> (commit=<sha>, date=<iso>)
```

vs a release build:

```
opendray vX.Y.Z (commit=<sha>, date=<iso>)
```

The `-dev.<sha>` suffix is set automatically by the build pipeline
(see `Dockerfile` and `.goreleaser.yml` for the `-ldflags -X` wiring
into `internal/version/`). Operators reading the banner should never
mistake a `main` HEAD build for a frozen release.

## Why this document exists

This file exists because issue #165 surfaced a real misalignment
between what the previous `v1.0.0` tag claimed (frozen public API)
and what `main` actually was (114 commits of active iteration in 7
days, including admin and backup contract rewrites). Re-tagging as
v2.0.0 only fixes the symptom; capturing the policy here is what
prevents the next contributor from re-introducing the same
confusion.

If you ever feel tempted to push `vX.0.0` as a milestone marker
without it actually being a generation boundary, **stop and read
this file again**. Use a minor (`vX.Y.0`) or a release candidate
(`vX.Y.Z-rcN`) instead.
