import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowUp,
  Check,
  Folder,
  FolderPlus,
  Home,
  Loader2,
  RefreshCw,
} from 'lucide-react'
import { toast } from 'sonner'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { listDir, makeDir, getHomeDir } from '@/lib/fs'

interface FileBrowserDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialPath?: string
  onSelect: (path: string) => void
}

export function FileBrowserDialog({
  open,
  onOpenChange,
  initialPath,
  onSelect,
}: FileBrowserDialogProps) {
  const qc = useQueryClient()
  const [path, setPath] = useState<string>(initialPath ?? '')
  const [pathInput, setPathInput] = useState<string>(initialPath ?? '')
  const [showMkdir, setShowMkdir] = useState(false)
  const [newName, setNewName] = useState('')

  // On open, resolve initial path: prefer initialPath, fall back to
  // server-side home dir lookup (works regardless of which host the
  // gateway is running on).
  useEffect(() => {
    if (!open) return
    if (initialPath) {
      setPath(initialPath)
      setPathInput(initialPath)
      return
    }
    if (!path) {
      getHomeDir()
        .then((p) => {
          setPath(p)
          setPathInput(p)
        })
        .catch(() => {})
    }
  }, [open, initialPath, path])

  const list = useQuery({
    queryKey: ['fs', path],
    queryFn: () => listDir(path),
    enabled: open && !!path,
  })

  // Sync path input box with the resolved canonical path returned by
  // the server (so ~/foo → /Users/.../foo after navigation).
  useEffect(() => {
    if (list.data?.path) setPathInput(list.data.path)
  }, [list.data?.path])

  const mkdir = useMutation({
    mutationFn: () => makeDir(path, newName.trim()),
    onSuccess: (created) => {
      qc.invalidateQueries({ queryKey: ['fs', path] })
      toast.success('Directory created')
      setNewName('')
      setShowMkdir(false)
      // Step into the new directory.
      setPath(created)
    },
    onError: (e: Error) =>
      toast.error('Mkdir failed', { description: e.message }),
  })

  const goHome = async () => {
    try {
      const home = await getHomeDir()
      setPath(home)
    } catch (e) {
      toast.error('Failed to read home', {
        description: (e as Error).message,
      })
    }
  }

  const handleSelect = () => {
    const target = pathInput.trim()
    if (!target) return
    onSelect(target)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[640px]">
        <DialogHeader>
          <DialogTitle>Choose working directory</DialogTitle>
          <DialogDescription>
            Browse the gateway host's filesystem and pick a folder.
          </DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-1.5">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="size-7"
                onClick={() => {
                  if (list.data?.parent) setPath(list.data.parent)
                }}
                disabled={!list.data?.parent}
                aria-label="Parent directory"
              >
                <ArrowUp className="size-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Parent directory</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="size-7"
                onClick={goHome}
                aria-label="Home directory"
              >
                <Home className="size-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Home directory</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="size-7"
                onClick={() =>
                  qc.invalidateQueries({ queryKey: ['fs', path] })
                }
                aria-label="Refresh"
              >
                <RefreshCw className="size-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Refresh</TooltipContent>
          </Tooltip>
          <Input
            value={pathInput}
            onChange={(e) => setPathInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                setPath(pathInput.trim())
              }
            }}
            placeholder="/Users/you/projects"
            className="flex-1 font-mono text-[12px]"
          />
        </div>

        <div className="rounded-md border border-border h-[280px] overflow-hidden">
          <ScrollArea className="h-full">
            {list.isLoading ? (
              <div className="flex items-center gap-2 p-3 text-[12px] text-muted-foreground">
                <Loader2 className="size-3.5 animate-spin" />
                Loading…
              </div>
            ) : list.error ? (
              <div className="p-3 text-[12px] text-destructive">
                {(list.error as Error).message}
              </div>
            ) : (list.data?.entries ?? []).length === 0 ? (
              <div className="p-3 text-[12px] text-muted-foreground italic">
                Empty directory.
              </div>
            ) : (
              <ul className="py-1">
                {(list.data?.entries ?? [])
                  .filter((e) => e.is_dir)
                  .map((e) => (
                    <li key={e.path}>
                      <button
                        type="button"
                        onClick={() => setPath(e.path)}
                        className="w-full flex items-center gap-2 px-3 py-1.5 text-left text-[12px] hover:bg-card transition-colors"
                      >
                        <Folder className="size-3.5 text-muted-foreground/70 shrink-0" />
                        <span className="truncate">{e.name}</span>
                      </button>
                    </li>
                  ))}
              </ul>
            )}
          </ScrollArea>
        </div>

        {showMkdir ? (
          <div className="flex items-center gap-1.5">
            <Input
              autoFocus
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  if (newName.trim()) mkdir.mutate()
                }
                if (e.key === 'Escape') setShowMkdir(false)
              }}
              placeholder="new-folder-name"
              className="flex-1 text-[12px]"
            />
            <Button
              variant="accent"
              size="sm"
              disabled={!newName.trim() || mkdir.isPending}
              onClick={() => mkdir.mutate()}
            >
              {mkdir.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Check className="size-3.5" />
              )}
              Create
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowMkdir(false)}
              disabled={mkdir.isPending}
            >
              Cancel
            </Button>
          </div>
        ) : (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowMkdir(true)}
            className="self-start text-[11px] gap-1"
          >
            <FolderPlus className="size-3.5" />
            New folder
          </Button>
        )}

        <DialogFooter>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="accent"
            size="sm"
            onClick={handleSelect}
            disabled={!pathInput.trim()}
          >
            Use this folder
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
