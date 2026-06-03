#!/usr/bin/env node
// i18n parity guard.
//
// Compares every app/i18n/<locale>.json against the canonical en.json and
// reports drift, so it surfaces in PR review instead of silently degrading
// to English at runtime. The web (i18next fallbackLng: 'en') and mobile
// (slang fallback_strategy: base_locale) both fall back to English for a
// missing key, which means drift is otherwise INVISIBLE: a translated UI
// just shows an English string in one spot and nobody notices.
//
// Severity:
//   FAIL  (blocking, exit 1) - invalid JSON, or an interpolation-token
//          mismatch on a shared key. A dropped or renamed {placeholder} or
//          <1> Trans tag breaks interpolation at runtime; the fallback does
//          NOT save you from that, so it is a real bug.
//   WARN  (advisory, exit 0) - a key in en is missing here (renders English
//          via fallback) or an extra key not in en (stale or a typo).
//
// Plural-aware: i18next CLDR plural suffixes (_zero/_one/_two/_few/_many/
// _other) are grouped by base key for the missing/extra comparison, so a
// language with different plural rules than English (Russian, Arabic, ...)
// is not falsely flagged.
//
// Zero dependencies. Runs in CI (`node scripts/check-i18n-parity.mjs`) and
// locally. Exits non-zero only on blocking issues.

import { readFileSync, readdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const I18N_DIR = resolve(import.meta.dirname, '../app/i18n')
const CANONICAL = 'en'
const inCI = !!process.env.GITHUB_ACTIONS

// {name} single-brace vars (i18next/slang) + <1>/</1> numbered Trans tags.
// Both MUST be preserved (order may change) across a translation.
const TOKEN = /\{[^{}]*\}|<\/?\d+>/g
const PLURAL = /_(zero|one|two|few|many|other)$/

const base = (k) => k.replace(PLURAL, '')
const tokens = (s) => (s.match(TOKEN) ?? []).sort()

function flatten(node, prefix, out) {
  if (typeof node === 'string') out.set(prefix, node)
  else if (Array.isArray(node)) node.forEach((v, i) => flatten(v, `${prefix}[${i}]`, out))
  else if (node && typeof node === 'object')
    for (const [k, v] of Object.entries(node)) flatten(v, prefix ? `${prefix}.${k}` : k, out)
  return out
}

const readFlat = (file) => flatten(JSON.parse(readFileSync(resolve(I18N_DIR, file), 'utf8')), '', new Map())

let enFlat
try {
  enFlat = readFlat(`${CANONICAL}.json`)
} catch (e) {
  console.error(`FATAL: cannot parse canonical ${CANONICAL}.json: ${e.message}`)
  process.exit(2)
}

// en groups (plural bases / plain keys) + a representative token set per group
// (prefer the _other form) for checking locale-only plural forms.
const enGroups = new Set()
const enGroupTokens = new Map()
for (const [k, v] of enFlat) {
  const g = base(k)
  enGroups.add(g)
  if (!enGroupTokens.has(g) || k.endsWith('_other')) enGroupTokens.set(g, tokens(v).join('|'))
}

const locales = readdirSync(I18N_DIR)
  .filter((f) => f.endsWith('.json') && f !== `${CANONICAL}.json`)
  .map((f) => f.replace(/\.json$/, ''))
  .sort()

let blocking = 0
const summary = []

const annotate = (level, file, msg) => {
  if (inCI) console.log(`::${level} file=${file}::${msg}`)
}

for (const loc of locales) {
  const file = `app/i18n/${loc}.json`
  let locFlat
  try {
    locFlat = readFlat(`${loc}.json`)
  } catch (e) {
    blocking++
    console.error(`\n[${loc}] FAIL invalid JSON: ${e.message}`)
    annotate('error', file, `invalid JSON: ${e.message}`)
    summary.push(`| \`${loc}\` | invalid JSON | - | - |`)
    continue
  }

  const locGroups = new Set([...locFlat.keys()].map(base))
  const missing = [...enGroups].filter((g) => !locGroups.has(g)).sort()
  const extra = [...locGroups].filter((g) => !enGroups.has(g)).sort()

  // Token check: every shared exact key, plus locale-only plural forms whose
  // base exists in en (compared against en's representative form).
  const mismatch = []
  for (const [k, v] of locFlat) {
    const exact = enFlat.has(k)
    const g = base(k)
    if (!exact && !enGroups.has(g)) continue // truly extra; reported above
    const want = exact ? tokens(enFlat.get(k)).join('|') : enGroupTokens.get(g)
    const got = tokens(v).join('|')
    if (want !== got) mismatch.push({ k, en: want, loc: got })
  }

  if (missing.length) blocking += 0 // advisory
  if (mismatch.length) blocking += mismatch.length

  const pct = Math.round(((enGroups.size - missing.length) / enGroups.size) * 100)
  console.log(
    `\n[${loc}] ${pct}% complete  (missing:${missing.length} extra:${extra.length} token-mismatch:${mismatch.length})`,
  )
  for (const g of missing) {
    console.log(`  WARN missing (renders English via fallback): ${g}`)
    annotate('warning', file, `missing key, renders English via fallback: ${g}`)
  }
  for (const g of extra) {
    console.log(`  WARN extra, not in en (stale or typo): ${g}`)
    annotate('warning', file, `extra key not in en (stale or typo): ${g}`)
  }
  for (const m of mismatch) {
    console.error(`  FAIL token mismatch ${m.k}: en=[${m.en}] ${loc}=[${m.loc}]`)
    annotate('error', file, `interpolation token mismatch at ${m.k}: en=[${m.en}] vs ${loc}=[${m.loc}]`)
  }

  const miss = missing.length ? `${missing.length} missing` : 'complete'
  summary.push(`| \`${loc}\` | ${miss} | ${extra.length} | ${mismatch.length} |`)
}

if (process.env.GITHUB_STEP_SUMMARY) {
  const md = [
    '### i18n parity vs `en.json`',
    '',
    '| locale | missing keys | extra keys | token mismatches |',
    '| --- | --- | --- | --- |',
    ...summary,
    '',
    blocking
      ? `**${blocking} blocking issue(s)** (token mismatch or invalid JSON).`
      : 'No blocking issues.',
    '',
    'Missing keys are advisory (they render English via the i18next / slang fallback). Token mismatches and invalid JSON are blocking, because a dropped or renamed `{placeholder}` / `<1>` tag breaks interpolation at runtime.',
    '',
  ]
  writeFileSync(process.env.GITHUB_STEP_SUMMARY, md.join('\n') + '\n', { flag: 'a' })
}

console.log(`\n${blocking ? `FAILED: ${blocking} blocking issue(s).` : 'OK: no blocking issues.'}`)
process.exit(blocking ? 1 : 0)
