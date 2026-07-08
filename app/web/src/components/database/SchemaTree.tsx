import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ChevronDown,
  ChevronRight,
  Database,
  Eye,
  Loader2,
  Table2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { listSchemas, listTables, type DBTable } from '@/lib/database'

export interface TableRef {
  schema: string
  table: string
}

interface SchemaTreeProps {
  connectionId: string
  selected: TableRef | null
  onSelect: (ref: TableRef) => void
}

// SchemaTree renders a lazy schema → table tree on the left of the
// Database panel. Schemas load up front; each schema's tables load when
// it is first expanded.
export function SchemaTree({
  connectionId,
  selected,
  onSelect,
}: SchemaTreeProps) {
  const { t } = useTranslation()
  const schemasQuery = useQuery({
    queryKey: ['db-schemas', connectionId],
    queryFn: () => listSchemas(connectionId),
  })

  if (schemasQuery.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-3 text-xs">
        <Loader2 className="h-3 w-3 animate-spin" />
        {t('web.database.tree.loading')}
      </div>
    )
  }
  if (schemasQuery.isError) {
    return (
      <div className="p-3 text-xs text-red-600 dark:text-red-400">
        {(schemasQuery.error as Error).message}
      </div>
    )
  }
  const schemas = schemasQuery.data ?? []
  if (schemas.length === 0) {
    return (
      <div className="text-muted-foreground p-3 text-xs">
        {t('web.database.tree.noSchemas')}
      </div>
    )
  }

  return (
    <div className="text-sm">
      {schemas.map((s) => (
        <SchemaNode
          key={s.name}
          connectionId={connectionId}
          schema={s.name}
          selected={selected}
          onSelect={onSelect}
          defaultOpen={schemas.length === 1 || s.name === 'public'}
        />
      ))}
    </div>
  )
}

function SchemaNode({
  connectionId,
  schema,
  selected,
  onSelect,
  defaultOpen,
}: {
  connectionId: string
  schema: string
  selected: TableRef | null
  onSelect: (ref: TableRef) => void
  defaultOpen: boolean
}) {
  const [open, setOpen] = useState(defaultOpen)
  const tablesQuery = useQuery({
    queryKey: ['db-tables', connectionId, schema],
    queryFn: () => listTables(connectionId, schema),
    enabled: open,
  })

  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="hover:bg-muted/50 flex w-full items-center gap-1 rounded px-1 py-0.5 text-left"
      >
        {open ? (
          <ChevronDown className="h-3 w-3 flex-none" />
        ) : (
          <ChevronRight className="h-3 w-3 flex-none" />
        )}
        <Database className="text-muted-foreground h-3 w-3 flex-none" />
        <span className="truncate font-medium">{schema}</span>
      </button>
      {open && (
        <div className="ml-4">
          {tablesQuery.isLoading && (
            <div className="text-muted-foreground flex items-center gap-1 py-0.5 pl-2 text-xs">
              <Loader2 className="h-3 w-3 animate-spin" />
            </div>
          )}
          {(tablesQuery.data ?? []).map((tb) => (
            <TableNode
              key={tb.name}
              table={tb}
              active={
                selected?.schema === schema && selected?.table === tb.name
              }
              onClick={() => onSelect({ schema, table: tb.name })}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function TableNode({
  table,
  active,
  onClick,
}: {
  table: DBTable
  active: boolean
  onClick: () => void
}) {
  const Icon = table.kind === 'view' ? Eye : Table2
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex w-full items-center gap-1 rounded px-1 py-0.5 text-left text-xs ${
        active ? 'bg-primary/15 text-primary' : 'hover:bg-muted/50'
      }`}
      title={table.name}
    >
      <Icon className="text-muted-foreground h-3 w-3 flex-none" />
      <span className="truncate">{table.name}</span>
    </button>
  )
}
