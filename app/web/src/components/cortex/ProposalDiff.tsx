import { Fragment } from 'react'
import { useTranslation } from 'react-i18next'
import type { DocDiff } from '@/lib/projectDocs'

// Reviewing a proposal used to mean reading the whole proposed document
// and spotting by eye what moved. This renders the server-computed
// line-level diff instead: changed lines with a little context, and an
// explicit marker for each collapsed region so the reviewer can tell
// "nothing else changed" from "the rest is hidden".
//
// Unified (single column) rather than side-by-side: knowledge pages run
// to hundreds of lines, and two columns of that is unreadable on a laptop
// and impossible on a phone.

export function ProposalDiff({
  diff,
  className = '',
}: {
  diff: DocDiff
  className?: string
}) {
  const { t } = useTranslation()

  if (diff.unchanged || diff.hunks.length === 0) {
    return (
      <div className={`text-muted-foreground p-3 text-xs ${className}`}>
        {t('web.diff.noChanges')}
      </div>
    )
  }

  return (
    <div className={className}>
      <div
        className="text-muted-foreground mb-2 flex items-center gap-3 text-[11px]"
        aria-label={t('web.diff.changedLines')}
      >
        <span className="text-emerald-400">
          {t('web.diff.added', { count: diff.added })}
        </span>
        <span className="text-red-400">
          {t('web.diff.removed', { count: diff.removed })}
        </span>
      </div>
      {/* Wide content scrolls inside its own box so the page body never
          scrolls horizontally. */}
      <div className="border-border overflow-x-auto rounded-md border">
        <table className="w-full border-collapse font-mono text-[11px]">
          <tbody>
            {diff.hunks.map((hunk, hi) => {
              let lineNo = hunk.start_line
              return (
                <Fragment key={hi}>
                  {hunk.skipped_before > 0 && (
                    <tr className="bg-muted/30">
                      <td
                        colSpan={2}
                        className="text-muted-foreground px-2 py-1 text-center text-[10px] italic"
                      >
                        {t('web.diff.collapsed', { count: hunk.skipped_before })}
                      </td>
                    </tr>
                  )}
                  {hunk.lines.map((ln, li) => {
                    // Removed lines don't exist in the new document, so they
                    // carry no new-document line number.
                    const n = ln.kind === 'remove' ? null : lineNo++
                    const tone =
                      ln.kind === 'add'
                        ? 'bg-emerald-500/10 text-emerald-200'
                        : ln.kind === 'remove'
                          ? 'bg-red-500/10 text-red-200'
                          : ''
                    const sign =
                      ln.kind === 'add' ? '+' : ln.kind === 'remove' ? '-' : ' '
                    return (
                      <tr key={`${hi}-${li}`} className={tone}>
                        <td className="text-muted-foreground/60 w-10 shrink-0 border-r px-1 py-0.5 text-right align-top tabular-nums select-none">
                          {n ?? ''}
                        </td>
                        <td className="px-2 py-0.5 align-top whitespace-pre-wrap">
                          <span className="select-none">{sign} </span>
                          {ln.text || ' '}
                        </td>
                      </tr>
                    )
                  })}
                </Fragment>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
