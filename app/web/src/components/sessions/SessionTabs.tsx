import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
import { useSessionTabs } from '@/stores/sessionTabs'

interface SessionTabsProps {
  // Called when the user clicks a tab's ✕. Owner decides the full
  // semantics (the page-level handler stops + removes the underlying
  // session, with a confirm for live ones).
  onCloseTab: (id: string) => void
}

export function SessionTabs({ onCloseTab }: SessionTabsProps) {
  const { t } = useTranslation()
  const tabs = useSessionTabs((s) => s.tabs)
  const currentId = useSessionTabs((s) => s.currentId)
  const setCurrent = useSessionTabs((s) => s.setCurrent)

  if (tabs.length === 0) return null

  return (
    <div className="h-9 flex items-center border-b border-border bg-background overflow-x-auto">
      {tabs.map((tab, i) => {
        const active = tab.id === currentId
        return (
          <div
            key={tab.id}
            role="tab"
            aria-selected={active}
            tabIndex={0}
            onClick={() => setCurrent(tab.id)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') setCurrent(tab.id)
            }}
            className={cn(
              'group h-9 px-3 flex items-center gap-2 border-r border-border cursor-pointer min-w-[140px] max-w-[220px] shrink-0 transition-colors',
              active
                ? 'bg-card text-foreground'
                : 'text-muted-foreground hover:text-foreground hover:bg-card/60',
            )}
          >
            <span className="text-[10px] font-mono text-muted-foreground/60 shrink-0">
              {i + 1}
            </span>
            <span className="text-[12px] truncate flex-1">
              {tab.name || tab.id}
            </span>
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                onCloseTab(tab.id)
              }}
              className="opacity-0 group-hover:opacity-100 hover:bg-border rounded-sm p-0.5 transition-opacity"
              aria-label={t('web.sessions.tabs.closeAria')}
              title={t('web.sessions.tabs.closeTitle')}
            >
              <X className="size-3" />
            </button>
            {active && (
              <span className="absolute h-px bg-accent" style={{ display: 'none' }} />
            )}
          </div>
        )
      })}
    </div>
  )
}
