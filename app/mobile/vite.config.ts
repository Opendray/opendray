import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'node:path'

// Mobile build feeds Capacitor's `webDir` (defaults to `dist` per
// capacitor.config.ts). Unlike web, there's no `/admin/` base prefix —
// the WebView serves from the bundle root.
export default defineConfig(() => ({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: [
      // Mirror the alias chain used by web/vite.config.ts so
      // shared-ui primitives' `@/lib/utils` and `@/stores/theme`
      // resolve identically across both entries. Specific patterns
      // win over generic — order matters.
      {
        find: /^@\/lib\//,
        replacement: path.resolve(__dirname, '../shared/src/lib') + '/',
      },
      {
        find: /^@\/stores\//,
        replacement: path.resolve(__dirname, '../shared/src/stores') + '/',
      },
      {
        find: /^@\/components\/ui\//,
        replacement: path.resolve(__dirname, '../shared-ui/src/primitives') + '/',
      },
      { find: '@', replacement: path.resolve(__dirname, './src') },
    ],
  },
  server: {
    port: 5174, // distinct from web's 5173 so both can run in parallel
    host: true,
  },
  build: {
    outDir: path.resolve(__dirname, 'dist'),
    emptyOutDir: true,
    chunkSizeWarningLimit: 1000,
  },
}))
