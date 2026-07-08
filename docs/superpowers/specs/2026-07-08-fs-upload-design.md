# Files Sidebar: Upload & Folder Creation

**Date:** 2026-07-08
**Status:** Approved design, architecture-verified — ready for implementation plan
**Module:** `github.com/opendray/opendray-v2` (Go 1.25) · web: pnpm workspace, Vite + React + TanStack Query v5

## Problem

The session inspector's files sidebar (`FilesPanel`) is read-only for content: an
operator can browse, view, download, and zip a session's working directory, but
cannot get *new* files into it from the browser. To make a document, dataset, or
folder available to the AI model in a session today, the operator must place it on
the gateway host out-of-band. We want to add, directly in the sidebar:

- **Create folder** (button)
- **Upload files** (button + drag-and-drop)
- **Upload whole folders**, recursively, preserving their subtree (button +
  drag-and-drop)

…so uploaded content lands in the session's working directory where the model
already reads it, with no per-session hardcoding, following the existing `/fs`
architecture.

## Constraints & Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Write scope | **Confined to the session cwd subtree** | Matches "available to the model in this session"; reuses the existing `resolveWithinRoot()` sandbox (the same one `/fs/download` and `/fs/zip` use); smallest blast radius. |
| Drop types | **Files + folders, recursive** | The stated requirement; subtree preserved under the drop target. |
| Conflict policy | **Auto-rename** (`name-1.ext`, `name-2.ext`, …) | Never loses data, never blocks. |
| Type filter | **None** — any file type | Any file could be relevant to the model. |
| Size cap | **~250 MiB per file, streamed** (`http.MaxBytesReader` + `io.Copy`) | Generous for real inputs without buffering whole files in memory. |
| Transport | **Approach B** — single-file endpoint, client orchestrates the walk | Trivial per-call sandbox-checked server; honest per-file progress; small streamed requests; kind to a shared gateway. |
| Wire format | **Query-param metadata + raw file body** (not multipart) | `ParseMultipartForm` spools a 250 MiB part to a temp file before we copy it (double write). Query params for `root`/`dir`/`relpath` mirror the existing `/fs/download` + `/fs/zip` convention; the body streams once. |

## Architecture

Three units, each independently understandable and testable:

1. **`POST /api/v1/fs/upload`** (server) — writes exactly one file into the
   sandboxed cwd, creating parent dirs and auto-renaming on conflict.
2. **`fs.ts` client helpers** (web, shared) — `walkDropEntries()` (drop →
   flat file list) and `uploadFile()` (one request, with progress). Stateless.
3. **`FilesPanel` UI** (web) — buttons, drag-and-drop, a concurrency-limited
   upload runner, progress, and tree refresh.

Reused as-is: the `/fs` chi router group and its admin Bearer middleware
(`internal/app/app.go:1216`), the `resolveWithinRoot()` + `canonicalize()` path
sandbox (`internal/fs/handler.go:440,480`), the package-local `writeJSON` /
`writeError` helpers (`handler.go:507,513`), the existing `makeDir` / `listDir`
client functions (`app/shared/src/lib/fs.ts`), and TanStack Query for tree state.

### Unit 1 — `POST /api/v1/fs/upload`

New handler in `internal/fs/handler.go`, registered `r.Post("/upload", h.upload)`
inside the existing `r.Route("/fs", …)` block (`handler.go:42`), inheriting admin
auth. One file per request. Carries a package doc-style comment justifying its
confinement model (matching the security-contract comments already in the file).

**Request:**
- Query params (URL-encoded, same style as download/zip): `root` (session cwd —
  the sandbox root), `dir` (absolute destination directory the drop targeted; a
  folder node's path, or `root` for a background drop), `relpath` (file path
  relative to `dir`, e.g. `src/utils/helper.ts`, or just `data.csv` for a loose
  file).
- Body: the raw file bytes (`Content-Type: application/octet-stream`).

**Logic:**

1. `resolveWithinRoot(dir, root)` — reject with **403** if `dir` escapes `root`
   (same status download/zip return).
2. Sanitize `relpath`: reject absolute paths, `..` segments, and null bytes;
   `filepath.Clean` each segment → **400** on violation.
3. `final := filepath.Join(dirResolved, relpath)`; re-run
   `resolveWithinRoot(final, root)` (defense in depth — validates even a
   not-yet-existing file, since `resolveWithinRoot` falls back to the cleaned
   absolute path when `EvalSymlinks` can't resolve the leaf, `handler.go:463`).
4. `os.MkdirAll(filepath.Dir(final), 0o755)` — materialize the nested subtree.
   (This is the *confined* directory-creation path used for folder uploads.)
5. **Auto-rename:** if `final` exists, probe `name-1.ext`, `name-2.ext`, … until
   a free name is found.
6. `r.Body = http.MaxBytesReader(w, r.Body, fsUploadMaxBytes)` (named const,
   `fsUploadMaxBytes = 250 * 1024 * 1024`, mirroring `uploadMaxBytes` in
   `internal/session/handler.go:500`). Stream via `io.Copy` to a temp file in the
   *same* destination directory, then `os.Rename` into place — atomic, so the
   model never sees a half-written file. On any copy error, remove the temp file
   (cleanup-on-error, as `session/handler.go:555` does).
   - *Note:* temp-file-then-rename is a **net-new** write pattern (the existing
     session upload uses `O_EXCL` create + cleanup because it writes random,
     guaranteed-unique names). We choose rename here specifically to get atomic
     visibility alongside auto-rename.
7. Respond via `writeJSON(w, http.StatusCreated, …)`:
   `{ "path": <final>, "size": <n>, "renamed_from": <original>|null }`.

**Errors** (via `writeError`, which emits `{"error": …}`): escape → 403; bad
`relpath` / missing params → 400; oversize → 413; write failure → 500.

**Security posture:** writes confined to `root` (two independent
`resolveWithinRoot` checks); symlink escapes blocked by the `EvalSymlinks` inside
`resolveWithinRoot`; size-capped; atomic rename; the client `relpath` is never
trusted verbatim.

**Documented asymmetry — `mkdir` is not root-confined.** The existing
`POST /fs/mkdir` (reused by the New-folder button) only rejects bad `name` chars
and `canonicalize`s `parent`; it does **not** call `resolveWithinRoot`
(`handler.go:392`). So `/fs/upload` is deliberately *stricter* than `/fs/mkdir`.
In practice the New-folder button only ever passes a `parent` inside the session
cwd, so the UI stays confined; the endpoint's broader reach is admin-only and
pre-existing. Tightening `mkdir` with an optional `root` check is noted as a
possible follow-up but is **out of scope** here (don't change existing endpoint
behavior for this feature).

### Unit 2 — `app/shared/src/lib/fs.ts` (imported by web as `@/lib/fs`)

Named exports matching the file's existing style (`export async function`, no
default export; JSON calls go through `api<T>()` from `./api`):

