// Lazy-loaded highlight.js wrapper. Languages register on first use
// so the inspector's file viewer doesn't drag the whole hljs grammar
// set into the entry chunk. Vite's manualChunks pulls the loaded
// modules into a separate `hljs` chunk.

import type { HLJSApi } from 'highlight.js'

let hljsPromise: Promise<HLJSApi> | null = null
const registered = new Set<string>()

// Map common file extensions / basenames to a highlight.js language
// id. Falls back to 'plaintext' which hljs renders as raw text.
const EXT_MAP: Record<string, string> = {
  ts: 'typescript',
  tsx: 'typescript',
  js: 'javascript',
  jsx: 'javascript',
  mjs: 'javascript',
  cjs: 'javascript',
  json: 'json',
  jsonc: 'json',
  go: 'go',
  py: 'python',
  rb: 'ruby',
  rs: 'rust',
  java: 'java',
  kt: 'kotlin',
  kts: 'kotlin',
  swift: 'swift',
  c: 'c',
  h: 'c',
  cpp: 'cpp',
  cc: 'cpp',
  hpp: 'cpp',
  cs: 'csharp',
  php: 'php',
  sh: 'bash',
  bash: 'bash',
  zsh: 'bash',
  fish: 'bash',
  yaml: 'yaml',
  yml: 'yaml',
  toml: 'ini',
  ini: 'ini',
  env: 'bash',
  conf: 'ini',
  sql: 'sql',
  md: 'markdown',
  markdown: 'markdown',
  html: 'xml',
  htm: 'xml',
  xml: 'xml',
  svg: 'xml',
  css: 'css',
  scss: 'scss',
  less: 'less',
  vue: 'xml',
  dockerfile: 'dockerfile',
  makefile: 'makefile',
  diff: 'diff',
  patch: 'diff',
  graphql: 'graphql',
  gql: 'graphql',
  proto: 'protobuf',
  lua: 'lua',
  dart: 'dart',
  scala: 'scala',
  r: 'r',
  ex: 'elixir',
  exs: 'elixir',
  erl: 'erlang',
  hs: 'haskell',
  ml: 'ocaml',
}

const BASENAME_MAP: Record<string, string> = {
  Dockerfile: 'dockerfile',
  Makefile: 'makefile',
  Justfile: 'makefile', // close enough for highlighting purposes
  Rakefile: 'ruby',
  Gemfile: 'ruby',
}

export function detectLanguage(path: string | null): string {
  if (!path) return 'plaintext'
  const base = path.split('/').pop() ?? path
  if (BASENAME_MAP[base]) return BASENAME_MAP[base]
  const dot = base.lastIndexOf('.')
  if (dot < 0) return 'plaintext'
  const ext = base.slice(dot + 1).toLowerCase()
  return EXT_MAP[ext] ?? 'plaintext'
}

async function loadCore(): Promise<HLJSApi> {
  if (!hljsPromise) {
    hljsPromise = import('highlight.js/lib/core').then((m) => m.default)
  }
  return hljsPromise
}

// Each language is a separate chunk inside the `hljs` bundle. We
// import the ones we declared in EXT_MAP — anything not listed
// falls back to plaintext which doesn't need a registration.
const LANG_LOADERS: Record<string, () => Promise<{ default: unknown }>> = {
  typescript: () => import('highlight.js/lib/languages/typescript'),
  javascript: () => import('highlight.js/lib/languages/javascript'),
  json: () => import('highlight.js/lib/languages/json'),
  go: () => import('highlight.js/lib/languages/go'),
  python: () => import('highlight.js/lib/languages/python'),
  ruby: () => import('highlight.js/lib/languages/ruby'),
  rust: () => import('highlight.js/lib/languages/rust'),
  java: () => import('highlight.js/lib/languages/java'),
  kotlin: () => import('highlight.js/lib/languages/kotlin'),
  swift: () => import('highlight.js/lib/languages/swift'),
  c: () => import('highlight.js/lib/languages/c'),
  cpp: () => import('highlight.js/lib/languages/cpp'),
  csharp: () => import('highlight.js/lib/languages/csharp'),
  php: () => import('highlight.js/lib/languages/php'),
  bash: () => import('highlight.js/lib/languages/bash'),
  yaml: () => import('highlight.js/lib/languages/yaml'),
  ini: () => import('highlight.js/lib/languages/ini'),
  sql: () => import('highlight.js/lib/languages/sql'),
  markdown: () => import('highlight.js/lib/languages/markdown'),
  xml: () => import('highlight.js/lib/languages/xml'),
  css: () => import('highlight.js/lib/languages/css'),
  scss: () => import('highlight.js/lib/languages/scss'),
  less: () => import('highlight.js/lib/languages/less'),
  dockerfile: () => import('highlight.js/lib/languages/dockerfile'),
  makefile: () => import('highlight.js/lib/languages/makefile'),
  diff: () => import('highlight.js/lib/languages/diff'),
  graphql: () => import('highlight.js/lib/languages/graphql'),
  protobuf: () => import('highlight.js/lib/languages/protobuf'),
  lua: () => import('highlight.js/lib/languages/lua'),
  dart: () => import('highlight.js/lib/languages/dart'),
  scala: () => import('highlight.js/lib/languages/scala'),
  r: () => import('highlight.js/lib/languages/r'),
  elixir: () => import('highlight.js/lib/languages/elixir'),
  erlang: () => import('highlight.js/lib/languages/erlang'),
  haskell: () => import('highlight.js/lib/languages/haskell'),
  ocaml: () => import('highlight.js/lib/languages/ocaml'),
}

// highlightCode loads core + the requested grammar (memoized) and
// returns the rendered HTML string. Plaintext is short-circuited to
// HTML-escaped text so we still get safe rendering.
export async function highlightCode(
  text: string,
  lang: string,
): Promise<string> {
  if (lang === 'plaintext' || !LANG_LOADERS[lang]) {
    return escapeHtml(text)
  }
  const hljs = await loadCore()
  if (!registered.has(lang)) {
    const mod = await LANG_LOADERS[lang]()
    // hljs language modules export a function as default.
    hljs.registerLanguage(lang, mod.default as never)
    registered.add(lang)
  }
  try {
    return hljs.highlight(text, { language: lang, ignoreIllegals: true }).value
  } catch {
    return escapeHtml(text)
  }
}

// splitHighlightedLines splits hljs's HTML output at newline
// boundaries while keeping any open <span class="..."> tags balanced
// across lines — a multi-line block comment / string spans many
// rows but each row stays valid HTML on its own.
export function splitHighlightedLines(html: string): string[] {
  const out: string[] = []
  const open: string[] = [] // stack of currently-open <span ...> opening tags
  let cur = ''
  let i = 0
  while (i < html.length) {
    const ch = html[i]
    if (ch === '<') {
      const close = html.indexOf('>', i)
      if (close < 0) {
        cur += html.slice(i)
        break
      }
      const tag = html.slice(i, close + 1)
      cur += tag
      if (tag.startsWith('</')) {
        open.pop()
      } else if (!tag.endsWith('/>')) {
        open.push(tag)
      }
      i = close + 1
      continue
    }
    if (ch === '\n') {
      // Close every open span on this line, then reopen on the next
      // so the colors continue across the line break.
      cur += '</span>'.repeat(open.length)
      out.push(cur)
      cur = open.join('')
      i++
      continue
    }
    cur += ch
    i++
  }
  if (cur.length > 0 || out.length === 0) {
    cur += '</span>'.repeat(open.length)
    out.push(cur)
  }
  return out
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
