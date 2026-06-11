import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ChevronRight,
  ChevronDown,
  FileText,
  Folder,
  Loader2,
  Download,
} from 'lucide-react'

import {
  listDir,
  fsDownloadURL,
  fsZipURL,
  triggerDownload,
} from '@/lib/fs'
import { useAuth } from '@/stores/auth'
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
  const token = useAuth((s) => s.token)

  const { data, isLoading, error } = useQuery({
    queryKey: ['fs', path],
    queryFn: () => listDir(path),
    enabled: open,
    staleTime: 10_000,
  })

  return (
    <div className="flex flex-col">
      <div className="group relative flex items-stretch">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          style={indent}
          className={cn(
            'flex-1 min-w-0 flex items-center gap-1 text-[12px] py-0.5 pr-8 rounded-sm',
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
        {token && (
          <DownloadIconButton
            ariaLabel={`Download ${name} as zip`}
            onActivate={() =>
              triggerDownload(fsZipURL(path, token), `${name || 'folder'}.zip`)
            }
          />
        )}
      </div>
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
              <div key={e.path} className="group relative flex items-stretch">
                <button
                  type="button"
                  onClick={() => onOpenFile(e.path)}
                  style={{ paddingLeft: `${(depth + 1) * 12 + 16}px` }}
                  className={cn(
                    'flex-1 min-w-0 flex items-center gap-1 text-[12px] py-0.5 pr-8 rounded-sm',
                    'hover:bg-card text-muted-foreground/90 hover:text-foreground text-left',
                  )}
                  title={e.path}
                >
                  <FileText className="size-3 shrink-0 opacity-60" />
                  <span className="truncate">{e.name}</span>
                </button>
                {token && (
                  <DownloadIconButton
                    ariaLabel={`Download ${e.name}`}
                    onActivate={() =>
                      triggerDownload(fsDownloadURL(e.path, token), e.name)
                    }
                  />
                )}
              </div>
            ),
          )}
        </div>
      )}
    </div>
  )
}

// DownloadIconButton overlays a small Download icon on the right edge
// of a tree row. Visible-on-hover (or always on touch via the row's
// group-hover and active fallbacks) so the tree stays scannable when
// the operator isn't reaching for a file. Anchored absolutely so the
// row's text isn't truncated to make space — the button sits over the
// row's right padding.
function DownloadIconButton({
  ariaLabel,
  onActivate,
}: {
  ariaLabel: string
  onActivate: () => void
}) {
  return (
    <button
      type="button"
      onClick={(e) => {
        // Stop the click from also triggering the row's open/view
        // action — the operator clicked the download icon, not the
        // row body.
        e.stopPropagation()
        onActivate()
      }}
      aria-label={ariaLabel}
      title={ariaLabel}
      className={cn(
        'absolute right-1 top-1/2 -translate-y-1/2',
        'size-5 flex items-center justify-center rounded-sm',
        'text-muted-foreground/70 hover:text-foreground hover:bg-border',
        'opacity-0 group-hover:opacity-100 focus-visible:opacity-100',
        'transition-opacity',
      )}
    >
      <Download className="size-3" />
    </button>
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
