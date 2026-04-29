import { Outlet } from '@tanstack/react-router'
import { SidebarNav } from './SidebarNav'
import { Topbar } from './Topbar'

export function AppShell() {
  return (
    <div className="h-svh flex flex-col bg-background text-foreground">
      <Topbar />
      <div className="flex-1 flex overflow-hidden min-h-0">
        <SidebarNav />
        <main className="flex-1 overflow-auto min-w-0">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
