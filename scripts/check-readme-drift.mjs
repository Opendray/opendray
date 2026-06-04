#!/usr/bin/env node
// README translation drift advisory.
//
// The repo ships 10 README files: the canonical README.md plus nine
// README.<lang>.md translations. When a PR edits README.md but leaves the
// translations untouched they drift silently, because nothing fails and the
// diff looks complete. This script posts (and keeps in sync) a single
// advisory comment on the PR listing the translations that were not updated.
//
// ADVISORY ONLY: it never exits non-zero on drift, so it can never block a
// merge. A typo or link fix in README.md is a legitimate reason to ignore
// the comment; a substantive rewrite is a reason to act on it. The author
// decides.
//
// Idempotent: the comment is matched by a hidden marker and upserted, so
// re-running on the same PR edits the existing comment instead of stacking
// duplicates. When the drift is resolved (translations now in the diff, or
// README.md no longer in the diff) the stale comment is removed.
//
// Zero npm dependencies — shells out to the `gh` CLI via execFileSync with
// array args (never a shell string), so nothing in the comment body or API
// payload can be interpreted by a shell. Mirrors the zero-dep style of
// scripts/check-i18n-parity.mjs.
//
// Env (set by GitHub Actions): GH_TOKEN, GH_REPO (owner/repo), PR (number).
// Run: node scripts/check-readme-drift.mjs

import { execFileSync } from 'node:child_process'

const MARKER = '<!-- readme-translation-drift -->'
const CANONICAL = 'README.md'
const TRANSLATIONS = [
  'README.zh.md', 'README.fa.md', 'README.es.md', 'README.pt-BR.md',
  'README.ja.md', 'README.ko.md', 'README.fr.md', 'README.de.md',
  'README.ru.md',
]

const PR = process.env.PR
const REPO = process.env.GH_REPO
if (!PR) { console.error('PR env var (pull request number) is required'); process.exit(1) }
if (!REPO) { console.error('GH_REPO env var (owner/repo) is required'); process.exit(1) }

// Thin wrapper: array args, no shell, inherit GH_TOKEN from the environment.
const gh = (args, input) =>
  execFileSync('gh', args, { encoding: 'utf8', input, stdio: ['pipe', 'pipe', 'inherit'] })

// Files changed in this PR (paths only).
const changed = new Set(
  gh(['pr', 'diff', PR, '--name-only']).split('\n').map((s) => s.trim()).filter(Boolean),
)

// Locate an existing advisory comment by its hidden marker so we can edit or
// delete it instead of stacking duplicates.
const comments = JSON.parse(
  gh(['api', `repos/${REPO}/issues/${PR}/comments`, '--paginate', '--slurp']),
).flat()
const existing = comments.find((c) => c.body?.includes(MARKER))

const deleteExisting = () => {
  if (existing) {
    console.log(`Resolving stale drift comment (id=${existing.id}).`)
    gh(['api', '-X', 'DELETE', `repos/${REPO}/issues/comments/${existing.id}`])
  }
}

// No drift possible unless README.md itself changed.
if (!changed.has(CANONICAL)) {
  console.log('README.md not changed in this PR; nothing to flag.')
  deleteExisting()
  process.exit(0)
}

// Translations not touched alongside README.md.
const stale = TRANSLATIONS.filter((f) => !changed.has(f))

if (stale.length === 0) {
  console.log('README.md changed and all translations were updated too.')
  deleteExisting()
  process.exit(0)
}

// Friendly, not nagging (project style), and no em-dashes (repo copy rule).
const body = `${MARKER}
FYI: \`README.md\` changed in this PR but the following translations were not updated:

${stale.map((f) => `- \`${f}\``).join('\n')}

If your change is substantive enough to need re-translation, please update them too. If it is a small edit (typo, broken link, formatting), feel free to ignore this comment.`

if (existing) {
  console.log(`Updating existing drift comment (id=${existing.id}).`)
  gh(['api', '-X', 'PATCH', `repos/${REPO}/issues/comments/${existing.id}`, '-F', 'body=@-'], body)
} else {
  console.log('Posting new drift comment.')
  gh(['api', `repos/${REPO}/issues/${PR}/comments`, '-F', 'body=@-'], body)
}

console.log(`Flagged ${stale.length} un-updated translation(s).`)
