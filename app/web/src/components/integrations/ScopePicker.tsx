import { Check } from 'lucide-react'

import { cn } from '@/lib/utils'
import { SCOPE_GROUPS, SCOPE_INFO, type ScopeGroup } from '@/lib/scopes'

interface ScopePickerProps {
  selected: string[]
  onChange: (next: string[]) => void
  /** Optional intro text rendered above the picker. */
  intro?: string
}

// ScopePicker renders every scope in lib/scopes.SCOPE_INFO, grouped
// by topic, with a human-readable title + description per row.
//
// Used by both Register and Edit integration dialogs so operators
// don't have to memorise the meaning of "event:subscribe:session.*".
export function ScopePicker({ selected, onChange, intro }: ScopePickerProps) {
  const toggle = (id: string) => {
    onChange(
      selected.includes(id)
        ? selected.filter((x) => x !== id)
        : [...selected, id],
    )
  }

  return (
    <div className="flex flex-col gap-4">
      {intro && (
        <p className="text-[11px] text-muted-foreground/80 leading-snug">
          {intro}
        </p>
      )}
      {SCOPE_GROUPS.map((g) => {
        const items = SCOPE_INFO.filter((s) => s.group === g.id)
        if (items.length === 0) return null
        return <Group key={g.id} group={g} items={items} selected={selected} onToggle={toggle} />
      })}
    </div>
  )
}

function Group({
  group,
  items,
  selected,
  onToggle,
}: {
  group: { id: ScopeGroup; label: string; blurb: string }
  items: typeof SCOPE_INFO
  selected: string[]
  onToggle: (id: string) => void
}) {
  return (
    <section className="flex flex-col gap-2">
      <div>
        <h4 className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground/70">
          {group.label}
        </h4>
        <p className="text-[11px] text-muted-foreground/60 mt-0.5">
          {group.blurb}
        </p>
      </div>
      <div className="flex flex-col gap-1.5">
        {items.map((s) => {
          const isOn = selected.includes(s.id)
          return (
            <button
              key={s.id}
              type="button"
              onClick={() => onToggle(s.id)}
              className={cn(
                'flex items-start gap-3 text-left rounded-md border px-3 py-2 transition-colors',
                isOn
                  ? 'border-accent/50 bg-accent/5'
                  : 'border-border hover:border-foreground/20 hover:bg-card/50',
              )}
            >
              <span
                className={cn(
                  'mt-0.5 size-4 shrink-0 rounded-sm border flex items-center justify-center',
                  isOn
                    ? 'bg-accent border-accent text-accent-foreground'
                    : 'border-border',
                )}
                aria-hidden
              >
                {isOn && <Check className="size-3" />}
              </span>
              <span className="flex-1 min-w-0">
                <span className="flex items-baseline gap-2 flex-wrap">
                  <span className="text-[12.5px] font-medium">{s.title}</span>
                  <code className="text-[10.5px] text-muted-foreground/80 font-mono">
                    {s.id}
                  </code>
                </span>
                <p className="text-[11px] text-muted-foreground/80 leading-snug mt-0.5">
                  {s.description}
                </p>
              </span>
            </button>
          )
        })}
      </div>
    </section>
  )
}
