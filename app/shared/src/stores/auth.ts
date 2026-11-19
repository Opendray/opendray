import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type Role = 'admin' | 'member'

interface AuthState {
  token: string | null
  username: string | null
  expiresAt: string | null
  // role gates admin-only UI (users, integrations, plugins, backups,
  // server settings). null until the login response or /auth/me says.
  role: Role | null
  // primary is the keyfile/config admin; users-table logins can only
  // change their own password, not their username.
  primary: boolean
  setSession: (token: string, username: string, expiresAt: string, role?: Role) => void
  setIdentity: (role: Role, primary: boolean) => void
  clear: () => void
  isAuthed: () => boolean
  isAdmin: () => boolean
}

export const useAuth = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      username: null,
      expiresAt: null,
      role: null,
      primary: false,
      setSession: (token, username, expiresAt, role) =>
        set({ token, username, expiresAt, role: role ?? get().role }),
      setIdentity: (role, primary) => set({ role, primary }),
      clear: () =>
        set({ token: null, username: null, expiresAt: null, role: null, primary: false }),
      isAuthed: () => {
        const { token, expiresAt } = get()
        if (!token || !expiresAt) return false
        return new Date(expiresAt).getTime() > Date.now()
      },
      // Unknown role (older persisted session, /auth/me not back yet)
      // shows the admin UI; the server still enforces the role.
      isAdmin: () => get().role !== 'member',
    }),
    { name: 'opendray.auth' },
  ),
)
