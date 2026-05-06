import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface AuthState {
  token: string | null
  username: string | null
  expiresAt: string | null
  setSession: (token: string, username: string, expiresAt: string) => void
  clear: () => void
  isAuthed: () => boolean
}

export const useAuth = create<AuthState>()(
  persist(
    (set, get) => ({
      token: null,
      username: null,
      expiresAt: null,
      setSession: (token, username, expiresAt) =>
        set({ token, username, expiresAt }),
      clear: () => set({ token: null, username: null, expiresAt: null }),
      isAuthed: () => {
        const { token, expiresAt } = get()
        if (!token || !expiresAt) return false
        return new Date(expiresAt).getTime() > Date.now()
      },
    }),
    { name: 'opendray.auth' },
  ),
)
