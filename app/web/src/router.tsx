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
import { DatabasePage } from '@/pages/Database'
import { ActivityPage } from '@/pages/Activity'
import { MemoryPage } from '@/pages/Memory'
import { MemoryWorkersPage } from '@/pages/MemoryWorkers'
import { NotesPage } from '@/pages/Project'
import { CortexPage } from '@/pages/Cortex'
import { QuarantinePage } from '@/pages/Quarantine'
import { ArchivedPage } from '@/pages/Archived'
import { BackupsPage } from '@/pages/Backups'
import { ExportPage } from '@/pages/Export'
import { VaultPage } from '@/pages/Notes'
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
  // `open` deep-links a session to auto-select on arrival — used when
  // escalating a Cortex discussion into a session. Without this the
  // param is stripped and the new session never gets focused.
  validateSearch: (search): { open?: string } =>
    typeof search.open === 'string' ? { open: search.open } : {},
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

const databaseRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/database',
  component: DatabasePage,
  validateSearch: (search) => ({
    cwd: typeof search.cwd === 'string' ? search.cwd : '',
  }),
})

const activityRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/activity',
  component: ActivityPage,
})

// ── Cortex — the unified Memory → Notes → Knowledge module ──────
// One nav entry, layered inside: home (the flywheel), project
// workspace (Notes), knowledge (global), memory (raw + hygiene +
// quarantine). The old /notes /memory /knowledge routes redirect so
// bookmarks survive.

const cortexRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/cortex',
  component: CortexPage,
})

const cortexProjectRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/cortex/project',
  component: NotesPage,
  validateSearch: (search) => ({
    cwd: typeof search.cwd === 'string' ? search.cwd : '',
  }),
})

const cortexKnowledgeRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/cortex/knowledge',
  component: KnowledgePage,
})

const cortexMemoryRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/cortex/memory',
  component: MemoryPage,
})

const cortexMemoryQuarantineRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/cortex/memory/quarantine',
  component: QuarantinePage,
})

const cortexMemoryArchivedRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/cortex/memory/archived',
  component: ArchivedPage,
})

// The Cortex settings page — every AI-drive knob in one place
// (spawn injection mode, per-task workers + models, capture rules,
// ambient profiles, cost audit). The old "memory workers" page grew
// into this; its route redirects.
const cortexSettingsRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/cortex/settings',
  component: MemoryWorkersPage,
})

const cortexMemoryWorkersRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/cortex/memory/workers',
  beforeLoad: () => {
    throw redirect({ to: '/cortex/settings' })
  },
})

// Legacy redirects — the silo tabs are gone, the data is not.
const notesRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/notes',
  validateSearch: (search) => ({
    cwd: typeof search.cwd === 'string' ? search.cwd : '',
  }),
  beforeLoad: ({ search }) => {
    throw redirect({ to: '/cortex/project', search })
  },
})

const knowledgeRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/knowledge',
  beforeLoad: () => {
    throw redirect({ to: '/cortex/knowledge' })
  },
})

const memoryRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/memory',
  beforeLoad: () => {
    throw redirect({ to: '/cortex/memory' })
  },
})

const memoryProjectRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/memory/project',
  validateSearch: (search) => ({
    cwd: typeof search.cwd === 'string' ? search.cwd : '',
  }),
  beforeLoad: ({ search }) => {
    throw redirect({ to: '/cortex/project', search })
  },
})

const memoryArchivedRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/memory/archived',
  beforeLoad: () => {
    throw redirect({ to: '/cortex/memory/archived' })
  },
})

const memoryWorkersRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/memory/workers',
  beforeLoad: () => {
    throw redirect({ to: '/cortex/settings' })
  },
})

// Vault — the markdown/Obsidian-sync utility, demoted out of the core
// Memory/Notes/Knowledge triad (Experience Flywheel §7).
const vaultRoute = createRoute({
  getParentRoute: () => protectedRoute,
  path: '/vault',
  component: VaultPage,
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
    databaseRoute,
    activityRoute,
    cortexRoute,
    cortexProjectRoute,
    cortexKnowledgeRoute,
    cortexSettingsRoute,
    cortexMemoryRoute,
    cortexMemoryQuarantineRoute,
    cortexMemoryArchivedRoute,
    cortexMemoryWorkersRoute,
    notesRoute,
    vaultRoute,
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
