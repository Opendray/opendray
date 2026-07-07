import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import type { SQLNamespace } from '@codemirror/lang-sql'
import { Loader2, Play } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { runQuery, type DBResultSet } from '@/lib/database'
import { SqlEditor } from './SqlEditor'
import { ResultsTable } from './ResultsTable'

interface SQLConsoleProps {
  connectionId: string
  // schema powers autocompletion; optional (loads lazily as tables open).
  schema?: SQLNamespace
}

// SQLConsole is the free-form query surface: a CodeMirror editor plus a
// results pane. Read vs write gating happens server-side (the connection
// scope + read_only flag); the console just surfaces whatever the
// gateway returns, including target-database errors verbatim.
export function SQLConsole({ connectionId, schema }: SQLConsoleProps) {
  const { t } = useTranslation()
  const [sql, setSql] = useState('')
  const [result, setResult] = useState<DBResultSet | null>(null)
  const [error, setError] = useState<string | null>(null)

  const runMut = useMutation({
    mutationFn: () => runQuery(connectionId, sql),
    onSuccess: (res) => {
      setResult(res)
      setError(null)
    },
    onError: (e: Error) => {
      setError(e.message)
      setResult(null)
    },
  })

  const run = () => {
    if (!sql.trim()) return
    runMut.mutate()
  }

  return (
    <div className="flex h-full flex-col gap-2 p-3">
      <SqlEditor
        value={sql}
        onChange={setSql}
        schema={schema}
        onRun={run}
        placeholder={t('web.database.console.placeholder')}
      />
      <div className="flex items-center gap-2">
        <Button size="sm" onClick={run} disabled={runMut.isPending || !sql.trim()}>
          {runMut.isPending ? (
            <Loader2 className="mr-1 h-3 w-3 animate-spin" />
          ) : (
            <Play className="mr-1 h-3 w-3" />
          )}
          {t('web.database.console.run')}
        </Button>
        <span className="text-muted-foreground text-[11px]">
          {t('web.database.console.runHint')}
        </span>
        {result && (
          <span className="text-muted-foreground ml-auto text-[11px]">
            {t('web.database.console.stats', {
              command: result.command ?? '',
              rows: result.rows.length || result.rows_affected,
              ms: result.duration_ms,
            })}
            {result.truncated ? ` · ${t('web.database.console.truncated')}` : ''}
          </span>
        )}
      </div>
      <div className="min-h-0 flex-1 overflow-auto rounded-md border">
        {error ? (
          <pre className="p-3 font-mono text-xs whitespace-pre-wrap text-red-600 dark:text-red-400">
            {error}
          </pre>
        ) : result ? (
          <ResultsTable result={result} />
        ) : (
          <div className="text-muted-foreground p-3 text-xs">
            {t('web.database.console.empty')}
          </div>
        )}
      </div>
    </div>
  )
}
