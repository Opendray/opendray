import { Suspense, useEffect, useState } from 'react'
import { Outlet } from '@tanstack/react-router'
import { Loader2, ChevronRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { SidebarNav } from './SidebarNav'
import { Topbar } from './Topbar'
import { CommandPalette, useCommandPaletteHotkey } from './CommandPalette'
import { HealthBanner } from './HealthBanner'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useLayout } from '@/stores/layout'
import { useIsMobile } from '../lib/useIsMobile'
import { cn } from '@/lib/utils'

export function AppShell() {
  const [paletteOpen, setPaletteOpen] = useState(false)
  useCommandPaletteHotkey(setPaletteOpen)

  const isMobile = useIsMobile()
  const sidebarCollapsed = useLayout((s) => s.sidebarCollapsed)
  const setSidebarCollapsed = useLayout((s) => s.setSidebarCollapsed)

  // Entering mobile collapses the nav so the workbench gets full width;
  // the user re-opens it as a slide-over via the edge arrow / topbar.
  useEffect(() => {
    if (isMobile) setSidebarCollapsed(true)
  }, [isMobile, setSidebarCollapsed])

  const navOpen = !sidebarCollapsed

  return (
    <TooltipProvider delayDuration={200}>
      <div className="h-svh flex flex-col bg-background text-foreground">
        <Topbar onOpenPalette={() => setPaletteOpen(true)} />
        <HealthBanner />
        <div className="flex-1 flex overflow-hidden min-h-0 relative">
          {isMobile ? (
            <>
              {/* Slide-over nav drawer */}
              <div
                className={cn(
                  'fixed inset-y-0 left-0 z-50 flex transition-transform duration-200 ease-out',
                  navOpen ? 'translate-x-0' : '-translate-x-full',
                )}
              >
                <SidebarNav />
              </div>
              {/* Backdrop (tap to close) */}
              {navOpen && (
                <div
                  className="fixed inset-0 z-40 bg-black/50"
                  onClick={() => setSidebarCollapsed(true)}
                  aria-hidden
                />
              )}
              {/* Edge handle to pull the nav in when closed */}
              {!navOpen && (
                <button
                  type="button"
                  onClick={() => setSidebarCollapsed(false)}
                  aria-label="Open navigation"
                  className="fixed left-0 top-1/2 -translate-y-1/2 z-30 h-12 w-5 rounded-r-md border border-l-0 border-border bg-card/90 text-muted-foreground flex items-center justify-center shadow-sm active:bg-card"
                >
                  <ChevronRight className="size-3.5" />
                </button>
              )}
            </>
          ) : (
            <SidebarNav />
          )}
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
