import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { SQLNamespace } from '@codemirror/lang-sql'
import {
  Database,
  Lock,
  Loader2,
  Pencil,
  Plus,
  Table2,
  TerminalSquare,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  deleteConnection,
  listConnections,
  type DBConnection,
} from '@/lib/database'
import { ConnectionDialog } from './ConnectionDialog'
import { SchemaTree, type TableRef } from './SchemaTree'
import { DataGrid } from './DataGrid'
import { SQLConsole } from './SQLConsole'

interface DatabasePanelProps {
  cwd: string
}

// DatabasePanel is the per-project database workbench: a connection
// selector plus two modes — Browse (schema tree → table data grid) and
// SQL console. It renders in a vertical, compact layout so it fits the
// Session inspector's narrow sidebar; every query is scoped to the
// project cwd it is given (the current session's working directory).
export function DatabasePanel({ cwd }: DatabasePanelProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [activeId, setActiveId] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editConn, setEditConn] = useState<DBConnection | null>(null)
  const [mode, setMode] = useState<'browse' | 'console'>('browse')
  const [table, setTable] = useState<TableRef | null>(null)

  const connQuery = useQuery({
    queryKey: ['db-connections', cwd],
    queryFn: () => listConnections(cwd),
  })

  const connections = connQuery.data ?? []
  const active = useMemo(
    () => connections.find((c) => c.id === activeId) ?? connections[0] ?? null,
    [connections, activeId],
  )

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteConnection(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['db-connections', cwd] })
      toast.success(t('web.database.panel.deleted'))
    },
    onError: (e: Error) => toast.error(e.message),
  })

  if (connQuery.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
        <Loader2 className="h-4 w-4 animate-spin" />
        {t('web.database.panel.loading')}
      </div>
    )
  }

  if (connections.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 p-8 text-center">
        <Database className="text-muted-foreground h-7 w-7" />
        <div className="text-sm font-medium">
          {t('web.database.panel.emptyTitle')}
        </div>
        <div className="text-muted-foreground text-xs">
          {t('web.database.panel.emptyBody')}
        </div>
        <Button size="sm" onClick={() => setDialogOpen(true)}>
          <Plus className="mr-1 h-3 w-3" />
          {t('web.database.panel.addConnection')}
        </Button>
        <ConnectionDialog
          cwd={cwd}
          open={dialogOpen}
          connection={null}
          onOpenChange={setDialogOpen}
        />
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {/* Connection row */}
      <div className="flex items-center gap-1">
        <Select
          value={active?.id ?? ''}
          onValueChange={(v) => {
            setActiveId(v)
            setTable(null)
          }}
        >
          <SelectTrigger className="h-8 flex-1">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {connections.map((c) => (
              <SelectItem key={c.id} value={c.id}>
                <span className="inline-flex items-center gap-1">
                  {c.read_only && <Lock className="h-3 w-3" />}
                  {c.name}
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          size="icon"
          variant="ghost"
          className="h-8 w-8"
          onClick={() => setDialogOpen(true)}
          title={t('web.database.panel.add')}
        >
          <Plus className="h-3.5 w-3.5" />
        </Button>
        {active && (
          <>
            <Button
              size="icon"
              variant="ghost"
              className="h-8 w-8"
              onClick={() => setEditConn(active)}
              title={t('web.database.panel.edit')}
            >
              <Pencil className="h-3.5 w-3.5" />
            </Button>
            <Button
              size="icon"
              variant="ghost"
              className="h-8 w-8"
              onClick={() => {
                if (window.confirm(t('web.database.panel.confirmDelete')))
                  deleteMut.mutate(active.id)
              }}
              title={t('web.database.panel.delete')}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </>
        )}
      </div>

      {/* Mode switch */}
      <div className="flex gap-1">
        <Button
          size="sm"
          className="h-7 flex-1"
          variant={mode === 'browse' ? 'default' : 'outline'}
          onClick={() => setMode('browse')}
        >
          <Table2 className="mr-1 h-3 w-3" />
          {t('web.database.panel.browse')}
        </Button>
        <Button
          size="sm"
          className="h-7 flex-1"
          variant={mode === 'console' ? 'default' : 'outline'}
          onClick={() => setMode('console')}
        >
          <TerminalSquare className="mr-1 h-3 w-3" />
          {t('web.database.panel.console')}
        </Button>
      </div>

      {active && mode === 'browse' && (
        <div className="space-y-2">
          <div className="max-h-52 overflow-auto rounded-md border p-1">
            <SchemaTree
              connectionId={active.id}
              selected={table}
              onSelect={setTable}
            />
          </div>
          {table ? (
            <div className="h-80 overflow-hidden rounded-md border">
              <DataGrid
                connectionId={active.id}
                table={table}
                readOnly={active.read_only}
              />
            </div>
          ) : (
            <div className="text-muted-foreground p-2 text-xs">
              {t('web.database.panel.pickTable')}
            </div>
          )}
        </div>
      )}

      {active && mode === 'console' && (
        <div className="h-96 overflow-hidden rounded-md border">
          <SQLConsole connectionId={active.id} schema={emptyNamespace} />
        </div>
      )}

      <ConnectionDialog
        cwd={cwd}
        open={dialogOpen}
        connection={null}
        onOpenChange={setDialogOpen}
      />
      <ConnectionDialog
        cwd={cwd}
        open={editConn !== null}
        connection={editConn}
        onOpenChange={(v) => !v && setEditConn(null)}
      />
    </div>
  )
}

// emptyNamespace is a placeholder autocompletion schema; table/column
// completion refines as the operator browses. Kept minimal to avoid
// eagerly introspecting every table just to power the editor.
const emptyNamespace: SQLNamespace = {}
