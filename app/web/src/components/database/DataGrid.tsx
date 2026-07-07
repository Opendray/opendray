import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowDown,
  ArrowUp,
  ChevronLeft,
  ChevronRight,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  deleteRows,
  getTableData,
  getTableMeta,
  type DBSort,
} from '@/lib/database'
import { cellText } from './cell'
import { RowDialog } from './RowDialog'
import type { TableRef } from './SchemaTree'

const PAGE_SIZE = 100

interface DataGridProps {
  connectionId: string
  table: TableRef
  // readOnly disables all mutation affordances (connection.read_only).
  readOnly: boolean
}

// DataGrid is the table browser: a paginated, sortable grid with row
// insert/edit/delete when the connection is writable and the table has a
// primary key. No virtualization — 100 rows/page keeps the DOM small.
export function DataGrid({ connectionId, table, readOnly }: DataGridProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [page, setPage] = useState(0)
  const [sort, setSort] = useState<DBSort[]>([])
  const [editRow, setEditRow] = useState<Record<string, unknown> | null>(null)
  const [insertOpen, setInsertOpen] = useState(false)

  const metaQuery = useQuery({
    queryKey: ['db-table-meta', connectionId, table.schema, table.table],
    queryFn: () => getTableMeta(connectionId, table.schema, table.table),
  })

  const dataQuery = useQuery({
    queryKey: [
      'db-table-data',
      connectionId,
      table.schema,
      table.table,
      page,
      sort,
    ],
    queryFn: () =>
      getTableData(connectionId, {
        schema: table.schema,
        table: table.table,
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
        sort,
      }),
  })

  const meta = metaQuery.data
  const canEdit = !readOnly && (meta?.primary_key.length ?? 0) > 0

  const pkIndex = useMemo(() => {
    if (!meta || !dataQuery.data) return []
    return meta.primary_key.map((pk) =>
      dataQuery.data.columns.findIndex((c) => c.name === pk),
    )
  }, [meta, dataQuery.data])

  const toggleSort = (col: string) => {
    setPage(0)
    setSort((prev) => {
      const existing = prev.find((s) => s.column === col)
      if (!existing) return [{ column: col, desc: false }]
      if (!existing.desc) return [{ column: col, desc: true }]
      return []
    })
  }

  const rowToRecord = (row: unknown[]): Record<string, unknown> => {
    const rec: Record<string, unknown> = {}
    dataQuery.data?.columns.forEach((c, i) => {
      rec[c.name] = row[i]
    })
    return rec
  }

  const deleteMut = useMutation({
    mutationFn: (row: unknown[]) => {
      if (!meta) throw new Error('metadata not loaded')
      const pk: Record<string, unknown> = {}
      meta.primary_key.forEach((k, i) => {
        pk[k] = row[pkIndex[i]]
      })
      return deleteRows(connectionId, table.schema, table.table, [pk])
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['db-table-data'] })
      toast.success(t('web.database.grid.deleted'))
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const data = dataQuery.data
  const sortFor = (col: string) => sort.find((s) => s.column === col)

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div className="text-sm font-medium">
          {table.schema}.{table.table}
        </div>
        <div className="flex items-center gap-1">
          <Button
            size="sm"
            variant="ghost"
            onClick={() =>
              qc.invalidateQueries({ queryKey: ['db-table-data'] })
            }
            title={t('web.database.grid.refresh')}
          >
            <RefreshCw className="h-3 w-3" />
          </Button>
          {canEdit && (
            <Button size="sm" variant="outline" onClick={() => setInsertOpen(true)}>
              <Plus className="mr-1 h-3 w-3" />
              {t('web.database.grid.insert')}
            </Button>
          )}
        </div>
      </div>

      {readOnly && (
        <div className="text-muted-foreground border-b bg-muted/30 px-3 py-1 text-[11px]">
          {t('web.database.grid.readOnlyHint')}
        </div>
      )}
      {!readOnly && meta && meta.primary_key.length === 0 && (
        <div className="border-b bg-amber-500/10 px-3 py-1 text-[11px] text-amber-700 dark:text-amber-400">
          {t('web.database.grid.noPkHint')}
        </div>
      )}

      <div className="flex-1 overflow-auto">
        {dataQuery.isLoading ? (
          <div className="text-muted-foreground flex items-center gap-2 p-3 text-xs">
            <Loader2 className="h-3 w-3 animate-spin" />
            {t('web.database.grid.loading')}
          </div>
        ) : dataQuery.isError ? (
          <div className="p-3 text-xs text-red-600 dark:text-red-400">
            {(dataQuery.error as Error).message}
          </div>
        ) : data ? (
          <table className="w-full border-collapse text-xs">
            <thead className="bg-muted/50 sticky top-0">
              <tr>
                {canEdit && <th className="w-16 border-b px-2 py-1" />}
                {data.columns.map((c) => {
                  const s = sortFor(c.name)
                  return (
                    <th
                      key={c.name}
                      className="hover:bg-muted cursor-pointer border-b px-2 py-1 text-left font-semibold whitespace-nowrap select-none"
                      onClick={() => toggleSort(c.name)}
                      title={c.type}
                    >
                      <span className="inline-flex items-center gap-1">
                        {c.name}
                        {s?.desc === false && <ArrowUp className="h-3 w-3" />}
                        {s?.desc === true && <ArrowDown className="h-3 w-3" />}
                      </span>
                    </th>
                  )
                })}
              </tr>
            </thead>
            <tbody>
              {data.rows.map((row, ri) => (
                <tr key={ri} className="hover:bg-muted/30">
                  {canEdit && (
                    <td className="border-b px-1 py-0.5 whitespace-nowrap">
                      <button
                        type="button"
                        className="hover:text-primary text-muted-foreground p-0.5"
                        onClick={() => setEditRow(rowToRecord(row))}
                        title={t('web.database.grid.edit')}
                      >
                        <Pencil className="h-3 w-3" />
                      </button>
                      <button
                        type="button"
                        className="p-0.5 text-muted-foreground hover:text-red-600"
                        onClick={() => {
                          if (window.confirm(t('web.database.grid.confirmDelete')))
                            deleteMut.mutate(row)
                        }}
                        title={t('web.database.grid.delete')}
                      >
                        <Trash2 className="h-3 w-3" />
                      </button>
                    </td>
                  )}
                  {row.map((cell, ci) => (
                    <td
                      key={ci}
                      className="max-w-[24rem] truncate border-b px-2 py-1 font-mono whitespace-nowrap"
                      title={cellText(cell)}
                    >
                      {cell === null || cell === undefined ? (
                        <span className="text-muted-foreground italic">NULL</span>
                      ) : (
                        cellText(cell)
                      )}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        ) : null}
      </div>

      <div className="flex items-center justify-between gap-2 border-t px-3 py-1.5 text-xs">
        <span className="text-muted-foreground">
          {t('web.database.grid.pageInfo', {
            from: page * PAGE_SIZE + 1,
            to: page * PAGE_SIZE + (data?.rows.length ?? 0),
          })}
          {data?.truncated ? ' +' : ''}
        </span>
        <div className="flex items-center gap-1">
          <Button
            size="sm"
            variant="ghost"
            disabled={page === 0}
            onClick={() => setPage((p) => Math.max(0, p - 1))}
          >
            <ChevronLeft className="h-3 w-3" />
          </Button>
          <span>{page + 1}</span>
          <Button
            size="sm"
            variant="ghost"
            disabled={!data?.truncated}
            onClick={() => setPage((p) => p + 1)}
          >
            <ChevronRight className="h-3 w-3" />
          </Button>
        </div>
      </div>

      {meta && (
        <>
          <RowDialog
            connectionId={connectionId}
            meta={meta}
            open={insertOpen}
            row={null}
            onOpenChange={setInsertOpen}
            onSaved={() =>
              qc.invalidateQueries({ queryKey: ['db-table-data'] })
            }
          />
          <RowDialog
            connectionId={connectionId}
            meta={meta}
            open={editRow !== null}
            row={editRow}
            onOpenChange={(v) => !v && setEditRow(null)}
            onSaved={() =>
              qc.invalidateQueries({ queryKey: ['db-table-data'] })
            }
          />
        </>
      )}
    </div>
  )
}
