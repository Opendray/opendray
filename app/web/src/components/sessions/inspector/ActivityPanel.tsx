import { EventTimeline } from '@/components/activity/EventTimeline'
import type { Session } from '@/lib/types'

// ActivityPanel — Inspector tab. Shows the persisted lifecycle event
// timeline for this session (start / idle / restart / end). Backed
// by audit_log so it survives page refresh and gateway restarts.
export function ActivityPanel({ session }: { session: Session }) {
  return (
    <div className="flex flex-col gap-2">
      <p className="text-[11px] text-muted-foreground/70 px-1">
        Lifecycle events recorded for this session.
      </p>
      <EventTimeline
        subject={{ kind: 'session', id: session.id }}
        pageSize={50}
        dense
        emptyHint="Lifecycle events will appear here as the session runs."
      />
    </div>
  )
}
