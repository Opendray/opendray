import {
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
  Outlet,
} from '@tanstack/react-router'
import { AppShell } from '@/components/AppShell'
import { LoginPage } from '@/pages/Login'
import { SessionsPage } from '@/pages/Sessions'
import { ProvidersPage } from '@/pages/Providers'
import { ChannelsPage } from '@/pages/Channels'
import { IntegrationsPage } from '@/pages/Integrations'
import { ActivityPage } from '@/pages/Activity'
import { SettingsPage } from '@/pages/Settings'
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
  component: ProvidersPage,
})

const channelsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/channels',
  component: ChannelsPage,
})

const integrationsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/integrations',
  component: IntegrationsPage,
})

const activityRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/activity',
  component: ActivityPage,
})

const settingsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/settings',
  component: SettingsPage,
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
