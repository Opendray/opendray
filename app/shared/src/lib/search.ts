import { api } from './api'

export type SearchCase = 'smart' | 'sensitive' | 'insensitive'

export interface SearchSubmatch {
  start: number
  end: number
}

export interface SearchMatch {
  path: string
  line: number
  text: string
  submatches?: SearchSubmatch[]
}

export interface SearchResponse {
  matches: SearchMatch[]
  truncated?: boolean
  elapsed?: string
}

export interface SearchOptions {
  path: string
  q: string
  case?: SearchCase
  include?: string
  max?: number
}

export async function searchCwd(opts: SearchOptions): Promise<SearchResponse> {
  const params = new URLSearchParams({ path: opts.path, q: opts.q })
  if (opts.case) params.set('case', opts.case)
  if (opts.include) params.set('include', opts.include)
  if (opts.max) params.set('max', String(opts.max))
  return api<SearchResponse>(`/api/v1/search?${params.toString()}`)
}
