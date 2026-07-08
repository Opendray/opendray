import { useRef, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  ChevronRight,
  ChevronDown,
  FileText,
  Folder,
  Loader2,
  Download,
  FolderPlus,
  Upload,
} from 'lucide-react'

import {
  listDir,
  makeDir,
  walkDropEntries,
  type WalkedFile,
  fsDownloadURL,
  fsZipURL,
  triggerDownload,
} from '@/lib/fs'
import { useAuth } from '@/stores/auth'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

import { FileViewerDialog } from './FileViewerDialog'
import { runUploads, type UploadItem } from './uploads'

interface FilesPanelProps {
  cwd: string
}

// FilesPanel renders a lazy file tree rooted at the session's cwd.
// Children are fetched via /api/v1/fs/list when a folder is expanded —
// no recursive crawl, so large repos stay fast. Clicking a file opens
// a modal viewer backed by /api/v1/fs/read.
export function FilesPanel({ cwd }: FilesPanelProps) {
  const [viewing, setViewing] = useState<string | null>(null)
  const [progress, setProgress] = useState<{ done: number; total: number } | null>(
    null,
  )
  const [dragging, setDragging] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const folderInputRef = useRef<HTMLInputElement>(null)
  const qc = useQueryClient()

  async function handleNewFolder() {
    const name = window.prompt('New folder name')?.trim()
    if (!name) return
    try {
      await makeDir(cwd, name)
      await qc.invalidateQueries({ queryKey: ['fs', cwd] })
      toast.success(`Created folder ${name}`)
    } catch (e) {
      toast.error('Could not create folder', {
        description: (e as Error).message,
      })
    }
  }

  // Drive an upload of walked files into a destination dir, showing
  // aggregate progress, refreshing every affected directory, and
  // reporting renames + failures.
  async function drive(walked: WalkedFile[], dir: string) {
    if (walked.length === 0) return
    const items: UploadItem[] = walked.map((w) => ({ ...w, dir }))
    setProgress({ done: 0, total: items.length })
    const { results, errors } = await runUploads(items, cwd, {
      onSettled: () => setProgress((p) => (p ? { ...p, done: p.done + 1 } : p)),
    })
    setProgress(null)

    // Refresh every directory that received a file (dir + each nested
    // parent created by the walk).
    const dirs = new Set<string>([dir])
    for (const it of items) {
      const parts = it.relpath.split('/').slice(0, -1)
      let acc = dir
      for (const seg of parts) {
        acc = `${acc}/${seg}`
        dirs.add(acc)
      }
    }
    await Promise.all(
      [...dirs].map((d) => qc.invalidateQueries({ queryKey: ['fs', d] })),
    )

    const renamed = results.filter((r) => r.renamed_from).length
    if (errors.length === 0) {
      toast.success(
        `Uploaded ${results.length} file${results.length === 1 ? '' : 's'}` +
          (renamed ? ` (${renamed} auto-renamed)` : ''),
      )
    } else {
      toast.error(
        `Uploaded ${results.length}, ${errors.length} failed`,
        { description: errors[0]?.error.message },
      )
    }
  }

  function onDropFiles(dir: string, dt: DataTransfer) {
    void walkDropEntries(dt)
      .then((w) => drive(w, dir))
      .catch((err) =>
        toast.error('Upload failed', {
          description: (err as Error).message,
        }),
      )
  }

  async function handleInputFiles(list: FileList | null, webkitRelative: boolean) {
    if (!list || list.length === 0) return
    const walked: WalkedFile[] = Array.from(list).map((file) => ({
      // The folder <input webkitdirectory> exposes webkitRelativePath
      // ("picked/dir/file.txt"); strip the top-level picked dir name so
      // the subtree lands under cwd, matching a drag of that folder.
      relpath: webkitRelative
        ? file.webkitRelativePath.split('/').slice(1).join('/') || file.name
        : file.name,
      file,
    }))
    await drive(walked, cwd)
  }

  return (
    <>
      <div className="flex items-center gap-1 px-1 pb-1">
        <Button variant="ghost" size="sm" onClick={handleNewFolder}>
          <FolderPlus className="size-3.5" />
          New folder
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => fileInputRef.current?.click()}
          disabled={progress != null}
        >
          <Upload className="size-3.5" />
          Upload
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => folderInputRef.current?.click()}
          disabled={progress != null}
        >
          <FolderPlus className="size-3.5" />
          Upload folder
        </Button>
        <input
          ref={fileInputRef}
          type="file"
          multiple
          className="hidden"
          onChange={(e) => {
            void handleInputFiles(e.target.files, false)
            e.target.value = ''
          }}
        />
        <input
          ref={folderInputRef}
          type="file"
          multiple
          className="hidden"
          // webkitdirectory/directory aren't in React's JSX types; the
          // `as any` spread is the standard escape hatch (a `// @ts-
          // expect-error` inside JSX props does not compile).
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          {...({ webkitdirectory: '', directory: '' } as any)}
          onChange={(e) => {
            void handleInputFiles(e.target.files, true)
            e.target.value = ''
          }}
        />
      </div>

      {progress && (
        <div className="px-2 pb-1">
          <div className="flex items-center gap-1 text-[11px] text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            Uploading {progress.done} of {progress.total}…
          </div>
          <div className="mt-0.5 h-1 w-full overflow-hidden rounded bg-border">
            <div
              className="h-full bg-accent transition-all"
              style={{
                width: `${Math.round((progress.done / progress.total) * 100)}%`,
              }}
            />
          </div>
        </div>
      )}

      <div
        onDragOver={(e) => {
          if (e.dataTransfer.types.includes('Files')) {
            e.preventDefault()
            e.dataTransfer.dropEffect = 'copy'
            setDragging(true)
          }
        }}
        onDragLeave={(e) => {
          if (e.currentTarget === e.target) setDragging(false)
        }}
        onDrop={(e) => {
          e.preventDefault()
          setDragging(false)
          void walkDropEntries(e.dataTransfer)
            .then((w) => drive(w, cwd))
            .catch((err) =>
              toast.error('Upload failed', {
                description: (err as Error).message,
              }),
            )
        }}
        className={cn(
          'flex flex-col gap-0.5 rounded-sm font-mono',
          dragging && 'ring-2 ring-accent/70',
        )}
      >
        <FsNode
          path={cwd}
          name={cwd}
          isRoot
          // downloadRoot is the cwd: the server-side download/zip
          // endpoints reject any resolved path that escapes it, so the
          // operator's reach matches what they can browse here.
          downloadRoot={cwd}
          onOpenFile={(p) => setViewing(p)}
          onDropFiles={onDropFiles}
        />
        {dragging && (
          <div className="px-2 py-1 text-[11px] text-muted-foreground">
            Drop files or folders here
          </div>
        )}
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
  // downloadRoot is the operator-allowed root the server uses to
  // confine download requests. Passed unchanged to every child so the
  // whole subtree shares the same confinement.
  downloadRoot: string
  onOpenFile: (path: string) => void
  onDropFiles: (dir: string, dt: DataTransfer) => void
}

