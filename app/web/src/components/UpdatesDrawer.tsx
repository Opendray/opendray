// Right-side "What's new" drawer opened from the sidebar Updates row.
// Fetches GitHub Releases (changelog fallback) and keeps read state in
// localStorage — see app/shared/src/lib/releases.ts.

import { useEffect } from 'react'
import { createPortal } from 'react-dom'
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { ExternalLink, Loader2, Sparkles, X } from 'lucide-react'
import { format, parseISO } from 'date-fns'

import {
  fetchLatestReleaseInfo,
  setLastReadRelease,
  type ReleaseInfo,
} from '@/lib/releases'
import { cn } from '@/lib/utils'

const POLL_MS = 6 * 60 * 60 * 1000

export function useLatestReleaseQuery() {
  return useQuery<ReleaseInfo>({
    queryKey: ['release-whats-new'],
    queryFn: fetchLatestReleaseInfo,
    staleTime: POLL_MS,
    refetchInterval: POLL_MS,
    retry: 1,
  })
}

export function UpdatesDrawer({
  open,
  onClose,
  onMarkedRead,
}: {
  open: boolean
  onClose: () => void
  onMarkedRead: () => void
}) {
  const { t } = useTranslation()
  const { data, isLoading, isError, error, refetch, isFetching } =
    useLatestReleaseQuery()

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  function markRead() {
    if (data?.version) {
      setLastReadRelease(data.version)
      onMarkedRead()
    }
    onClose()
  }

  const dateLabel = formatPublished(data?.publishedAt)

  return createPortal(
    <div className="fixed inset-0 z-[60] flex justify-end">
      <div
        aria-hidden
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px]"
        onClick={onClose}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="updates-drawer-title"
        className="relative h-full w-full max-w-md bg-background border-l border-border shadow-2xl flex flex-col"
      >
        <div className="flex items-start gap-2 px-4 py-3 border-b border-border">
          <span className="mt-0.5 flex size-7 items-center justify-center rounded-md bg-accent/15 text-accent">
            <Sparkles className="size-3.5" />
          </span>
          <div className="min-w-0 flex-1">
            <h2
              id="updates-drawer-title"
              className="text-[13px] font-medium leading-snug"
            >
              {data
                ? t('nav.updates.whatsNew', { version: data.tag })
                : t('nav.updates.title')}
            </h2>
            {dateLabel && (
              <p className="text-[11px] text-muted-foreground mt-0.5">
                {dateLabel}
              </p>
            )}
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={t('common.close')}
            className="rounded-md p-1 text-muted-foreground hover:text-foreground hover:bg-card"
          >
            <X className="size-3.5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-3">
          {isLoading && (
            <div className="flex items-center gap-2 text-[12px] text-muted-foreground py-6 justify-center">
              <Loader2 className="size-3.5 animate-spin" />
              {t('nav.updates.loading')}
            </div>
          )}

          {isError && !data && (
            <div className="space-y-3 py-4">
              <p className="text-[12px] text-muted-foreground leading-relaxed">
                {t('nav.updates.loadFailed', {
                  error: (error as Error)?.message ?? 'unknown',
                })}
              </p>
              <button
                type="button"
                onClick={() => void refetch()}
                className="rounded border border-border px-2.5 py-1 text-[11px] hover:bg-card"
              >
                {t('common.retry')}
              </button>
            </div>
          )}

          {data && (
            <div className="space-y-3">
              {data.highlights.length === 0 ? (
                <p className="text-[12px] text-muted-foreground leading-relaxed">
                  {t('nav.updates.noHighlights')}
                </p>
              ) : (
                <ul className="space-y-2.5">
                  {data.highlights.map((h, i) => (
                    <li
                      key={i}
                      className="flex gap-2 text-[12px] leading-relaxed text-foreground/90"
                    >
                      <span
                        className="mt-1.5 size-1.5 shrink-0 rounded-full bg-accent"
                        aria-hidden
                      />
                      <span>{h}</span>
                    </li>
                  ))}
                </ul>
              )}
              {data.source === 'changelog' && (
                <p className="text-[10px] text-muted-foreground/80">
                  {t('nav.updates.sourceChangelog')}
                </p>
              )}
            </div>
          )}
        </div>

        <div className="border-t border-border px-4 py-3 flex flex-col gap-2">
          <a
            href={data?.htmlUrl ?? 'https://github.com/Opendray/opendray/releases'}
            target="_blank"
            rel="noreferrer"
            className={cn(
              'inline-flex items-center justify-center gap-1.5 rounded-md border border-border',
              'px-3 py-1.5 text-[12px] font-medium hover:bg-card transition-colors',
            )}
          >
            {t('nav.updates.openFull')}
            <ExternalLink className="size-3 opacity-70" />
          </a>
          <button
            type="button"
            onClick={markRead}
            disabled={!data}
            className={cn(
              'inline-flex items-center justify-center rounded-md bg-primary px-3 py-1.5',
              'text-[12px] font-medium text-primary-foreground',
              'disabled:opacity-50 hover:opacity-95 transition-opacity',
            )}
          >
            {isFetching && !data
              ? t('nav.updates.loading')
              : t('nav.updates.markRead')}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  )
}

function formatPublished(iso: string | undefined): string | null {
  if (!iso) return null
  try {
    return format(parseISO(iso), 'MMM d, yyyy')
  } catch {
    return iso.slice(0, 10)
  }
}
