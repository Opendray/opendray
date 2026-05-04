// Persistent state for the demo client. A real third-party app
// would store its api_key in a secret manager (Vaultwarden, AWS
// Secrets Manager, GCP Secret Manager, OS keychain) — we use a
// flat JSON file because the demo is a single-machine learning
// example, not a production deployment.
//
// File layout:
//
//   examples/integrations/demo-client/.demo-state.json
//   {
//     "integration_id":   "int_xxx",
//     "integration_name": "demo-client",
//     "api_key":          "odk_live_...",
//     "registered_at":    "2026-05-04T05:55:25.571Z"
//   }
//
// Permissions are forced to 0600 (owner-only read/write). The file
// is .gitignored so the secret never lands in version control.

import {
  chmodSync,
  existsSync,
  readFileSync,
  unlinkSync,
  writeFileSync,
} from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'

const moduleDir = dirname(fileURLToPath(import.meta.url))
// state.ts lives in src/ — store the file one level up next to
// package.json so it's easy to spot and easy to delete.
export const STATE_PATH = resolve(moduleDir, '..', '.demo-state.json')

export interface DemoState {
  integration_id: string
  integration_name: string
  api_key: string
  registered_at: string
}

/** Read state. Returns null when the file is missing or unreadable. */
export function loadState(): DemoState | null {
  if (!existsSync(STATE_PATH)) return null
  try {
    const body = readFileSync(STATE_PATH, 'utf8')
    const parsed = JSON.parse(body) as DemoState
    if (
      !parsed ||
      typeof parsed.integration_id !== 'string' ||
      typeof parsed.api_key !== 'string'
    ) {
      return null
    }
    return parsed
  } catch {
    return null
  }
}

/** Write state. Force mode 0600 even if the file already existed. */
export function saveState(s: DemoState): void {
  writeFileSync(STATE_PATH, JSON.stringify(s, null, 2) + '\n', { mode: 0o600 })
  chmodSync(STATE_PATH, 0o600)
}

/** Remove the state file. No-op when missing. */
export function clearState(): void {
  if (existsSync(STATE_PATH)) unlinkSync(STATE_PATH)
}
