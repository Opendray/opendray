import { useEffect, useRef } from 'react'
import { EditorState, type Extension } from '@codemirror/state'
import { EditorView, keymap, placeholder as cmPlaceholder } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { sql, PostgreSQL, type SQLNamespace } from '@codemirror/lang-sql'
import { autocompletion } from '@codemirror/autocomplete'

import { useTheme } from '@/stores/theme'

interface SqlEditorProps {
  value: string
  onChange: (value: string) => void
  // schema feeds table/column autocompletion: { "schema.table": ["col", …] }.
  schema?: SQLNamespace
  // onRun fires on Cmd/Ctrl-Enter so the console can execute without a
  // separate button press.
  onRun?: () => void
  placeholder?: string
}

// SqlEditor is a thin CodeMirror 6 wrapper: SQL syntax highlighting,
// schema-aware autocompletion, history, and a Cmd/Ctrl-Enter run binding.
// It is uncontrolled internally (CodeMirror owns the doc) but mirrors
// external value changes that don't originate from typing.
export function SqlEditor({
  value,
  onChange,
  schema,
  onRun,
  placeholder,
}: SqlEditorProps) {
  const host = useRef<HTMLDivElement | null>(null)
  const view = useRef<EditorView | null>(null)
  const onChangeRef = useRef(onChange)
  const onRunRef = useRef(onRun)
  const dark = useTheme((s) => s.applied()) === 'dark'

  // Keep the callback refs current without touching them during render —
  // the CodeMirror extensions close over the refs, not the props.
  useEffect(() => {
    onChangeRef.current = onChange
    onRunRef.current = onRun
  })

  useEffect(() => {
    if (!host.current) return
    const runKeymap = keymap.of([
      {
        key: 'Mod-Enter',
        run: () => {
          onRunRef.current?.()
          return true
        },
      },
    ])
    const extensions: Extension[] = [
      history(),
      keymap.of([...defaultKeymap, ...historyKeymap]),
      runKeymap,
      sql({ dialect: PostgreSQL, schema, upperCaseKeywords: true }),
      autocompletion(),
      EditorView.lineWrapping,
      EditorView.updateListener.of((u) => {
        if (u.docChanged) onChangeRef.current(u.state.doc.toString())
      }),
      EditorView.theme({
        '&': { fontSize: '13px' },
        '.cm-content': { fontFamily: 'var(--font-mono, ui-monospace, monospace)' },
        '&.cm-focused': { outline: 'none' },
      }),
    ]
    if (placeholder) extensions.push(cmPlaceholder(placeholder))
    if (dark) extensions.push(darkTheme)

    const state = EditorState.create({ doc: value, extensions })
    const v = new EditorView({ state, parent: host.current })
    view.current = v
    return () => {
      v.destroy()
      view.current = null
    }
    // Recreate the editor when the theme or schema identity changes;
    // value is synced separately below to avoid clobbering the cursor.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dark, schema, placeholder])

  // Mirror external value changes (e.g. "load into console") without
  // resetting the cursor on every keystroke.
  useEffect(() => {
    const v = view.current
    if (!v) return
    const current = v.state.doc.toString()
    if (current !== value) {
      v.dispatch({
        changes: { from: 0, to: current.length, insert: value },
      })
    }
  }, [value])

  return (
    <div
      ref={host}
      className="border-input bg-background max-h-64 min-h-24 overflow-auto rounded-md border"
    />
  )
}

// Minimal dark palette — CodeMirror ships no default dark theme and we
// avoid pulling @codemirror/theme-one-dark as an extra dependency.
const darkTheme = EditorView.theme(
  {
    '&': { color: '#e6e6e6', backgroundColor: 'transparent' },
    '.cm-content': { caretColor: '#e6e6e6' },
    '.cm-cursor, .cm-dropCursor': { borderLeftColor: '#e6e6e6' },
    '.cm-activeLine': { backgroundColor: 'rgba(255,255,255,0.04)' },
    '.cm-gutters': { backgroundColor: 'transparent', border: 'none' },
    '.cm-tooltip': {
      backgroundColor: '#1e1e1e',
      border: '1px solid #333',
      color: '#e6e6e6',
    },
  },
  { dark: true },
)
