import { useImperativeHandle, useMemo, useState, forwardRef } from 'react'
import {
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
export const NotesTreeView = forwardRef<NotesTreeViewHandle, NotesTreeViewProps>(
  function NotesTreeView(
    { notes, selected, onSelect, initialExpanded },
    ref,
  ) {
  const { t } = useTranslation()
  const tree = useMemo(() => buildTree(notes), [notes])

  // Auto-expand the path leading to the selected note so it stays
  // visible across re-renders / search clears. Other folders default
  // to whatever caller passed (or empty = collapsed).
  const autoExpand = useMemo(() => {
    const set = new Set<string>(initialExpanded ?? new Set<string>())
    if (selected) {
      const parts = selected.split('/').slice(0, -1)
      let cur = ''
      for (const p of parts) {
        cur = cur ? `${cur}/${p}` : p
        set.add(cur)
      }
    }
    return set
  }, [tree, selected, initialExpanded])

  const [expanded, setExpanded] = useState<Set<string>>(autoExpand)

  // Re-merge when autoExpand changes (e.g. selected note in a fresh
  // folder branch). Preserves user's explicit toggles.
  useMemoSync(autoExpand, expanded, setExpanded)

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
          />
        ))
      )}
    </div>
  )
})

function TreeRow({
  node,
  depth,
  expanded,
  setExpanded,
  selected,
  onSelect,
}: {
  node: TreeNode
  depth: number
  expanded: Set<string>
  setExpanded: (s: Set<string>) => void
  selected: string | null
  onSelect: (path: string) => void
}) {
  const indent = { paddingLeft: `${depth * 12 + 4}px` }
  const isOpen = expanded.has(node.path)

  if (node.isDir) {
    return (
      <div className="flex flex-col">
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
            'flex items-center gap-1 py-0.5 pr-1 rounded-sm text-left',
            'hover:bg-card text-foreground/85',
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
              />
            ))}
          </div>
        )}
      </div>
    )
  }

  const isSelected = selected === node.path
  return (
    <button
      type="button"
      onClick={() => onSelect(node.path)}
      style={indent}
      className={cn(
        'flex items-center gap-1 py-0.5 pr-1 rounded-sm text-left',
        'hover:bg-card',
        isSelected
          ? 'bg-card text-foreground border-l-2 border-state-running'
          : 'text-muted-foreground/90',
      )}
      title={node.path}
    >
      <span className="size-3 shrink-0" />
      <FileText className="size-3 shrink-0 opacity-60" />
      <span className="truncate">{node.name}</span>
    </button>
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

// useMemoSync merges newly-required expansions into the existing user
// state without dropping toggles the user made. React doesn't expose
// a clean "merge into setState if value changed" so we compare via
// JSON serialisation of the sorted paths — cheap for sets of <100.
function useMemoSync(
  next: Set<string>,
  current: Set<string>,
  set: (s: Set<string>) => void,
) {
  const nextKey = JSON.stringify([...next].sort())
  const currentKey = JSON.stringify([...current].sort())
  // Effect-less sync: only fire when next has additions current doesn't.
  // Avoids infinite loops while still picking up auto-expand.
  if (nextKey !== currentKey) {
    let changed = false
    const merged = new Set(current)
    for (const p of next) {
      if (!merged.has(p)) {
        merged.add(p)
        changed = true
      }
    }
    if (changed) {
      // queueMicrotask defers to after render so React doesn't warn.
      queueMicrotask(() => set(merged))
    }
  }
}
