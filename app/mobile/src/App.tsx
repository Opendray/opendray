import { useEffect, useState } from 'react'

import { OnboardingScreen } from './screens/OnboardingScreen'
import { LoginScreen } from './screens/LoginScreen'
import { SessionsScreen } from './screens/SessionsScreen'
import { SessionDetailScreen } from './screens/SessionDetailScreen'
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
//   serverURL set, valid token    → home
//
// B5 will replace HomeScreen with the real Sessions list. B4 will
// add a biometric gate between launch and home (auto-unlock the
// stored token via Face ID / Touch ID).
export function App() {
  const [state, setState] = useState<AppState>('loading')
  const [prefs, setPrefs] = useState<StoredPrefs | null>(null)
  const [activeSession, setActiveSession] = useState<SessionSummary | null>(null)

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
          // setAuth has already written; re-read so HomeScreen has the
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

  return (
    <SessionsScreen
      serverURL={prefs!.serverURL!}
      token={prefs!.token!}
      username={prefs!.username ?? 'admin'}
      onLogout={onClearAuthAndReturnToLogin}
      onAuthExpired={onClearAuthAndReturnToLogin}
      onOpenSession={(s) => {
        setActiveSession(s)
        setState('session')
      }}
    />
  )
}
