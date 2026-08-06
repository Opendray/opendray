import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { ExternalLink } from 'lucide-react'

import { Button } from '@/components/ui/button'
import type { Session } from '@/lib/types'

import { CanvasStage } from './CanvasStage'

interface CanvasPanelProps {
  session: Session
}

// CanvasPanel — the Inspector "Canvas" tab. A compact entry point onto the
// shared CanvasStage, plus a "pop out" button that opens the same canvas in a
// full, standalone window (the inspector column is too narrow for a real design
// preview) — leaving the terminal untouched so the agent conversation and the
// large preview stay usable side by side.
export function CanvasPanel({ session }: CanvasPanelProps) {
  const { t } = useTranslation()

  const popOut = useCallback(() => {
    const base = import.meta.env.BASE_URL.replace(/\/$/, '') // "" dev, "/admin" prod
    const url =
      `${base}/canvas?session=${encodeURIComponent(session.id)}` +
      `&cwd=${encodeURIComponent(session.cwd)}`
    // A named target reuses the same window for this session, so re-clicking
    // focuses the existing pop-out instead of stacking windows.
    window.open(url, `opendray-canvas-${session.id}`, 'popup,width=1280,height=900')
  }, [session.id, session.cwd])

  const popoutButton = (
    <Button size="sm" variant="ghost" onClick={popOut} title={t('web.sessions.inspector.canvas.popOutHint')}>
      <ExternalLink className="size-3.5" />
      {t('web.sessions.inspector.canvas.popOut')}
    </Button>
  )

  return (
    <CanvasStage
      sessionId={session.id}
      cwd={session.cwd}
      variant="panel"
      toolbarExtra={popoutButton}
    />
  )
}
