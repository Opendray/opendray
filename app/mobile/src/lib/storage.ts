// Capacitor Preferences wrapper for the mobile app's persistent state.
//
// On iOS this maps to Keychain Services (so the bearer token is
// encrypted at rest behind the device passcode + biometric); on
// Android it maps to EncryptedSharedPreferences. On web (development
// preview without a device) it falls back to localStorage — fine for
// dev iteration, never used in shipped builds.
//
// Keys are namespaced under `opendray.*` so we don't collide with
// any plugins that also use Preferences.

import { Preferences } from '@capacitor/preferences'

const KEY_SERVER_URL = 'opendray.server_url'
const KEY_TOKEN = 'opendray.token'
const KEY_TOKEN_EXPIRES_AT = 'opendray.token_expires_at'
const KEY_USERNAME = 'opendray.username'

export interface StoredPrefs {
  serverURL: string | null
  token: string | null
  expiresAt: string | null // RFC3339 from /auth/mobile-login
  username: string | null
}

export async function getPrefs(): Promise<StoredPrefs> {
  const [su, tk, te, un] = await Promise.all([
    Preferences.get({ key: KEY_SERVER_URL }),
    Preferences.get({ key: KEY_TOKEN }),
    Preferences.get({ key: KEY_TOKEN_EXPIRES_AT }),
    Preferences.get({ key: KEY_USERNAME }),
  ])
  return {
    serverURL: su.value,
    token: tk.value,
    expiresAt: te.value,
    username: un.value,
  }
}

export async function setServerURL(url: string): Promise<void> {
  await Preferences.set({ key: KEY_SERVER_URL, value: url })
}

export async function setAuth(
  token: string,
  expiresAt: string,
  username: string,
): Promise<void> {
  await Promise.all([
    Preferences.set({ key: KEY_TOKEN, value: token }),
    Preferences.set({ key: KEY_TOKEN_EXPIRES_AT, value: expiresAt }),
    Preferences.set({ key: KEY_USERNAME, value: username }),
  ])
}

export async function clearAuth(): Promise<void> {
  await Promise.all([
    Preferences.remove({ key: KEY_TOKEN }),
    Preferences.remove({ key: KEY_TOKEN_EXPIRES_AT }),
    Preferences.remove({ key: KEY_USERNAME }),
  ])
}

export async function clearAll(): Promise<void> {
  await clearAuth()
  await Preferences.remove({ key: KEY_SERVER_URL })
}

// True when expiresAt is in the past (or unset).
export function tokenExpired(expiresAt: string | null): boolean {
  if (!expiresAt) return true
  const t = Date.parse(expiresAt)
  if (Number.isNaN(t)) return true
  return t <= Date.now()
}
