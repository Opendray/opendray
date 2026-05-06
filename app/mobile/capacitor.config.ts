import type { CapacitorConfig } from '@capacitor/cli'

// Capacitor configuration for the opendray-v2 mobile app.
//
// `webDir` is the directory Capacitor copies into the native shells
// during `cap sync`. Vite writes there via the `build.outDir` setting
// in this package's vite.config.ts. Keep them in lockstep.
//
// Native projects (`ios/`, `android/`) are NOT committed to the
// repository at this stage. Run the following on first checkout to
// generate them locally:
//
//   pnpm --filter mobile exec cap add ios
//   pnpm --filter mobile exec cap add android
//
// See app/mobile/README.md for the full setup walkthrough.
const config: CapacitorConfig = {
  appId: 'online.linivek.opendray',
  appName: 'OpenDray',
  webDir: 'dist',
  // No `server.url` — production points at the bundled webDir. For
  // device dev with live reload, override per-developer via
  // `pnpm --filter mobile exec cap run ios -l --external` which
  // injects a transient server.url at runtime.
}

export default config
