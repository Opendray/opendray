# Canvas (beta) — rollback

The Canvas feature is fully self-contained and reversible. It adds one table,
one gateway route group, one MCP tool, and one web Inspector tab — nothing
existing is modified in a breaking way.

## What it touches

**Backend**
- `internal/store/migrations/0082_canvas_artifacts.sql` — new `canvas_artifacts`
  table (idempotent, touches no existing table/enum).
- `internal/store/migrations/0083_canvas_kind.sql` — adds the `kind` column
  (`ui|flow|mindmap|graph|doc`) to that same new table.
- `internal/canvas/` — the whole package (store + service + handler + tests).
- `internal/app/app.go` — three added blocks: the import, the
  construction/wiring block (`canvasStore`/`canvasSvc`/`canvasHandlers` under
  the `CANVAS (beta)` comment), the `canvasHandlers.Mount(r)` line, and the
  `canvasSessionInjector` adapter type.
- `cmd/opendray/mcp_memory.go` — the `canvas_render` + `canvas_context` tools:
  two `toolDefs` entries, two `dispatchTool` cases, one `writeToolNames` entry
  (`canvas_render`), the `callCanvasRender` / `callCanvasContext` methods, and
  the Canvas lines in the instructions blurb.
- `internal/catalog/adapter.go` — the "### The opendray Canvas" section of
  `memoryGuidanceText` (the system-prompt injection).

**Frontend**
- `app/shared/src/lib/canvas.ts` — new API + events client.
- `app/web/src/components/sessions/inspector/CanvasPanel.tsx` — the tab shell,
  and `CanvasStage.tsx` — the canvas list / annotation / feedback surface.
- `app/web/src/pages/CanvasPopout.tsx` + `app/web/src/router.tsx` — the
  shell-less `/canvas` pop-out route.
- `app/web/src/components/sessions/InspectorPanel.tsx` — added the Canvas tab
  (import, `TabsTrigger`, `TabsContent`), made the tabs controlled, and wired
  the Files→Canvas preview handler.
- `app/web/src/components/sessions/inspector/FilesPanel.tsx` — added the
  optional `onPreviewHtml` prop + the per-row `Eye` preview button.
- `app/i18n/{en,zh,es}.json` — `web.sessions.inspector.tabs.canvas` and the
  `web.sessions.inspector.canvas.*` block.

## To remove entirely

1. `DROP TABLE IF EXISTS canvas_artifacts;` (and delete the `schema_migrations`
   rows `0082_canvas_artifacts` / `0083_canvas_kind` if you want the runner to
   forget them).
2. Delete `internal/canvas/` and both migration files.
3. Revert the `internal/app/app.go` blocks, the `cmd/opendray/mcp_memory.go`
   additions, and the Canvas section in `internal/catalog/adapter.go`.
4. Delete the frontend files above and revert `InspectorPanel.tsx` /
   `FilesPanel.tsx` / `router.tsx` / the i18n keys.
5. Delete the `feat/canvas-beta` branch.

The data is disposable — `canvas_artifacts` holds only rendered previews;
dropping it loses nothing an agent can't re-render. The focused-canvas state is
in-memory only (no table), so it disappears with the process.

## Note on scope

An earlier iteration also had a "Live app" mode (embed / screenshot / annotate a
real running app). It was removed: a cross-origin page cannot be read or
captured without a same-origin reverse proxy, and adding a web proxy conflicts
with opendray's role as an AI-CLI gateway. The Canvas is deliberately mock-only.
