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
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
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
    // (xterm, hljs, react-markdown, tanstack, react) are split out
    // by manualChunks below; the remaining chunk is the application
    // code itself, which currently sits ~990 kB minified / ~290 kB
    // gzipped. Below 500 kB needs route-level React.lazy() splits
    // that haven't been wired yet — tracked in the codemap.
    chunkSizeWarningLimit: 1100,
    rolldownOptions: {
      output: {
        // Pull big runtimes out of the entry chunk so the login route +
        // small admin pages paint fast. SessionsPage's React.lazy()
        // further splits xterm.js into its own branch.
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
          if (
            id.includes('node_modules/react/') ||
            id.includes('node_modules/react-dom/') ||
            id.includes('node_modules/scheduler/')
          ) {
            return 'react'
          }
          return undefined
        },
      },
    },
  },
}))
