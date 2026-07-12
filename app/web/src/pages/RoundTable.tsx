import { useTranslation } from 'react-i18next'
import { Users } from 'lucide-react'

import { RoundTablePanel } from '@/components/roundtable/RoundTablePanel'

// Round Table (experimental) — cross-vendor multi-agent discussion. A
// deterministic chair drives claude/codex/antigravity seats through
// propose → critique → synthesize and produces a structured Verdict.
export function RoundTablePage() {
  const { t } = useTranslation()
  return (
    <div className="flex-1 min-h-0 flex flex-col">
      <header className="px-6 py-4 border-b border-border bg-card/30">
        <h1 className="text-base font-medium flex items-center gap-2">
          <Users className="size-4 text-accent" />
          {t('web.roundTable.title')}
          <span className="rounded-full border border-border px-1.5 text-[10px] uppercase tracking-wide text-muted-foreground">
            {t('web.roundTable.experimental')}
          </span>
        </h1>
        <p className="text-[12px] text-muted-foreground mt-0.5">
          {t('web.roundTable.subtitle')}
        </p>
      </header>
      <div className="flex-1 min-h-0 overflow-hidden px-6 py-5">
        <RoundTablePanel />
      </div>
    </div>
  )
}
