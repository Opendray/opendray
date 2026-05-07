import { useEffect, useState } from 'react'

import { OnboardingScreen } from './screens/OnboardingScreen'
import { LoginScreen } from './screens/LoginScreen'
import { SessionsScreen } from './screens/SessionsScreen'
import { SessionDetailScreen } from './screens/SessionDetailScreen'
import { MemoryScreen } from './screens/MemoryScreen'
import { NotesScreen } from './screens/NotesScreen'
import { ActivityScreen } from './screens/ActivityScreen'
import { MoreScreen, type SubPage } from './screens/MoreScreen'
import { ChannelsScreen } from './screens/ChannelsScreen'
import { IntegrationsScreen } from './screens/IntegrationsScreen'
import { ProvidersScreen } from './screens/ProvidersScreen'
import { BackupsScreen } from './screens/BackupsScreen'
import { SettingsScreen } from './screens/SettingsScreen'
import { BottomTabBar, type Tab } from './components/BottomTabBar'
import { type SessionSummary } from './lib/api'
import {
  type StoredPrefs,
  clearAll,
  clearAuth,
  getPrefs,
  tokenExpired,
} from './lib/storage'

type AppState =
  | 'loading'
  | 'onboarding'
  | 'login'
  | 'home'
  | 'session'
  | SubPage

// Top-level state machine. Drives which screen renders based on what
// we have persisted in Capacitor Preferences:
//
//   serverURL absent              → onboarding
//   serverURL set, no/expired tok → login
//   serverURL set, valid token    → home (5 bottom tabs)
//   home + tap session card       → session detail (full-screen)
//   "More" tab + tap entry        → channels / integrations / providers
//                                    (full-screen, no tab bar)
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

  const serverURL = prefs!.serverURL!
  const token = prefs!.token!
  const username = prefs!.username ?? 'admin'

  // ── Full-screen sub-pages (no bottom tab bar) ────────────────────

  if (state === 'session' && activeSession) {
    return (
      <SessionDetailScreen
        serverURL={serverURL}
        token={token}
        sessionId={activeSession.id}
        session={activeSession}
        onBack={() => {
          setActiveSession(null)
          setState('home')
        }}
      />
    )
  }

  if (state === 'channels') {
    return (
      <ChannelsScreen
        serverURL={serverURL}
        token={token}
        onBack={() => setState('home')}
        onAuthExpired={onClearAuthAndReturnToLogin}
      />
    )
  }

  if (state === 'integrations') {
    return (
      <IntegrationsScreen
        serverURL={serverURL}
        token={token}
        onBack={() => setState('home')}
        onAuthExpired={onClearAuthAndReturnToLogin}
      />
    )
  }

  if (state === 'providers') {
    return (
      <ProvidersScreen
        serverURL={serverURL}
        token={token}
        onBack={() => setState('home')}
        onAuthExpired={onClearAuthAndReturnToLogin}
      />
    )
  }

  if (state === 'backups') {
    return (
      <BackupsScreen
        serverURL={serverURL}
        token={token}
        onBack={() => setState('home')}
        onAuthExpired={onClearAuthAndReturnToLogin}
      />
    )
  }

  if (state === 'settings') {
    return (
      <SettingsScreen
        username={username}
        serverURL={serverURL}
        expiresAt={prefs!.expiresAt}
        onBack={() => setState('home')}
      />
    )
  }

  // ── Home shell with tab bar ──────────────────────────────────────

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
        {tab === 'more' && (
          <MoreScreen
            username={username}
            serverURL={serverURL}
            expiresAt={prefs!.expiresAt}
            onOpen={(page) => setState(page)}
            onLogout={onClearAuthAndReturnToLogin}
          />
        )}
      </div>
      <BottomTabBar active={tab} onChange={setTab} />
    </div>
  )
}
