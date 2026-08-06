import { useSearch } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { CanvasStage } from '@/components/sessions/inspector/CanvasStage'

// CanvasPopoutPage — the standalone, full-window canvas. Opened via
// window.open from the Inspector's Canvas tab so a design gets the whole screen
// (or a second monitor) while the main window keeps the terminal + agent chat.
// Auth/theme/locale come from the shared root (main.tsx) since it's a same-
// origin window; feedback posts straight back to the session via the API.
export function CanvasPopoutPage() {
  const { t } = useTranslation()
  const search = useSearch({ strict: false }) as { session?: string; cwd?: string }
  const sessionId = search.session || ''
  const cwd = search.cwd || ''

  if (!sessionId || !cwd) {
    return (
      <div className="p-6 text-sm text-muted-foreground">
        {t('web.sessions.inspector.canvas.popOutMissing')}
      </div>
    )
  }

  return (
    <div className="h-screen w-screen flex flex-col bg-background text-foreground p-3">
      <div className="shrink-0 pb-2 flex items-center gap-2 min-w-0">
        <span className="text-sm font-medium">
          {t('web.sessions.inspector.canvas.popOutTitle')}
        </span>
        <code className="text-[11px] text-muted-foreground truncate" title={cwd}>
          {cwd}
        </code>
      </div>
      <div className="flex-1 min-h-0">
        <CanvasStage sessionId={sessionId} cwd={cwd} variant="popout" />
      </div>
    </div>
  )
}
