import {
  forwardRef,
  memo,
  useEffect,
  useImperativeHandle,
  useMemo,
  useState,
} from 'react'
import {
  BookOpen,
  ChevronRight,
  ChevronDown,
  Folder,
  FileText,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
import type { Note } from '@/lib/notes'

interface NotesTreeViewProps {
  notes: Note[]
  selected?: string | null
  onSelect: (path: string) => void
  // initialExpanded controls which folders are open on first render.
  // Defaults to "all collapsed" so a vault with hundreds of folders
  // stays scannable. Pass a populated set to override.
  initialExpanded?: Set<string>
  // renderFileAction adds a trailing control to each file row (rename,
  // delete…). Optional so the plain read-only tree stays unchanged;
  // rendered outside the row's own button, since nesting a button in a
  // button is invalid HTML and swallows the click.
  renderFileAction?: (path: string) => React.ReactNode
  // onOpenFolderIndex, when given, turns a folder that holds a
  // README/index note into something you can open: the chevron still
  // expands, and the extra control reads the folder's own page. Return
  // undefined for folders without one and no control is rendered.
  folderIndexPath?: (dir: string) => string | undefined
  onOpenFolderIndex?: (path: string) => void
}

// Imperative handle exposed via ref so the parent's toolbar can drive
// "Expand all" / "Collapse all" without lifting tree state up.
export interface NotesTreeViewHandle {
  expandAll(): void
  collapseAll(): void
}

interface TreeNode {
  name: string
  path: string // vault-relative; folders end without trailing slash
  isDir: boolean
  children: Map<string, TreeNode>
  note?: Note
}

// NotesTreeView renders a hierarchical view of all .md files under
// the vault. Built by chunking each note's path into segments and
// merging into a tree on the fly — no recursive backend listing
// needed (the /notes/list endpoint already returns the flat list).
// memo, because the page that hosts this re-renders on every edit to
// the open document. Re-walking and re-rendering every row of a vault
// for each keystroke is work nobody asked for: the tree only changes
// when the note list or the selection does.
export const NotesTreeView = memo(
  forwardRef<NotesTreeViewHandle, NotesTreeViewProps>(
  function NotesTreeView(
    {
      notes,
      selected,
      onSelect,
      initialExpanded,
      renderFileAction,
      folderIndexPath,
      onOpenFolderIndex,
    },
    ref,
  ) {
  const { t } = useTranslation()
  const tree = useMemo(() => buildTree(notes), [notes])

  const [expanded, setExpanded] = useState<Set<string>>(
    () => new Set(initialExpanded ?? []),
  )

  // Reveal the selected note by opening its ancestors — ONCE per
  // selection, in an effect.
  //
  // This used to re-merge on every render, which made the folder
  // holding the selected note impossible to close: the collapse landed,
  // the next render put it straight back, and "Collapse all" left that
  // one branch open. Auto-expand is a response to the selection
  // changing, not a rule about what must stay open.
  useEffect(() => {
    if (!selected) return
    const parts = selected.split('/').slice(0, -1)
    if (parts.length === 0) return
    setExpanded((prev) => {
      const next = new Set(prev)
      let cur = ''
      let changed = false
      for (const p of parts) {
        cur = cur ? `${cur}/${p}` : p
        if (!next.has(cur)) {
          next.add(cur)
          changed = true
        }
      }
      return changed ? next : prev
    })
  }, [selected])

  // Expose imperative expand/collapse-all so the parent toolbar can
  // hit them without us lifting the expanded state up.
  useImperativeHandle(
    ref,
    () => ({
      expandAll: () => {
        const all = new Set<string>()
        const walk = (n: TreeNode) => {
          if (n.isDir && n.path) all.add(n.path)
          for (const c of n.children.values()) walk(c)
        }
        walk(tree)
        setExpanded(all)
      },
      collapseAll: () => setExpanded(new Set()),
    }),
    [tree],
  )

  return (
    <div className="flex flex-col font-mono text-[12px]">
      {tree.children.size === 0 ? (
        <div className="px-2 py-3 text-[11px] text-muted-foreground/60">
          {t('web.notes.tree.empty')}
        </div>
      ) : (
        Array.from(tree.children.values()).map((child) => (
          <TreeRow
            key={child.path}
            node={child}
            depth={0}
            expanded={expanded}
            setExpanded={setExpanded}
            selected={selected ?? null}
            onSelect={onSelect}
            renderFileAction={renderFileAction}
            folderIndexPath={folderIndexPath}
            onOpenFolderIndex={onOpenFolderIndex}
          />
        ))
      )}
    </div>
  )
}),
)

function TreeRow({
  node,
  depth,
  expanded,
  setExpanded,
  selected,
  onSelect,
  renderFileAction,
  folderIndexPath,
  onOpenFolderIndex,
}: {
  node: TreeNode
  depth: number
  expanded: Set<string>
  setExpanded: (s: Set<string>) => void
  selected: string | null
  onSelect: (path: string) => void
  renderFileAction?: (path: string) => React.ReactNode
  folderIndexPath?: (dir: string) => string | undefined
  onOpenFolderIndex?: (path: string) => void
}) {
  const { t } = useTranslation()
  const indent = { paddingLeft: `${depth * 12 + 4}px` }
  const isOpen = expanded.has(node.path)

  if (node.isDir) {
    const indexPath = folderIndexPath?.(node.path)
    return (
      <div className="flex flex-col">
        <div className="group flex items-center rounded-sm hover:bg-card">
          <button
            type="button"
            onClick={() => {
              const next = new Set(expanded)
              if (isOpen) next.delete(node.path)
              else next.add(node.path)
              setExpanded(next)
            }}
            style={indent}
            className={cn(
              'flex min-w-0 flex-1 items-center gap-1 py-0.5 pr-1 text-left',
              'text-foreground/85',
            )}
            title={node.path}
          >
            {isOpen ? (
              <ChevronDown className="size-3 shrink-0 opacity-60" />
            ) : (
              <ChevronRight className="size-3 shrink-0 opacity-60" />
            )}
            <Folder className="size-3 shrink-0 text-muted-foreground" />
            <span className="truncate">{node.name}</span>
            <span className="ml-1 text-[10px] text-muted-foreground/50">
              {node.children.size}
            </span>
          </button>
          {indexPath && onOpenFolderIndex && (
            <button
              type="button"
              onClick={() => onOpenFolderIndex(indexPath)}
              className="shrink-0 px-1 text-muted-foreground hover:text-foreground"
              title={`${t('web.sessions.inspector.vaultPanel.openFolderIndexTitle')} — ${indexPath}`}
            >
              <BookOpen className="size-3" />
            </button>
          )}
        </div>
        {isOpen && (
          <div className="flex flex-col">
            {Array.from(node.children.values()).map((child) => (
              <TreeRow
                key={child.path}
                node={child}
                depth={depth + 1}
                expanded={expanded}
                setExpanded={setExpanded}
                selected={selected}
                onSelect={onSelect}
                renderFileAction={renderFileAction}
                folderIndexPath={folderIndexPath}
                onOpenFolderIndex={onOpenFolderIndex}
              />
            ))}
          </div>
        )}
      </div>
    )
  }

  const isSelected = selected === node.path
  const action = renderFileAction?.(node.path)
  return (
    <div
      className={cn(
        'group flex items-center rounded-sm',
        isSelected
          ? 'bg-card border-l-2 border-state-running'
          : 'hover:bg-card',
      )}
    >
      <button
        type="button"
        onClick={() => onSelect(node.path)}
        style={indent}
        className={cn(
          'flex min-w-0 flex-1 items-center gap-1 py-0.5 pr-1 text-left',
          isSelected ? 'text-foreground' : 'text-muted-foreground/90',
        )}
        title={node.path}
      >
        <span className="size-3 shrink-0" />
        <FileText className="size-3 shrink-0 opacity-60" />
        <span className="truncate">{node.name}</span>
      </button>
      {action && (
        <div className="shrink-0 pr-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
          {action}
        </div>
      )}
    </div>
  )
}

function buildTree(notes: Note[]): TreeNode {
  const root: TreeNode = {
    name: '',
    path: '',
    isDir: true,
    children: new Map(),
  }
  // Folders are sorted before files within each level so the tree
  // reads top-down from broad → specific. Pre-sort the input by path
  // so the merge order is deterministic.
  const sorted = [...notes].sort((a, b) => a.path.localeCompare(b.path))
  for (const n of sorted) {
    insert(root, n)
  }
  // Sort children: dirs first, then by name. Done after merge so each
  // level can reorder cheaply.
  walkSort(root)
  return root
}

function insert(root: TreeNode, n: Note) {
  const parts = n.path.split('/')
  let cur = root
  for (let i = 0; i < parts.length; i++) {
    const name = parts[i]
    const isLeaf = i === parts.length - 1
    const path = parts.slice(0, i + 1).join('/')
    let next = cur.children.get(name)
    if (!next) {
      next = {
        name,
        path,
        isDir: !isLeaf,
        children: new Map(),
      }
      if (isLeaf) next.note = n
      cur.children.set(name, next)
    }
    cur = next
  }
}

function walkSort(node: TreeNode) {
  if (!node.children.size) return
  const entries = Array.from(node.children.entries())
  entries.sort((a, b) => {
    if (a[1].isDir !== b[1].isDir) return a[1].isDir ? -1 : 1
    return a[0].localeCompare(b[0])
  })
  node.children = new Map(entries)
  for (const [, child] of node.children) walkSort(child)
}

