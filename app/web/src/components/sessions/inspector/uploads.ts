import { uploadFile, type WalkedFile, type UploadResult } from '@/lib/fs'

export interface UploadItem extends WalkedFile {
  dir: string
}

export interface RunUploadsOpts {
  concurrency?: number
  onSettled?: (index: number) => void
  signal?: AbortSignal
}

export interface RunUploadsResult {
  results: UploadResult[]
  errors: { relpath: string; error: Error }[]
}

/**
 * Upload `items` with bounded concurrency. Each item carries its own
 * destination `dir` (drop target) and `relpath` (from the folder walk).
 * `onSettled` fires once per file, success or failure, for progress.
 * The pool never rejects — failures are collected in `errors`.
 */
export async function runUploads(
  items: UploadItem[],
  root: string,
  opts: RunUploadsOpts = {},
): Promise<RunUploadsResult> {
  const { concurrency = 3, onSettled, signal } = opts
  const results: UploadResult[] = []
  const errors: { relpath: string; error: Error }[] = []
  let next = 0

  async function worker() {
    while (next < items.length) {
      const i = next++
      const it = items[i]
      try {
        results.push(
          await uploadFile({
            root,
            dir: it.dir,
            relpath: it.relpath,
            file: it.file,
            signal,
          }),
        )
      } catch (e) {
        errors.push({ relpath: it.relpath, error: e as Error })
      } finally {
        onSettled?.(i)
      }
    }
  }

  const pool = Array.from(
    { length: Math.min(concurrency, items.length) },
    worker,
  )
  await Promise.all(pool)
  return { results, errors }
}
