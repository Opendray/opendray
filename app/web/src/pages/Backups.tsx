import { Archive, Package } from 'lucide-react'
import { Link } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import { BackupsView } from '@/components/backup/BackupsView'

// BackupsPage is the operator-facing dashboard for the
// disaster-recovery backup feature (A in the v1 design): trigger
// runs, manage schedules + targets, inspect prior backups.
//
// User-level data exports (C) live at /export — there's a button on
// this page to jump there.
export function BackupsPage() {
  return (
    <div className="flex-1 min-h-0 flex flex-col">
      <header className="px-6 py-4 border-b border-border bg-card/30">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-base font-medium flex items-center gap-2">
              <Archive className="size-4 text-accent" />
              Backups
            </h1>
            <p className="text-[12px] text-muted-foreground mt-0.5">
              Encrypted PostgreSQL dumps written to a pluggable target.
              Configure schedules + retention, or trigger one-off
              backups for a quick safety net.
            </p>
          </div>
          <Button asChild variant="outline" size="sm" className="h-8 text-[11px]">
            <Link to="/export">
              <Package className="size-3.5 mr-1.5" />
              Export data
            </Link>
          </Button>
        </div>
      </header>
      <div className="flex-1 min-h-0 overflow-y-auto px-6 py-5">
        <div className="max-w-5xl">
          <BackupsView />
        </div>
      </div>
    </div>
  )
}
