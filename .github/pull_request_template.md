<!--
  Thanks for sending a PR! Filling this in helps reviewers land your change
  quickly. The conventions below are what we follow internally.

  PR title: conventional-commits format
    feat(scope): ...        new user-facing capability
    fix(scope): ...         bug fix
    docs(scope): ...        documentation only
    ci(scope): ...          CI / build / release pipeline
    chore(scope): ...       infra, dependency bumps, repo housekeeping
    refactor(scope): ...    non-behaviour-changing internal restructure

  See CONTRIBUTING.md for full conventions.
-->

## Summary

<!-- 1-3 sentences. What does this change do and why? -->

## Related issue

<!-- `Closes #N` if this fully resolves an issue (auto-closes on merge). Use
     `Refs #N` for partial work or follow-ups. -->

Closes #

## Type of change

- [ ] 🐞 Bug fix (non-breaking change that fixes an issue)
- [ ] ✨ New feature (non-breaking change that adds capability)
- [ ] 💥 Breaking change (existing behaviour changes — bump VERSIONING.md if user-visible)
- [ ] 📖 Documentation only
- [ ] 🔧 CI / build / release pipeline
- [ ] 🧹 Chore (deps, refactor, repo housekeeping)

## Surface(s) touched

<!-- Tick all that apply so reviewers know what to focus on. -->

- [ ] Backend (Go) — `internal/*`, `cmd/*`
- [ ] Web admin (React) — `app/web/`
- [ ] Mobile app (Flutter) — `app/mobile/`
- [ ] Channel adapter — `internal/channel/*`
- [ ] Memory subsystem — `internal/memory/*`
- [ ] Backup / restore — `internal/backup/*`
- [ ] Notes / vault — `internal/notes/*`, `internal/vaultgit/*`
- [ ] Integration API — `internal/integration/*`, `internal/gateway/*`
- [ ] DB migrations — `internal/store/migrations/*`
- [ ] Deploy artefacts — `deploy/`, `scripts/install*.sh`, `scripts/uninstall*.sh`
- [ ] Docs — `README*`, `docs/`, `CHANGELOG.md`

## Test plan

<!-- How did you verify this? Be concrete enough that a reviewer can repeat it. -->

- [ ] `go test -race ./...` passes locally (or DB-skipped tests skip cleanly)
- [ ] `cd app/web && pnpm build` (TS strict + Vite prod) succeeds
- [ ] Manual smoke: ...
- [ ] Tested against a fresh database (if DB-touching)
- [ ] Tested with both `config.toml` and pure env-var modes (if config-touching)

## DB / config / operator impact

<!-- Tick whichever applies. Be explicit when 'no' so the reviewer doesn't have to check. -->

- [ ] No DB migration
- [ ] Adds a new migration (`internal/store/migrations/NNNN_*.sql`) — idempotent + safe to re-run
- [ ] Changes existing user-facing config (env var rename, key removal, ...) — call out as **BREAKING** in CHANGELOG
- [ ] Adds new env vars / config keys — documented in `config.example.toml` and `docs/operator-guide.md`
- [ ] None of the above

## Screenshots / before-after (UI changes)

<!-- Drop screenshots or GIFs if web/mobile UI changed. -->

## Anything else?

<!-- Trade-offs, design choices to flag, follow-ups left intentionally unimplemented. -->
