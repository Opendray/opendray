// Minimal Vite env typing for the shared package, which doesn't pull in
// vite/client. Only BASE_URL is needed here — Vite replaces
// `import.meta.env.BASE_URL` with the configured base (e.g. "/admin/")
// at build time. (Interface-merges harmlessly with vite/client in the
// web app's own compilation.)
interface ImportMetaEnv {
  readonly BASE_URL: string
}
interface ImportMeta {
  readonly env: ImportMetaEnv
}
