// Bottom tab bar — lets the user switch between top-level mobile
// surfaces (Sessions, Memory, Notes, Activity) without navigating
// through a hamburger menu. Mobile UX standard.
//
// Sits above the home indicator via the `pb-safe` Tailwind utility
// added in #27. Active-tab indicator is the accent color underline.

export type Tab = 'sessions' | 'memory' | 'notes' | 'activity'

interface Props {
  active: Tab
  onChange: (tab: Tab) => void
}

const TABS: { id: Tab; label: string; icon: string }[] = [
  { id: 'sessions', label: 'Sessions', icon: '▢' },
  { id: 'memory', label: 'Memory', icon: '◇' },
  { id: 'notes', label: 'Notes', icon: '☰' },
  { id: 'activity', label: 'Activity', icon: '⏱' },
]

export function BottomTabBar({ active, onChange }: Props) {
  return (
    <nav className="border-t border-border bg-card pb-safe">
      <ul className="flex">
        {TABS.map((t) => {
          const isActive = t.id === active
          return (
            <li key={t.id} className="flex-1">
              <button
                type="button"
                onClick={() => onChange(t.id)}
                className={`w-full flex flex-col items-center gap-0.5 py-2 px-1 text-[10px] tracking-wide select-none transition-colors ${
                  isActive
                    ? 'text-accent'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                <span className="text-base leading-none">{t.icon}</span>
                <span>{t.label}</span>
              </button>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
