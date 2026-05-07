// Bottom tab bar — lets the user switch between top-level mobile
// surfaces (Sessions, Memory, Notes, Activity) without navigating
// through a hamburger menu. Mobile UX standard.
//
// Sits above the home indicator via the `pb-safe` Tailwind utility
// added in #27. Active-tab uses the accent color.
//
// Uses lucide-react icons (SVG) instead of Unicode glyphs because
// iOS 26.3 / WKWebView falls back to .notdef ([?] boxes) for the
// uncommon symbol-block code points we'd otherwise rely on.

import {
  Activity as ActivityIcon,
  Brain,
  FileText,
  MoreHorizontal,
  Terminal,
  type LucideIcon,
} from 'lucide-react'

export type Tab = 'sessions' | 'memory' | 'notes' | 'activity' | 'more'

interface Props {
  active: Tab
  onChange: (tab: Tab) => void
}

const TABS: { id: Tab; label: string; icon: LucideIcon }[] = [
  { id: 'sessions', label: 'Sessions', icon: Terminal },
  { id: 'memory', label: 'Memory', icon: Brain },
  { id: 'notes', label: 'Notes', icon: FileText },
  { id: 'activity', label: 'Activity', icon: ActivityIcon },
  { id: 'more', label: 'More', icon: MoreHorizontal },
]

export function BottomTabBar({ active, onChange }: Props) {
  return (
    <nav className="border-t border-border bg-card pb-safe">
      <ul className="flex">
        {TABS.map((t) => {
          const isActive = t.id === active
          const Icon = t.icon
          return (
            <li key={t.id} className="flex-1">
              <button
                type="button"
                onClick={() => onChange(t.id)}
                className={`w-full flex flex-col items-center gap-1 py-1.5 px-1 text-[10px] tracking-wide select-none transition-colors ${
                  isActive
                    ? 'text-accent'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                <Icon className="w-5 h-5" />
                <span>{t.label}</span>
              </button>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
