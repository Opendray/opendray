import { useEffect, useState } from 'react'

import { OnboardingScreen } from './screens/OnboardingScreen'
import { LoginScreen } from './screens/LoginScreen'
import { SessionsScreen } from './screens/SessionsScreen'
import { SessionDetailScreen } from './screens/SessionDetailScreen'
import { MemoryScreen } from './screens/MemoryScreen'
import { NotesScreen } from './screens/NotesScreen'
import { ActivityScreen } from './screens/ActivityScreen'
import { BottomTabBar, type Tab } from './components/BottomTabBar'
import { type SessionSummary } from './lib/api'
import {
  type StoredPrefs,
  clearAll,
  clearAuth,
  getPrefs,
  tokenExpired,
} from './lib/storage'

type AppState = 'loading' | 'onboarding' | 'login' | 'home' | 'session'

// Top-level state machine. Drives which screen renders based on what
// we have persisted in Capacitor Preferences:
//
//   serverURL absent              → onboarding
//   serverURL set, no/expired tok → login
//   serverURL set, valid token    → home (with 4 tabs)
//   home tab + tap session card   → session detail (full-screen)
export function App() {
  const [state, setState] = useState<AppState>('loading')
  const [prefs, setPrefs] = useState<StoredPrefs | null>(null)
  const [activeSession, setActiveSession] = useState<SessionSummary | null>(null)
  const [tab, setTab] = useState<Tab>('sessions')

  useEffect(() => {
    void bootstrap()
  }, [])

  async function bootstrap() {
    const p = await getPrefs()
    setPrefs(p)
    if (!p.serverURL) {
      setState('onboarding')
      return
    }
    if (!p.token || tokenExpired(p.expiresAt)) {
      setState('login')
      return
    }
    setState('home')
  }

  if (state === 'loading') {
    return (
      <div className="min-h-screen bg-background text-foreground flex items-center justify-center">
        <div className="text-sm text-muted-foreground">Loading…</div>
      </div>
    )
  }

  if (state === 'onboarding') {
    return (
      <OnboardingScreen
        onConnected={(url) => {
          setPrefs((prev) => ({
            serverURL: url,
            token: prev?.token ?? null,
            expiresAt: prev?.expiresAt ?? null,
            username: prev?.username ?? null,
          }))
          setState('login')
        }}
      />
    )
  }

  if (state === 'login') {
    return (
      <LoginScreen
        serverURL={prefs!.serverURL!}
        onAuthed={async () => {
          // setAuth has already written; re-read so home tabs see the
          // canonical values.
          const fresh = await getPrefs()
          setPrefs(fresh)
          setState('home')
        }}
        onChangeServer={async () => {
          await clearAll()
          setPrefs(null)
          setState('onboarding')
        }}
      />
    )
  }

  const onClearAuthAndReturnToLogin = async () => {
    await clearAuth()
    setPrefs((prev) =>
      prev ? { ...prev, token: null, expiresAt: null, username: null } : null,
    )
    setActiveSession(null)
    setState('login')
  }

  // Session detail is full-screen — no bottom tab bar — so the
  // terminal can use every available pixel.
  if (state === 'session' && activeSession) {
    return (
      <SessionDetailScreen
        serverURL={prefs!.serverURL!}
        token={prefs!.token!}
        sessionId={activeSession.id}
        session={activeSession}
        onBack={() => {
          setActiveSession(null)
          setState('home')
        }}
      />
    )
  }

  // Home shell — content fills the screen above the bottom tab bar.
  // Each tab manages its own header + scrollable content; we just
  // render the right component based on the active tab.
  const serverURL = prefs!.serverURL!
  const token = prefs!.token!
  const username = prefs!.username ?? 'admin'

  return (
    <div className="flex flex-col min-h-screen">
      <div className="flex-1 flex flex-col min-h-0">
        {tab === 'sessions' && (
          <SessionsScreen
            serverURL={serverURL}
            token={token}
            username={username}
            onLogout={onClearAuthAndReturnToLogin}
            onAuthExpired={onClearAuthAndReturnToLogin}
            onOpenSession={(s) => {
              setActiveSession(s)
              setState('session')
            }}
          />
        )}
        {tab === 'memory' && (
          <MemoryScreen
            serverURL={serverURL}
            token={token}
            onAuthExpired={onClearAuthAndReturnToLogin}
          />
        )}
        {tab === 'notes' && (
          <NotesScreen
            serverURL={serverURL}
            token={token}
            onAuthExpired={onClearAuthAndReturnToLogin}
          />
        )}
        {tab === 'activity' && (
          <ActivityScreen
            serverURL={serverURL}
            token={token}
            onAuthExpired={onClearAuthAndReturnToLogin}
          />
        )}
      </div>
      <BottomTabBar active={tab} onChange={setTab} />
    </div>
  )
}
