import { Archive, Brain, FolderTree, Workflow } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { MemoryInspector } from '@/components/settings/MemoryInspector'

// MemoryPage is the top-level browser/editor for the cross-CLI
// persistent memory store. Configuration (which embedder, dim,
// scope defaults) lives under Settings → Server → Memory; this
// page is the *runtime* view: browse / search / edit / delete
// what's actually stored.
//
// Surfaces shortcuts to project-scoped memory (goal / plan / journal
// / inbox) and the cross-project Archived (restorable) view, so the
// operator doesn't have to dig through scope dropdowns to find the
// unified-memory surfaces.
export function MemoryPage() {
  const { t } = useTranslation()

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <header className="border-border bg-card/30 border-b px-6 py-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="flex items-center gap-2 text-base font-medium">
              <Brain className="text-accent size-4" />
              {t('web.memory.title')}
            </h1>
            <p className="text-muted-foreground mt-0.5 text-[12px]">
              {t('web.memory.subtitle')}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              asChild
              variant="outline"
              size="sm"
              className="h-8 text-[11px]"
            >
              <Link to="/notes" search={{ mode: 'project', cwd: '' }}>
                <FolderTree className="mr-1 size-3" />
                {t('web.memory.navProject')}
              </Link>
            </Button>
            <Button
              asChild
              variant="outline"
              size="sm"
              className="h-8 text-[11px]"
            >
              <Link to="/memory/archived">
                <Archive className="mr-1 size-3" />
                {t('web.memory.navArchived')}
              </Link>
            </Button>
            <Button
              asChild
              variant="outline"
              size="sm"
              className="h-8 text-[11px]"
            >
              <Link to="/memory/workers">
                <Workflow className="mr-1 size-3" />
                {t('web.memory.navWorkers')}
              </Link>
            </Button>
            <Button
              asChild
              variant="outline"
              size="sm"
              className="h-8 text-[11px]"
            >
              <Link to="/settings" search={{ section: 'server.memory' }}>
                {t('web.memory.navConfiguration')}
              </Link>
            </Button>
          </div>
        </div>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
        <div className="max-w-4xl">
          <MemoryInspector />
        </div>
      </div>
    </div>
  )
}
