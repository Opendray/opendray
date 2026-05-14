import { Suspense, useState } from 'react'
import { Outlet } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { SidebarNav } from './SidebarNav'
import { Topbar } from './Topbar'
import { CommandPalette, useCommandPaletteHotkey } from './CommandPalette'
import { HealthBanner } from './HealthBanner'
import { TooltipProvider } from '@/components/ui/tooltip'

export function AppShell() {
  const [paletteOpen, setPaletteOpen] = useState(false)
  useCommandPaletteHotkey(setPaletteOpen)

  return (
    <TooltipProvider delayDuration={200}>
      <div className="h-svh flex flex-col bg-background text-foreground">
        <Topbar onOpenPalette={() => setPaletteOpen(true)} />
        <HealthBanner />
        <div className="flex-1 flex overflow-hidden min-h-0">
          <SidebarNav />
          <main className="flex-1 overflow-auto min-w-0">
            <Suspense fallback={<RouteFallback />}>
              <Outlet />
            </Suspense>
          </main>
        </div>
      </div>
      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
    </TooltipProvider>
  )
}

function RouteFallback() {
  const { t } = useTranslation()
  return (
    <div className="h-full flex items-center justify-center gap-2 text-[12px] text-muted-foreground">
      <Loader2 className="size-3.5 animate-spin" />
      {t('web.loading')}
    </div>
  )
}