- **`walkDropEntries(dataTransfer): Promise<{ relpath: string; file: File }[]>`** —
  expands a drop via `DataTransferItem.webkitGetAsEntry()`, recursing directories
  through `FileSystemDirectoryEntry.createReader()` (draining `readEntries` in a
  loop, since it returns at most 100 entries per call) and collecting file leaves
  with their folder-relative path. Loose files return `relpath = file.name`.
  **Pure** — no network, no store — so it stays unit-testable if a JS test runner
  is added later (see Testing).
- **`uploadFile({ root, dir, relpath, file, signal, onProgress })`** — one
  `POST /fs/upload` via `XMLHttpRequest` (chosen over `fetch` solely for
  `upload.onprogress`). Builds the URL with `root`/`dir`/`relpath` as encoded
  query params; sends `file` as the raw body. Attaches the Bearer token exactly as
  `api()` does but from a non-hook context:
  `xhr.setRequestHeader('Authorization', \`Bearer ${useAuth.getState().token}\`)`
  (synchronous store read, `useAuth` from `@/stores/auth`). Supports `AbortSignal`
  via `xhr.abort()`. Resolves `{ path, size, renamed_from }`.

The lib stays stateless; sequencing/concurrency lives in the UI.

### Unit 3 — `FilesPanel.tsx` (and small colocated components)

`FilesPanel` already receives the session root as the **`cwd`** prop (from
`session.cwd`, prop-drilled via `InspectorPanel.tsx:169`); the destination `dir`
for a background drop is this `cwd`, and for a folder-targeted drop it's that
node's path. No new session lookup needed.

**Header buttons** (`@/components/ui/button`, `lucide-react` icons):
- **New folder** (`FolderPlus`) — prompts for a name, calls the existing
  `makeDir(parent, name)`, then invalidates `['fs', parent]`.
- **Upload** (`Upload`) — hidden `<input type="file" multiple>` plus a
  `webkitdirectory` input for whole-folder picking; same code path as drop.

**Drag & drop** — mirror the existing overlay pattern in `Plugins.tsx:1088`
(`ring-2 ring-accent/70` highlight, `dataTransfer.types.includes('Files')` guard,
`dropEffect='copy'`):
- Over a **folder row** → highlight; that folder is the destination `dir`.
- Over the **panel background** → destination is `cwd`.
- On drop: `walkDropEntries` → run uploads through a **concurrency-3 runner** →
  each upload's `dir` = the drop target, `relpath` from the walk.

**Feedback & refresh:**
- Inline progress: an "Uploading 7 of 23…" line with a lightweight bar (a plain
  `div` width-percentage from `onProgress`; no progress primitive exists and none
  is added). A `Loader2` spinner while active.
- Success/failure via `sonner` `toast.success` / `toast.error(msg, { description })`.
  Per-file errors surfaced, not swallowed; auto-renames reported
  ("uploaded as data-1.csv").
- On completion, `queryClient.invalidateQueries({ queryKey: ['fs', <dir>] })` for
  every affected directory so the tree shows new files/folders and `-1` renames.
- Empty-state hint on the background: "Drop files or folders here."

**TypeScript/lint conventions** (`app/web` tsconfig + flat ESLint): `strict` is
not enabled, but `noUnusedLocals`/`noUnusedParameters` and `verbatimModuleSyntax`
are — so new code uses `import type` for type-only imports, leaves no unused
symbols, and obeys `react-hooks` rules for any new hooks.

## Testing

**Go** (`internal/fs/handler_test.go` — `go test` is the standard, already-present
harness): upload lands within root; `../` and absolute `relpath` rejected; nested
`relpath` creates the subtree; auto-rename picks `-1` on conflict; size cap
enforced (413); symlink-escape rejected (403); missing params → 400.

**Web:** the repo has **no JS test runner** (no vitest/jest, no `*.test.ts`, no
`test` script). `walkDropEntries` is written pure so it *can* be unit-tested, but
adding a test runner is **out of scope** for this feature; the drop/upload path is
validated manually in the browser plus by the Go endpoint tests. (If we later
adopt vitest, a `walkDropEntries` unit test is the first candidate.)

## Out of Scope (YAGNI)

- Delete/rename/move within the sidebar (separate feature).
- Writing outside the session cwd subtree.
- Root-confining the existing `/fs/mkdir` endpoint (possible follow-up).
- Resumable/chunked uploads for very large files.
- Adding a JS test runner / web unit tests.
- In-browser zipping (Approach C) and single-request multipart (Approach A).
