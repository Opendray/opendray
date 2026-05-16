import { useCallback, useRef } from 'react'
import {
  Brain,
  Folder,
  GitBranch,
  Search,
  Play,
  NotebookPen,
  History as HistoryIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { ScrollArea } from '@/components/ui/scroll-area'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  INSPECTOR_WIDTH_DEFAULT,
  INSPECTOR_WIDTH_MAX,
  INSPECTOR_WIDTH_MIN,
  useLayout,
} from '@/stores/layout'
import type { Session } from '@/lib/types'

import { FilesPanel } from './inspector/FilesPanel'
import { GitPanel } from './inspector/GitPanel'
import { HistoryPanel } from './inspector/HistoryPanel'
import { MemoryPanel } from './inspector/MemoryPanel'
import { NotesPanel } from './inspector/NotesPanel'
import { SearchPanel } from './inspector/SearchPanel'
import { TaskRunnerPanel } from './inspector/TaskRunnerPanel'

interface InspectorPanelProps {
  session: Session
}

// InspectorPanel — right-hand workbench sidebar. All four tabs are
// scoped to the current session's working directory. Width is
// user-resizable via the drag handle on the left edge; the size is
// persisted in the layout store so it survives reloads. During the
// drag we write the width directly to the aside's style attribute
// (bypassing React) so the panel tracks the pointer at native
// frame rate — the store only gets the final value on release.
export function InspectorPanel({ session }: InspectorPanelProps) {
  const { t } = useTranslation()
  const width = useLayout((s) => s.inspectorWidth)
  const setWidth = useLayout((s) => s.setInspectorWidth)
  const asideRef = useRef<HTMLElement>(null)

  const onHandlePointerDown = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      e.preventDefault()
      const handle = e.currentTarget
      handle.setPointerCapture(e.pointerId)
      const aside = asideRef.current
      if (!aside) return
      const clamp = (v: number) =>
        Math.min(INSPECTOR_WIDTH_MAX, Math.max(INSPECTOR_WIDTH_MIN, v))

      const onMove = (ev: PointerEvent) => {
        // Inspector is anchored to the right edge of the viewport,
        // so width = viewport - pointerX.
        const next = clamp(Math.round(window.innerWidth - ev.clientX))
        aside.style.width = `${next}px`
      }
      const onUp = (ev: PointerEvent) => {
        handle.removeEventListener('pointermove', onMove)
        handle.removeEventListener('pointerup', onUp)
        handle.removeEventListener('pointercancel', onUp)
        const final = clamp(Math.round(window.innerWidth - ev.clientX))
        setWidth(final)
      }
      handle.addEventListener('pointermove', onMove)
      handle.addEventListener('pointerup', onUp)
      handle.addEventListener('pointercancel', onUp)
    },
    [setWidth],
  )

  const onHandleDoubleClick = useCallback(() => {
    if (asideRef.current) {
      asideRef.current.style.width = `${INSPECTOR_WIDTH_DEFAULT}px`
    }
    setWidth(INSPECTOR_WIDTH_DEFAULT)
  }, [setWidth])

  return (
    <aside
      ref={asideRef}
      style={{ width }}
      className="shrink-0 border-l border-border bg-background flex flex-col relative"
    >
      {/* Drag handle: a 6px column on the absolute left edge of
          the panel. Hover and active states highlight the inner
          1px line so the operator can see what's grabbable.
          Double-click resets to the default width. */}
      <div
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize inspector"
        onPointerDown={onHandlePointerDown}
        onDoubleClick={onHandleDoubleClick}
        className="absolute left-0 top-0 bottom-0 w-1.5 -translate-x-1/2 z-10 cursor-ew-resize group"
      >
        <div className="absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-transparent group-hover:bg-primary/40 group-active:bg-primary transition-colors" />
      </div>
      <Tabs defaultValue="files" className="flex-1 flex flex-col min-h-0">
        <div className="px-2 py-2 border-b border-border shrink-0">
          {/* 7 tabs in a 4-col grid → row 1: Files / Git / Search / Tasks,
              row 2: History (2) + Notes (2),
              row 3: Memory (4) — mirrors mobile's 🏁 Project memory
              shortcut that jumps from session detail to ProjectScreen. */}
          <TabsList className="bg-transparent border-0 p-0 gap-0.5 w-full grid grid-cols-4 gap-y-0.5">
            <TabsTrigger
              value="files"
              className="flex items-center justify-center gap-1.5 data-[state=active]:bg-card"
            >
              <Folder className="size-3" />
              {t('web.sessions.inspector.tabs.files')}
            </TabsTrigger>
            <TabsTrigger
              value="git"
              className="flex items-center justify-center gap-1.5 data-[state=active]:bg-card"
            >
              <GitBranch className="size-3" />
              {t('web.sessions.inspector.tabs.git')}
            </TabsTrigger>
            <TabsTrigger
              value="search"
              className="flex items-center justify-center gap-1.5 data-[state=active]:bg-card"
            >
              <Search className="size-3" />
              {t('web.sessions.inspector.tabs.search')}
            </TabsTrigger>
            <TabsTrigger
              value="tasks"
              className="flex items-center justify-center gap-1.5 data-[state=active]:bg-card"
            >
              <Play className="size-3" />
              {t('web.sessions.inspector.tabs.tasks')}
            </TabsTrigger>
            <TabsTrigger
              value="history"
              className="flex items-center justify-center gap-1.5 col-span-2 data-[state=active]:bg-card"
            >
              <HistoryIcon className="size-3" />
              {t('web.sessions.inspector.tabs.history')}
            </TabsTrigger>
            <TabsTrigger
              value="notes"
              className="flex items-center justify-center gap-1.5 col-span-2 data-[state=active]:bg-card"
            >
              <NotebookPen className="size-3" />
              {t('web.sessions.inspector.tabs.notes')}
            </TabsTrigger>
            <TabsTrigger
              value="memory"
              className="flex items-center justify-center gap-1.5 col-span-4 data-[state=active]:bg-card"
            >
              <Brain className="size-3" />
              {t('web.sessions.inspector.tabs.memory')}
            </TabsTrigger>
          </TabsList>
        </div>

        <ScrollArea className="flex-1 min-h-0">
          <TabsContent value="files" className="m-0 p-3">
            <FilesPanel cwd={session.cwd} />
          </TabsContent>
          <TabsContent value="git" className="m-0 p-3">
            <GitPanel cwd={session.cwd} />
          </TabsContent>
          <TabsContent value="search" className="m-0 p-3">
            <SearchPanel cwd={session.cwd} />
          </TabsContent>
          <TabsContent value="tasks" className="m-0 p-3">
            <TaskRunnerPanel session={session} />
          </TabsContent>
          <TabsContent value="history" className="m-0 p-3">
            <HistoryPanel session={session} />
          </TabsContent>
          <TabsContent value="notes" className="m-0 p-3">
            <NotesPanel cwd={session.cwd} />
          </TabsContent>
          <TabsContent value="memory" className="m-0 p-3">
            <MemoryPanel cwd={session.cwd} />
          </TabsContent>
        </ScrollArea>
      </Tabs>
    </aside>
  )
}
