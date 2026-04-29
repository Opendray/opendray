import { cn } from '@/lib/utils'
import type { SessionState } from '@/lib/types'

const styles: Record<SessionState, string> = {
  pending:
    'bg-state-idle/20 text-state-idle border-state-idle/30',
  running:
    'bg-state-running/20 text-state-running border-state-running/30',
  idle:
    'bg-state-idle/20 text-state-idle border-state-idle/30',
  ended:
    'bg-muted text-muted-foreground border-border',
}

const labels: Record<SessionState, string> = {
  pending: 'pending',
  running: 'running',
  idle: 'idle',
  ended: 'ended',
}

export function StatePill({
  state,
  exitCode,
  className,
}: {
  state: SessionState
  exitCode?: number
  className?: string
}) {
  // Non-zero exit on an ended session reads as failed.
  const isFailed = state === 'ended' && exitCode != null && exitCode !== 0

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 px-1.5 h-4 rounded-full border text-[9px] font-medium uppercase tracking-wide',
        isFailed
          ? 'bg-state-failed/20 text-state-failed border-state-failed/30'
          : styles[state],
        className,
      )}
    >
      <span
        className={cn(
          'size-1 rounded-full',
          isFailed
            ? 'bg-state-failed'
            : state === 'running'
              ? 'bg-state-running animate-pulse'
              : state === 'idle' || state === 'pending'
                ? 'bg-state-idle'
                : 'bg-muted-foreground',
        )}
      />
      {isFailed ? `failed (${exitCode})` : labels[state]}
    </span>
  )
}
