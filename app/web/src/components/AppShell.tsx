import { Suspense, useEffect, useState } from 'react'
import { Outlet } from '@tanstack/react-router'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { SidebarNav } from './SidebarNav'
import { Topbar } from './Topbar'
import { SlideOverAside } from './SlideOverAside'
import { CommandPalette, useCommandPaletteHotkey } from './CommandPalette'
import { HealthBanner } from './HealthBanner'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useLayout } from '@/stores/layout'
import { useIsCompact, useIsMobile } from '../lib/useIsMobile'

export function AppShell() {
  const { t } = useTranslation()
  const [paletteOpen, setPaletteOpen] = useState(false)
  useCommandPaletteHotkey(setPaletteOpen)

  const isMobile = useIsMobile()
  const isCompact = useIsCompact()
  const sidebarCollapsed = useLayout((s) => s.sidebarCollapsed)
  const setSidebarCollapsed = useLayout((s) => s.setSidebarCollapsed)

  // Entering a narrow viewport collapses the nav so the workbench gets
  // the width. What "collapsed" means differs by size: on a tablet the
  // nav stays inline as a 48px icon rail (still one tap from anywhere),
  // on a phone there is no room even for that, so it becomes a
  // slide-over the user pulls in from the edge or the topbar toggle.
  useEffect(() => {
    if (isCompact) setSidebarCollapsed(true)
  }, [isCompact, setSidebarCollapsed])

  const navOpen = !sidebarCollapsed

  return (
    <TooltipProvider delayDuration={200}>
      {/* h-dvh (not h-svh): on iOS the visual viewport shrinks when the
          soft keyboard opens or grows when the address bar collapses.
          h-svh locks the shell to the address-bar-visible height, which
          leaves the bottom of the chat clipped past the visible viewport
          when the keyboard is up. h-dvh tracks the live viewport so the
          shell always fits exactly what the user can see. */}
      <div className="h-dvh flex flex-col bg-background text-foreground">
        <Topbar onOpenPalette={() => setPaletteOpen(true)} />
        <HealthBanner />
        <div className="flex-1 flex overflow-hidden min-h-0 relative">
          <SlideOverAside
            side="left"
            compact={isMobile}
            open={isMobile ? navOpen : true}
            onOpenChange={(v) => setSidebarCollapsed(!v)}
            label={t('web.topbar.expandSidebar')}
            // No edge handle: the topbar's sidebar toggle is always
            // visible, and a second handle here lands on top of the
            // page's own left-edge handle (session list, note tree, …).
            hideHandle
          >
            <SidebarNav />
          </SlideOverAside>
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
