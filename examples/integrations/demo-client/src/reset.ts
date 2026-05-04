// Reset script — undoes everything the demo persisted.
//
// Run via: `pnpm reset`.
//
// Steps:
//   1. Load .demo-state.json. No state? Nothing to do.
//   2. Login as admin (needed to delete the integration row).
//   3. DELETE /integrations/{id} — even if the row's already gone,
//      we soldier on (404 is fine).
//   4. unlink .demo-state.json.
//
// After running this, the next `pnpm dev` runs the fresh-registration
// branch from the start — the same path a brand-new operator would
// see on first install.

import 'dotenv/config'

import { OpendrayClient, type ApiError } from './client.js'
import { clearState, loadState, STATE_PATH } from './state.js'

const cfg = {
  base: process.env.OPENDRAY_BASE ?? 'http://127.0.0.1:8770',
  adminUser: process.env.OPENDRAY_ADMIN_USER ?? 'admin',
  adminPassword: process.env.OPENDRAY_ADMIN_PASSWORD ?? '',
}

async function main() {
  const state = loadState()
  if (!state) {
    console.log(`no state file at ${STATE_PATH} — nothing to reset.`)
    return
  }

  console.log(`found state for integration ${state.integration_id}`)
  const admin = new OpendrayClient({ base: cfg.base })
  try {
    await admin.login(cfg.adminUser, cfg.adminPassword)
  } catch (err) {
    console.warn(
      `⚠ admin login failed (${(err as Error).message}); removing state file anyway.`,
    )
    clearState()
    return
  }

  try {
    await admin.deleteIntegration(state.integration_id)
    console.log(`✓ deleted integration ${state.integration_id}`)
  } catch (err) {
    const status = (err as ApiError).status
    if (status === 404) {
      console.log(`  integration ${state.integration_id} already gone (404)`)
    } else {
      console.warn(
        `⚠ delete failed (${(err as Error).message}); removing state file anyway`,
      )
    }
  }

  clearState()
  console.log(`✓ removed ${STATE_PATH}`)
}

main().catch((err) => {
  console.error('reset failed:', err)
  process.exit(1)
})
