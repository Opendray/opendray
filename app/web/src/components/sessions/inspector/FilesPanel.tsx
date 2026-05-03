import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ChevronRight, ChevronDown, FileText, Folder, Loader2 } from 'lucide-react'

import { listDir } from '@/lib/fs'
import { cn } from '@/lib/utils'

import { FileViewerDialog } from './FileViewerDialog'

interface FilesPanelProps {
  cwd: string
}

// FilesPanel renders a lazy file tree rooted at the session's cwd.
// Children are fetched via /api/v1/fs/list when a folder is expanded —
// no recursive crawl, so large repos stay fast. Clicking a file opens
// a modal viewer backed by /api/v1/fs/read.
export function FilesPanel({ cwd }: FilesPanelProps) {
  const [viewing, setViewing] = useState<string | null>(null)

  return (
    <>
      <div className="flex flex-col gap-0.5 font-mono">
        <FsNode
          path={cwd}
          name={cwd}
          isRoot
          onOpenFile={(p) => setViewing(p)}
        />
      </div>
      <FileViewerDialog
        path={viewing}
        open={viewing != null}
        onOpenChange={(v) => !v && setViewing(null)}
      />
    </>
  )
}

interface FsNodeProps {
  path: string
  name: string
  isRoot?: boolean
  depth?: number
  onOpenFile: (path: string) => void
}

function FsNode({
  path,
  name,
  isRoot = false,
  depth = 0,
  onOpenFile,
}: FsNodeProps) {
  const [open, setOpen] = useState(isRoot)
  const indent = { paddingLeft: `${depth * 12}px` }

  const { data, isLoading, error } = useQuery({
    queryKey: ['fs', path],
    queryFn: () => listDir(path),
    enabled: open,
    staleTime: 10_000,
  })

  return (
    <div className="flex flex-col">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        style={indent}
        className={cn(
          'flex items-center gap-1 text-[12px] py-0.5 pr-1 rounded-sm',
          'hover:bg-card text-foreground/90 text-left',
        )}
        title={path}
      >
        {open ? (
          <ChevronDown className="size-3 shrink-0 opacity-60" />
        ) : (
          <ChevronRight className="size-3 shrink-0 opacity-60" />
        )}
        <Folder className="size-3 shrink-0 text-muted-foreground" />
        <span className="truncate">{isRoot ? trimRoot(name) : name}</span>
      </button>
      {open && (
        <div className="flex flex-col">
          {isLoading && (
            <div
              style={{ paddingLeft: `${(depth + 1) * 12 + 16}px` }}
              className="flex items-center gap-1 text-[11px] text-muted-foreground py-0.5"
            >
              <Loader2 className="size-3 animate-spin" />
              loading…
            </div>
          )}
          {error && (
            <div
              style={{ paddingLeft: `${(depth + 1) * 12 + 16}px` }}
              className="text-[11px] text-state-failed py-0.5"
            >
              {(error as Error).message}
            </div>
          )}
          {data?.entries.map((e) =>
            e.is_dir ? (
              <FsNode
                key={e.path}
                path={e.path}
                name={e.name}
                depth={depth + 1}
                onOpenFile={onOpenFile}
              />
            ) : (
              <button
                key={e.path}
                type="button"
                onClick={() => onOpenFile(e.path)}
                style={{ paddingLeft: `${(depth + 1) * 12 + 16}px` }}
                className={cn(
                  'flex items-center gap-1 text-[12px] py-0.5 pr-1 rounded-sm',
                  'hover:bg-card text-muted-foreground/90 hover:text-foreground text-left',
                )}
                title={e.path}
              >
                <FileText className="size-3 shrink-0 opacity-60" />
                <span className="truncate">{e.name}</span>
              </button>
            ),
          )}
        </div>
      )}
    </div>
  )
}

// Display the cwd as its trailing 1-2 segments at the root row, full
// path stays in the title attribute so hover gives the user the
// absolute location.
function trimRoot(p: string): string {
  const parts = p.split('/').filter(Boolean)
  if (parts.length <= 2) return '/' + parts.join('/')
  return '…/' + parts.slice(-2).join('/')
}
