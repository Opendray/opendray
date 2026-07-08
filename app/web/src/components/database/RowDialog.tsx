import { useEffect, useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  insertRow,
  updateRow,
  type DBTableMeta,
} from '@/lib/database'

interface RowDialogProps {
  connectionId: string
  meta: DBTableMeta
  open: boolean
  // row set = edit mode (values prefilled, keyed by column); null = insert.
  row: Record<string, unknown> | null
  onOpenChange: (v: boolean) => void
  onSaved: () => void
}

// A field value the user typed, plus whether they cleared it to NULL.
interface FieldState {
  text: string
  isNull: boolean
}

// RowDialog inserts or edits one row. Every editable column gets a text
// input plus a NULL toggle. On edit, the primary key columns address the
// row (sent unchanged in the WHERE); the backend rejects a PK subset, so
// we always send the full stored PK.
export function RowDialog({
  connectionId,
  meta,
  open,
  row,
  onOpenChange,
  onSaved,
}: RowDialogProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const editing = !!row
  const [fields, setFields] = useState<Record<string, FieldState>>({})

  useEffect(() => {
    if (!open) return
    const init: Record<string, FieldState> = {}
    for (const col of meta.columns) {
      const v = row?.[col.name]
      init[col.name] = {
        text: v === null || v === undefined ? '' : cellToInput(v),
        isNull: editing ? v === null || v === undefined : false,
      }
    }
    setFields(init)
  }, [open, meta, row, editing])

  const setField = (col: string, patch: Partial<FieldState>) =>
    setFields((f) => ({ ...f, [col]: { ...f[col], ...patch } }))

  // Build the values map. On edit we send only changed columns; on insert
  // we send every non-NULL field plus explicit NULLs the user set.
  const buildValues = (): Record<string, unknown> => {
    const out: Record<string, unknown> = {}
    for (const col of meta.columns) {
      const f = fields[col.name]
      if (!f) continue
      const original = row?.[col.name]
      const newVal = f.isNull ? null : coerce(f.text)
      if (editing) {
        const origNorm = original === undefined ? null : original
        if (JSON.stringify(origNorm) !== JSON.stringify(newVal)) {
          out[col.name] = newVal
        }
      } else if (f.isNull) {
        out[col.name] = null
      } else if (f.text !== '') {
        out[col.name] = newVal
      }
    }
    return out
  }

  const saveMut = useMutation({
    mutationFn: () => {
      const values = buildValues()
      if (editing) {
        const pk: Record<string, unknown> = {}
        for (const k of meta.primary_key) pk[k] = row?.[k]
        return updateRow(connectionId, meta.schema, meta.table, pk, values)
      }
      return insertRow(connectionId, meta.schema, meta.table, values)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['db-table-data'] })
      toast.success(
        editing
          ? t('web.database.row.savedEdit')
          : t('web.database.row.savedInsert'),
      )
      onSaved()
      onOpenChange(false)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (editing && Object.keys(buildValues()).length === 0) {
      toast.info(t('web.database.row.noChanges'))
      return
    }
    saveMut.mutate()
  }

  const pkSet = new Set(meta.primary_key)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {editing
              ? t('web.database.row.editTitle')
              : t('web.database.row.insertTitle')}
          </DialogTitle>
          <DialogDescription>
            {meta.schema}.{meta.table}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={onSubmit} className="max-h-[60vh] space-y-2 overflow-auto">
          {meta.columns.map((col) => {
            const f = fields[col.name] ?? { text: '', isNull: false }
            const isPK = pkSet.has(col.name)
            return (
              <div key={col.name} className="grid grid-cols-3 items-center gap-2">
                <Label className="truncate text-xs" title={col.data_type}>
                  {col.name}
                  {isPK && <span className="text-primary ml-1">PK</span>}
                </Label>
                <Input
                  className="col-span-2 font-mono text-xs"
                  value={f.text}
                  disabled={f.isNull}
                  placeholder={f.isNull ? 'NULL' : col.data_type}
                  onChange={(e) => setField(col.name, { text: e.target.value })}
                />
                {col.nullable && (
                  <label className="text-muted-foreground col-span-3 flex items-center justify-end gap-1 text-[10px]">
                    <input
                      type="checkbox"
                      checked={f.isNull}
                      onChange={(e) =>
                        setField(col.name, { isNull: e.target.checked })
                      }
                    />
                    {t('web.database.row.setNull')}
                  </label>
                )}
              </div>
            )
          })}
          <DialogFooter className="pt-2">
            <Button type="submit" disabled={saveMut.isPending}>
              {saveMut.isPending && (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              )}
              {t('web.database.row.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

// cellToInput renders a stored value for the text input (objects as JSON).
function cellToInput(v: unknown): string {
  if (typeof v === 'object' && v !== null) return JSON.stringify(v)
  return String(v)
}

// coerce turns the raw input text into a JSON value. Numbers and
// booleans are detected; JSON objects/arrays are parsed; everything else
// stays a string. The backend passes it as a positional parameter, so
// Postgres does the final cast against the column type.
function coerce(text: string): unknown {
  const trimmed = text.trim()
  if (trimmed === '') return ''
  if (trimmed === 'true') return true
  if (trimmed === 'false') return false
  if (/^-?\d+$/.test(trimmed)) {
    const n = Number(trimmed)
    if (Number.isSafeInteger(n)) return n
  }
  if (/^-?\d*\.\d+$/.test(trimmed)) return Number(trimmed)
  if (
    (trimmed.startsWith('{') && trimmed.endsWith('}')) ||
    (trimmed.startsWith('[') && trimmed.endsWith(']'))
  ) {
    try {
      return JSON.parse(trimmed)
    } catch {
      return text
    }
  }
  return text
}
