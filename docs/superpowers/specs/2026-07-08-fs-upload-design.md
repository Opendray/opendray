# Files Sidebar: Upload & Folder Creation

**Date:** 2026-07-08
**Status:** Approved design — ready for implementation plan

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
| Write scope | **Confined to the session cwd subtree** | Matches "available to the model in this session"; reuses the existing `resolveWithinRoot()` sandbox; smallest blast radius. |
| Drop types | **Files + folders, recursive** | The stated requirement; subtree preserved under the drop target. |
| Conflict policy | **Auto-rename** (`name-1.ext`, `name-2.ext`, …) | Never loses data, never blocks. |
| Type filter | **None** — any file type | Any file could be relevant to the model. |
| Size cap | **~250 MiB per file, streamed** (`http.MaxBytesReader` + `io.Copy`) | Generous for real inputs without buffering whole files in memory. |
| Transport | **Approach B** — single-file endpoint, client orchestrates the walk | Trivial per-call sandbox-checked server; honest per-file progress; small streamed requests; kind to a shared gateway. |

## Architecture

Three units, each independently understandable and testable:

1. **`POST /api/v1/fs/upload`** (server) — writes exactly one file into the
   sandboxed cwd, creating parent dirs and auto-renaming on conflict.
2. **`fs.ts` client helpers** (web, shared) — `walkDropEntries()` (drop →
   flat file list) and `uploadFile()` (one POST, with progress). Stateless.
3. **`FilesPanel` UI** (web) — buttons, drag-and-drop, a concurrency-limited
   upload runner, progress, and tree refresh.

Reused as-is: the `/fs` chi router group, its admin Bearer middleware, the
`resolveWithinRoot()` + `canonicalize()` path sandbox, the existing
`makeDir`/`listDir` client functions, and React Query for tree state.

### Unit 1 — `POST /api/v1/fs/upload`

New handler in `internal/fs/handler.go`, mounted in the existing `/fs` group
(inherits admin auth). One file per request.

**Request** (`multipart/form-data`):

| field | meaning |
|---|---|
| `root` | session cwd — the sandbox root (same value the panel passes to download/zip) |
| `dir` | absolute destination directory the drop targeted (a folder node's path, or `root` for a background drop) |
| `relpath` | file path relative to `dir` (e.g. `src/utils/helper.ts`, or just `data.csv` for a loose file) |
| `file` | the file part (streamed) |

**Logic:**

1. `resolveWithinRoot(dir, root)` — reject if `dir` escapes `root`.
2. Sanitize `relpath`: reject absolute paths, `..` segments, and null bytes;
   `filepath.Clean` each segment.
3. `final := filepath.Join(dirResolved, relpath)`; re-run
   `resolveWithinRoot(final, root)` (defense in depth).
4. `os.MkdirAll(filepath.Dir(final), 0o755)` — materialize the nested subtree.
5. **Auto-rename:** if `final` exists, probe `name-1.ext`, `name-2.ext`, … until
   a free name is found.
6. Wrap the body in `http.MaxBytesReader` (~250 MiB); `io.Copy` to a temp file in
   the *same* destination directory, then `os.Rename` into place — atomic, so the
   model never sees a half-written file.
7. Respond `{ "path": <final>, "size": <n>, "renamed_from": <original>|null }`.

**Errors:** escape attempt → 400; oversize → 413; non-multipart / missing part →
400; write failure → 500 (temp file cleaned up).

**Security posture:** two independent within-root checks; symlink escapes blocked
by the existing `EvalSymlinks` in `resolveWithinRoot`; size-capped; atomic rename;
the client filename is never trusted verbatim.

### Unit 2 — `app/shared/src/lib/fs.ts`

- **`walkDropEntries(dataTransfer): Promise<{ relpath: string; file: File }[]>`** —
  expands a drop via `DataTransferItem.webkitGetAsEntry()`, recursing directories
  through `FileSystemDirectoryEntry.createReader()` and collecting file leaves with
  their folder-relative path. Loose files return `relpath = file.name`. Pure and
  unit-testable against a faked entry tree.
- **`uploadFile({ root, dir, relpath, file, signal, onProgress })`** — one
  `POST /fs/upload` via `XMLHttpRequest` (chosen over `fetch` only for
  `upload.onprogress`), attaching the Bearer token from `useAuth.getState().token`
  exactly as `api()` does. Returns `{ path, size, renamed_from }`.

The lib stays stateless; sequencing lives in the UI.

### Unit 3 — `FilesPanel.tsx`

**Header buttons:**
- **New folder** — prompts for a name, calls the existing `makeDir(parent, name)`,
  invalidates the parent's query.
- **Upload** — hidden `<input type="file" multiple>` plus a `webkitdirectory`
  input for whole-folder picking; same code path as drop.

**Drag & drop:**
- Over a **folder row** → highlight; that folder is the destination `dir`.
- Over the **panel background** → destination is the session cwd root.
- On drop: `walkDropEntries` → run uploads through a **concurrency-3 runner** →
  each upload's `dir` = the drop target, `relpath` from the walk.

**Feedback & refresh:**
- Inline progress ("Uploading 7 of 23…") with a small bar; per-file errors are
  surfaced, not swallowed.
- On completion, invalidate the React Query keys for every affected directory so
  the tree shows new files/folders and any `-1` auto-renames.
- Empty-state hint on the background: "Drop files or folders here."

## Testing

**Go** (`internal/fs/handler_test.go`):
- upload lands within root;
- `../` and absolute `relpath` rejected;
- nested `relpath` creates the subtree;
- auto-rename picks `-1` on conflict;
- size cap enforced (413);
- non-multipart rejected;
- symlink-escape rejected.

**Web:**
- unit-test `walkDropEntries` against a faked entry tree (loose files, nested
  folders, mixed);
- light interaction test of the upload/drop wiring if the existing suite supports
  it.

## Out of Scope (YAGNI)

- Delete/rename/move within the sidebar (separate feature).
- Writing outside the session cwd subtree.
- Resumable/chunked uploads for very large files.
- In-browser zipping (Approach C) and single-request multi-part (Approach A).
