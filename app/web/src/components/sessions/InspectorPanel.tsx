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
// scoped to the current session's working directory.
export function InspectorPanel({ session }: InspectorPanelProps) {
  const { t } = useTranslation()
  return (
    <aside className="w-80 shrink-0 border-l border-border bg-background flex flex-col">
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
