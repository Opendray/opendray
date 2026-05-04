import { Brain } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { MemoryInspector } from '@/components/settings/MemoryInspector'

// MemoryPage is the top-level browser/editor for the cross-CLI
// persistent memory store. Configuration (which embedder, dim,
// scope defaults) lives under Settings → Server → Memory; this
// page is the *runtime* view: browse / search / edit / delete
// what's actually stored.
export function MemoryPage() {
  return (
    <div className="flex-1 min-h-0 flex flex-col">
      <header className="px-6 py-4 border-b border-border bg-card/30">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-base font-medium flex items-center gap-2">
              <Brain className="size-4 text-accent" />
              Memory
            </h1>
            <p className="text-[12px] text-muted-foreground mt-0.5">
              Browse, search and edit memories agents have stored via the
              opendray-memory MCP server.
            </p>
          </div>
          <Button asChild variant="outline" size="sm" className="h-8 text-[11px]">
            <Link to="/settings">Configuration →</Link>
          </Button>
        </div>
      </header>
      <div className="flex-1 min-h-0 overflow-y-auto px-6 py-5">
        <div className="max-w-4xl">
          <MemoryInspector />
        </div>
      </div>
    </div>
  )
}
