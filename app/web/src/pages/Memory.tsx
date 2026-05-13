import { Brain, FolderTree, Inbox, Workflow } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { MemoryInspector } from '@/components/settings/MemoryInspector'
import { listCleanupDecisions } from '@/lib/memoryCleanup'

// MemoryPage is the top-level browser/editor for the cross-CLI
// persistent memory store. Configuration (which embedder, dim,
// scope defaults) lives under Settings → Server → Memory; this
// page is the *runtime* view: browse / search / edit / delete
// what's actually stored.
//
// M9b+M14b: surfaces shortcuts to project-scoped memory (goal /
// plan / journal / inbox / cleanup) and the cross-project cleanup
// inbox, so the operator doesn't have to dig through scope dropdowns
// to find the new unified-memory surfaces.
export function MemoryPage() {
  const pendingDecisions = useQuery({
    queryKey: ['cleanup-decisions', 'pending-count'],
    queryFn: () =>
      listCleanupDecisions({ status: 'pending', limit: 200 }),
    staleTime: 30_000,
  })
  const pendingCount = pendingDecisions.data?.length ?? 0

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <header className="border-border bg-card/30 border-b px-6 py-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="flex items-center gap-2 text-base font-medium">
              <Brain className="text-accent size-4" />
              Memory
            </h1>
            <p className="text-muted-foreground mt-0.5 text-[12px]">
              Browse, search and edit memories agents have stored via the
              opendray-memory MCP server.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              asChild
              variant="outline"
              size="sm"
              className="h-8 text-[11px]"
            >
              <Link to="/memory/project" search={{ cwd: '' }}>
                <FolderTree className="mr-1 size-3" />
                Project
              </Link>
            </Button>
            <Button
              asChild
              variant="outline"
              size="sm"
              className="h-8 text-[11px]"
            >
              <Link to="/memory/cleanup">
                <Inbox className="mr-1 size-3" />
                Cleanup inbox
                {pendingCount > 0 && (
                  <Badge variant="danger" className="ml-1 text-[9px]">
                    {pendingCount}
                  </Badge>
                )}
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
                Workers
              </Link>
            </Button>
            <Button
              asChild
              variant="outline"
              size="sm"
              className="h-8 text-[11px]"
            >
              <Link to="/settings" search={{ section: 'server.memory' }}>
                Configuration →
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
