import {
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
  Outlet,
} from '@tanstack/react-router'
import { Cpu, MessageSquare, Plug, Activity, Settings } from 'lucide-react'

import { AppShell } from '@/components/AppShell'
import { LoginPage } from '@/pages/Login'
import { SessionsPage } from '@/pages/Sessions'
import { Placeholder } from '@/pages/Placeholder'
import { useAuth } from '@/stores/auth'

const rootRoute = createRootRoute({ component: () => <Outlet /> })

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginPage,
  validateSearch: (search) => ({ next: (search.next as string) || undefined }),
})

const protectedRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'protected',
  beforeLoad: ({ location }) => {
    if (!useAuth.getState().isAuthed()) {
      throw redirect({ to: '/login', search: { next: location.pathname } })
    }
  },
  component: AppShell,
})

const indexRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/sessions' })
  },
})

const sessionsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/sessions',
  component: SessionsPage,
})

const providersRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/providers',
  component: () => (
    <Placeholder
      icon={Cpu}
      title="Providers"
      body="Catalog list and per-provider config form arrive in W3."
    />
  ),
})

const channelsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/channels',
  component: () => (
    <Placeholder
      icon={MessageSquare}
      title="Channels"
      body="Telegram / Slack channel CRUD and test-send arrive in W3."
    />
  ),
})

const integrationsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/integrations',
  component: () => (
    <Placeholder
      icon={Plug}
      title="Integrations"
      body="Register / rotate-key / reverse-proxy console arrive in W3 + W4."
    />
  ),
})

const activityRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/activity',
  component: () => (
    <Placeholder
      icon={Activity}
      title="Activity"
      body="Audit log + live event stream viewer arrive in W4."
    />
  ),
})

const settingsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/settings',
  component: () => (
    <Placeholder
      icon={Settings}
      title="Settings"
      body="Theme is in the topbar. Account + about arrive in W5."
    />
  ),
})

const routeTree = rootRoute.addChildren([
  loginRoute,
  protectedRoute.addChildren([
    indexRoute,
    sessionsRoute,
    providersRoute,
    channelsRoute,
    integrationsRoute,
    activityRoute,
    settingsRoute,
  ]),
])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
