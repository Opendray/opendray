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
import { MemoryPage } from '@/pages/Memory'
import { MemoryWorkersPage } from '@/pages/MemoryWorkers'
import { ProjectPage } from '@/pages/Project'
import { ArchivedPage } from '@/pages/Archived'
import { BackupsPage } from '@/pages/Backups'
import { ExportPage } from '@/pages/Export'
import { NotesPage } from '@/pages/Notes'
import { KnowledgePage } from '@/pages/Knowledge'
import { PluginsPage } from '@/pages/Plugins'
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

const notesRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/notes',
  component: NotesPage,
  // Notes hosts two modes: the project's official doc ('project', default)
  // and the freeform markdown vault ('vault'). cwd selects the project.
  validateSearch: (search) => ({
    mode: search.mode === 'vault' ? 'vault' : 'project',
    cwd: typeof search.cwd === 'string' ? search.cwd : '',
  }),
})

const knowledgeRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/knowledge',
  component: KnowledgePage,
})

const memoryRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/memory',
  component: MemoryPage,
})

const memoryProjectRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/memory/project',
  component: ProjectPage,
  validateSearch: (search) => ({
    cwd: typeof search.cwd === 'string' ? search.cwd : '',
  }),
})

const memoryArchivedRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/memory/archived',
  component: ArchivedPage,
})

const memoryWorkersRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/memory/workers',
  component: MemoryWorkersPage,
})

const pluginsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/plugins',
  component: PluginsPage,
})

const settingsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/settings',
  component: SettingsPage,
  validateSearch: (search) => ({
    section: typeof search.section === 'string' ? search.section : undefined,
  }),
})

const backupsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/backups',
  component: BackupsPage,
})

const exportRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/export',
  component: ExportPage,
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
    notesRoute,
    knowledgeRoute,
    memoryRoute,
    memoryProjectRoute,
    memoryArchivedRoute,
    memoryWorkersRoute,
    pluginsRoute,
    settingsRoute,
    backupsRoute,
    exportRoute,
  ]),
])

// Strip trailing slash so '/admin/' → '/admin' (router expects no
// trailing slash). Empty string in dev (BASE_URL='/').
const basepath = import.meta.env.BASE_URL.replace(/\/$/, '')

export const router = createRouter({
  routeTree,
  basepath: basepath || undefined,
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
