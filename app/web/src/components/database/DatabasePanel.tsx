import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Database,
  Lock,
  Loader2,
  Maximize2,
  Pencil,
  Plus,
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
import { DatabaseWorkbenchDialog } from './DatabaseWorkbenchDialog'

interface DatabasePanelProps {
  cwd: string
}

// DatabasePanel is the compact Database entry inside the Session
// inspector sidebar: a connection selector and a schema tree for quick
// browsing. Anything that needs real width — the data grid, the SQL
// console — opens in the near-fullscreen DatabaseWorkbenchDialog instead
// of being cramped into the sidebar. Scoped to the current session cwd.
export function DatabasePanel({ cwd }: DatabasePanelProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [activeId, setActiveId] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editConn, setEditConn] = useState<DBConnection | null>(null)
  const [wbOpen, setWbOpen] = useState(false)
  const [wbTable, setWbTable] = useState<TableRef | null>(null)

  const connQuery = useQuery({
    queryKey: ['db-connections', cwd],
    queryFn: () => listConnections(cwd),
  })

  const connections = connQuery.data ?? []
  const active = useMemo(
    () => connections.find((c) => c.id === activeId) ?? connections[0] ?? null,
    [connections, activeId],
  )

  const openWorkbench = (table: TableRef | null) => {
    setWbTable(table)
    setWbOpen(true)
  }

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
          onValueChange={(v) => setActiveId(v)}
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

      {active && (
        <>
          <Button
            size="sm"
            variant="outline"
            className="h-7 w-full"
            onClick={() => openWorkbench(null)}
          >
            <Maximize2 className="mr-1 h-3 w-3" />
            {t('web.database.panel.openWorkbench')}
          </Button>

          <div className="max-h-[calc(100vh-16rem)] overflow-auto rounded-md border p-1">
            <SchemaTree
              connectionId={active.id}
              selected={null}
              onSelect={(ref) => openWorkbench(ref)}
            />
          </div>
        </>
      )}

      <DatabaseWorkbenchDialog
        cwd={cwd}
        open={wbOpen}
        onOpenChange={setWbOpen}
        connectionId={active?.id}
        table={wbTable}
      />
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
