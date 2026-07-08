// DatabasePage — the standalone Database tool (/database). Pick a project,
// then manage its database connections, browse schemas/tables, edit rows,
// and run SQL. Connections stay per-project (cwd-scoped) — the picker just
// chooses which project's databases to work with. Kept out of the Cortex
// module: the Database tool operates data, it is not knowledge curation.

import { useSearch, useNavigate } from '@tanstack/react-router'
import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { ProjectCwdPicker } from '@/components/project/ProjectCwdPicker'
import { DatabasePanel } from '@/components/database/DatabasePanel'

export function DatabasePage() {
  const { t } = useTranslation()
  const search = useSearch({ strict: false }) as { cwd?: string }
  const navigate = useNavigate()

  if (!search.cwd) {
    return (
      <ProjectCwdPicker
        title={t('web.database.page.pickTitle')}
        subtitle={t('web.database.page.pickSubtitle')}
        onSelect={(cwd) => navigate({ to: '/database', search: { cwd } })}
      />
    )
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2 border-b px-4 py-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate({ to: '/database', search: { cwd: '' } })}
        >
          <ArrowLeft className="mr-1 h-3 w-3" />
          {t('web.database.page.changeProject')}
        </Button>
        <span className="text-muted-foreground truncate font-mono text-xs">
          {search.cwd}
        </span>
      </div>
      <div className="min-h-0 flex-1 overflow-auto">
        <DatabasePanel cwd={search.cwd} />
      </div>
    </div>
  )
}
