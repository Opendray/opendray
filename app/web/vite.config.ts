import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'node:path'

// dev mode is served at the root by Vite (proxying /api to the Go
// gateway). Production builds are embedded into the Go binary and
// mounted at /admin/, so asset URLs in dist/index.html must resolve
// under that prefix.
export default defineConfig(({ command }) => ({
  base: command === 'build' ? '/admin/' : '/',
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: [
      // Most specific first — anything that resolves into a sibling
      // workspace package (shared, shared-ui) wins over the generic
      // `@` → web/src fallback. Order matters: Vite's alias resolver
      // does first-match-wins on the `find` field.
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
    port: 5173,
    // Bind on all interfaces so the dev server is reachable from
    // phones / tablets on the LAN during testing. Vite's default
    // is 'localhost' only.
    host: true,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8770',
        // ws:true forwards WebSocket upgrades to the Go gateway —
        // required for the terminal stream and the events viewer.
        // Without it WS handshakes 502 silently in dev mode.
        ws: true,
        changeOrigin: true,
      },
    },
  },
  build: {
    // Production build feeds the Go binary's embed.FS in
    // internal/web/dist; the Go side serves it under /admin/*.
    outDir:
      command === 'build'
        ? path.resolve(__dirname, '../../internal/web/dist')
        : path.resolve(__dirname, 'dist'),
    emptyOutDir: true,
    // Bumped from the 500 kB default. Heavy node_modules deps
    // (xterm, hljs, react-markdown, tanstack) are split out by
    // manualChunks below; the remaining chunk is React + scheduler
    // (~180 kB minified) plus the application's own code, totalling
    // ~1.17 MB minified / ~349 kB gzipped. React deliberately stays
    // in the main chunk — see the manualChunks comment for the
    // dispatcher-null bug that splitting caused. Getting this back
    // under 500 kB needs route-level React.lazy() splits that
    // haven't been wired yet.
    chunkSizeWarningLimit: 1300,
    rolldownOptions: {
      output: {
        // Pull big runtimes out of the entry chunk so the login route +
        // small admin pages paint fast. SessionsPage's React.lazy()
        // further splits xterm.js into its own branch.
        //
        // React + scheduler are deliberately NOT split into a separate
        // chunk. Earlier versions did, which surfaced as a black-screen
        // boot with `Cannot read properties of null (reading
        // 'useCallback')` in production: the manualChunks split moved
        // React's `__SECRET_INTERNALS_DO_NOT_USE` into one chunk while
        // a downstream subgraph (one of markdown / tanstack /
        // react-markdown) imported a React internal that was loaded
        // before the React chunk's top-level had run, so useCallback's
        // dispatcher was still null. Bundling React into the main
        // chunk avoids the chunk-evaluation-order race entirely; the
        // main chunk grows ~180 kB minified / ~57 kB gzipped, well
        // under the 1100 kB limit.
        manualChunks(id: string) {
          if (id.includes('node_modules/@xterm/')) return 'xterm'
          if (id.includes('node_modules/highlight.js/')) return 'hljs'
          if (
            id.includes('node_modules/react-markdown/') ||
            id.includes('node_modules/remark-') ||
            id.includes('node_modules/rehype-') ||
            id.includes('node_modules/micromark') ||
            id.includes('node_modules/mdast-util') ||
            id.includes('node_modules/unist-util')
          ) {
            return 'markdown'
          }
          if (id.includes('node_modules/@tanstack/')) return 'tanstack'
          return undefined
        },
      },
    },
  },
}))
