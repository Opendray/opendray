import { useTranslation } from 'react-i18next'

import type { DBResultSet } from '@/lib/database'
import { cellText } from './cell'

interface ResultsTableProps {
  result: DBResultSet
  // maxHeight caps the scroll region; the table scrolls independently.
  className?: string
}

// ResultsTable renders a DBResultSet as a scrollable grid. It is
// read-only display shared by the SQL console and any query preview —
// row editing lives in DataGrid, which is table-aware.
export function ResultsTable({ result, className }: ResultsTableProps) {
  const { t } = useTranslation()
  // The API returns [] for these; guard against null so a stray null can't
  // crash the console with "Cannot read properties of null (reading 'map')".
  const columns = result.columns ?? []
  const rows = result.rows ?? []
  if (columns.length === 0) {
    return (
      <div className="text-muted-foreground p-3 text-xs">
        {t('web.database.results.noColumns', {
          rows: result.rows_affected,
          command: result.command ?? '',
        })}
      </div>
    )
  }
  return (
    <div className={className}>
      <div className="overflow-auto">
        <table className="w-full border-collapse text-xs">
          <thead className="bg-muted/50 sticky top-0">
            <tr>
              {columns.map((c) => (
                <th
                  key={c.name}
                  className="border-b px-2 py-1 text-left font-semibold whitespace-nowrap"
                  title={c.type}
                >
                  {c.name}
                  <span className="text-muted-foreground ml-1 font-normal">
                    {c.type}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, ri) => (
              <tr key={ri} className="hover:bg-muted/30">
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
      </div>
    </div>
  )
}
