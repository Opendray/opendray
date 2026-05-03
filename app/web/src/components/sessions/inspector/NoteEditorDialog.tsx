import { useEffect, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { FileText, Trash2, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { deleteNote } from '@/lib/notes'

import { NoteEditor } from './NoteEditor'

interface NoteEditorDialogProps {
  path: string | null
  open: boolean
  onOpenChange: (v: boolean) => void
  // onDeleted fires after a successful delete so the caller can update
  // any list it's showing.
  onDeleted?: (path: string) => void
}

// NoteEditorDialog opens a vault note in a wide modal — preview by
// default (project docs are read-mostly), with one click to switch to
// source. The full NoteEditor is rendered inside, so saves go through
// the same debounced auto-save path as the inline editor.
export function NoteEditorDialog({
  path,
  open,
  onOpenChange,
  onDeleted,
}: NoteEditorDialogProps) {
  const qc = useQueryClient()

  // currentPath swaps the dialog's contents in-place when the user
  // clicks a wiki-link in preview / a backlink row — same modal, new
  // note. We seed it from the prop and keep it in sync when the
  // parent opens the dialog with a new note.
  const [currentPath, setCurrentPath] = useState<string | null>(path)
  useEffect(() => {
    if (open) setCurrentPath(path)
  }, [open, path])

  const remove = useMutation({
    mutationFn: () => deleteNote(currentPath!),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['notes-list'] })
      qc.invalidateQueries({ queryKey: ['note', currentPath] })
      toast.success('Note deleted')
      onDeleted?.(currentPath!)
      onOpenChange(false)
    },
    onError: (err: Error) =>
      toast.error('Delete failed', { description: err.message }),
  })

  const filename = currentPath
    ? currentPath.split('/').pop() ?? currentPath
    : ''

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        // Override Radix Dialog's default `grid` with a constrained
        // flex-column shell so the body (NoteEditor in fillParent
        // mode) scrolls internally instead of pushing the dialog off
        // the viewport.
        className="max-w-[min(92vw,1000px)] w-[min(92vw,1000px)] h-[min(85vh,800px)] gap-2 flex flex-col"
      >
        <DialogHeader className="shrink-0">
          <div className="flex items-start gap-2 min-w-0">
            <FileText className="size-4 mt-0.5 text-muted-foreground shrink-0" />
            <div className="flex flex-col min-w-0 flex-1">
              <DialogTitle className="truncate">{filename}</DialogTitle>
              <span
                className="text-[11px] text-muted-foreground/70 font-mono truncate"
                title={currentPath ?? ''}
              >
                {currentPath}
              </span>
            </div>
            {currentPath && currentPath !== path && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setCurrentPath(path)}
                className="text-[11px] shrink-0"
                title={`Back to ${path}`}
              >
                ← back
              </Button>
            )}
            {currentPath && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  if (
                    confirm(
                      `Delete note "${currentPath}"? This removes the .md file from the vault.`,
                    )
                  ) {
                    remove.mutate()
                  }
                }}
                disabled={remove.isPending}
                className="text-[11px] gap-1 text-muted-foreground hover:text-destructive shrink-0"
              >
                {remove.isPending ? (
                  <Loader2 className="size-3 animate-spin" />
                ) : (
                  <Trash2 className="size-3" />
                )}
                Delete
              </Button>
            )}
          </div>
        </DialogHeader>

        {currentPath && open && (
          <NoteEditor
            // remount when path changes so the autosave / draft state
            // resets cleanly.
            key={currentPath}
            path={currentPath}
            initialMode="preview"
            fillParent
            showBacklinks
            onOpenLink={(p) => setCurrentPath(p)}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}