function FsNode({
  path,
  name,
  isRoot = false,
  depth = 0,
  downloadRoot,
  onOpenFile,
  onDropFiles,
}: FsNodeProps) {
  const [open, setOpen] = useState(isRoot)
  const [rowDrag, setRowDrag] = useState(false)
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
      <div
        className={cn(
          'group relative flex items-stretch rounded-sm',
          rowDrag && 'ring-2 ring-accent/70',
        )}
        onDragOver={(e) => {
          if (e.dataTransfer.types.includes('Files')) {
            e.preventDefault()
            e.stopPropagation()
            e.dataTransfer.dropEffect = 'copy'
            setRowDrag(true)
          }
        }}
        onDragLeave={() => setRowDrag(false)}
        onDrop={(e) => {
          e.preventDefault()
          e.stopPropagation()
          setRowDrag(false)
          onDropFiles(path, e.dataTransfer)
        }}
      >
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
              triggerDownload(
                fsZipURL(path, downloadRoot, token),
                `${name || 'folder'}.zip`,
              )
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
                downloadRoot={downloadRoot}
                onOpenFile={onOpenFile}
                onDropFiles={onDropFiles}
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
                      triggerDownload(
                        fsDownloadURL(e.path, downloadRoot, token),
                        e.name,
                      )
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
