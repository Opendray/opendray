import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { SQLNamespace } from '@codemirror/lang-sql'
import {
  Database,
  Lock,
  Loader2,
  Pencil,
  Plus,
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

type RightView = { kind: 'table'; ref: TableRef } | { kind: 'console' }

// DatabasePanel is the Database project tab: a connection selector, a
// left schema tree, and a right pane that shows either a table's data
// grid or the SQL console.
export function DatabasePanel({ cwd }: DatabasePanelProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [activeId, setActiveId] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editConn, setEditConn] = useState<DBConnection | null>(null)
  const [view, setView] = useState<RightView>({ kind: 'console' })

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
      <div className="flex flex-col items-center justify-center gap-3 p-10 text-center">
        <Database className="text-muted-foreground h-8 w-8" />
        <div className="text-sm font-medium">
          {t('web.database.panel.emptyTitle')}
        </div>
        <div className="text-muted-foreground max-w-sm text-xs">
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
    <div className="flex h-[70vh] flex-col">
      <div className="flex items-center gap-2 border-b px-3 py-2">
        <Select
          value={active?.id ?? ''}
          onValueChange={(v) => {
            setActiveId(v)
            setView({ kind: 'console' })
          }}
        >
          <SelectTrigger className="w-64">
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
        <Button size="sm" variant="ghost" onClick={() => setDialogOpen(true)}>
          <Plus className="mr-1 h-3 w-3" />
          {t('web.database.panel.add')}
        </Button>
        {active && (
          <>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => setEditConn(active)}
              title={t('web.database.panel.edit')}
            >
              <Pencil className="h-3 w-3" />
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => {
                if (window.confirm(t('web.database.panel.confirmDelete')))
                  deleteMut.mutate(active.id)
              }}
              title={t('web.database.panel.delete')}
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          </>
        )}
        <Button
          size="sm"
          variant={view.kind === 'console' ? 'default' : 'outline'}
          className="ml-auto"
          onClick={() => setView({ kind: 'console' })}
        >
          <TerminalSquare className="mr-1 h-3 w-3" />
          {t('web.database.panel.console')}
        </Button>
      </div>

      {active && (
        <div className="flex min-h-0 flex-1">
          <div className="w-56 flex-none overflow-auto border-r p-2">
            <SchemaTree
              connectionId={active.id}
              selected={view.kind === 'table' ? view.ref : null}
              onSelect={(ref) => setView({ kind: 'table', ref })}
            />
          </div>
          <div className="min-w-0 flex-1">
            {view.kind === 'table' ? (
              <DataGrid
                connectionId={active.id}
                table={view.ref}
                readOnly={active.read_only}
              />
            ) : (
              <SQLConsole connectionId={active.id} schema={emptyNamespace} />
            )}
          </div>
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
