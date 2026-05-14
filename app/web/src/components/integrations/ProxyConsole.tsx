import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Send,
  Loader2,
  History,
  ChevronRight,
  Terminal as TerminalIcon,
  Plug,
} from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Code } from '@/components/ui/code'
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from '@/components/ui/select'
import { Trans, useTranslation } from 'react-i18next'
import { listIntegrations } from '@/lib/integrations'
import { useAuth } from '@/stores/auth'
import { cn } from '@/lib/utils'

const METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']

interface HistoryEntry {
  ts: number
  method: string
  path: string
  status: number
  durationMs: number
}

interface ResponseSummary {
  status: number
  statusText: string
  durationMs: number
  headers: Array<[string, string]>
  body: string
  contentType: string | null
}

function historyKey(integrationId: string): string {
  return `opendray.proxy-history.${integrationId}`
}

function loadHistory(id: string): HistoryEntry[] {
  try {
    const raw = localStorage.getItem(historyKey(id))
    return raw ? (JSON.parse(raw) as HistoryEntry[]) : []
  } catch {
    return []
  }
}

function saveHistory(id: string, list: HistoryEntry[]) {
  try {
    localStorage.setItem(historyKey(id), JSON.stringify(list.slice(0, 30)))
  } catch {
    /* localStorage full / disabled — ignore */
  }
}

export function ProxyConsole() {
  const { t } = useTranslation()
  const token = useAuth((s) => s.token)
  const { data: integrations } = useQuery({
    queryKey: ['integrations'],
    queryFn: listIntegrations,
  })

  const [selectedId, setSelectedId] = useState<string | null>(null)
  const selected = useMemo(
    () => integrations?.find((i) => i.id === selectedId) ?? null,
    [integrations, selectedId],
  )

  // Default to first enabled integration.
  useEffect(() => {
    if (!selectedId && integrations && integrations.length > 0) {
      setSelectedId(integrations[0].id)
    }
  }, [integrations, selectedId])

  const [method, setMethod] = useState('GET')
  const [path, setPath] = useState('/health')
  const [headersText, setHeadersText] = useState('')
  const [body, setBody] = useState('')
  const [response, setResponse] = useState<ResponseSummary | null>(null)
  const [sending, setSending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [history, setHistory] = useState<HistoryEntry[]>([])

  // Reload history when integration changes.
  useEffect(() => {
    if (selectedId) {
      setHistory(loadHistory(selectedId))
    } else {
      setHistory([])
    }
    setResponse(null)
    setError(null)
  }, [selectedId])

  const send = async () => {
    if (!selected || !token) return
    setSending(true)
    setError(null)
    setResponse(null)
    try {
      const url = `/api/v1/proxy/${selected.route_prefix}${
        path.startsWith('/') ? path : `/${path}`
      }`
      const headers = new Headers()
      headers.set('Authorization', `Bearer ${token}`)
      for (const line of headersText.split('\n')) {
        const idx = line.indexOf(':')
        if (idx <= 0) continue
        const name = line.slice(0, idx).trim()
        const value = line.slice(idx + 1).trim()
        if (name && value) headers.set(name, value)
      }
      const init: RequestInit = { method, headers }
      if (method !== 'GET' && method !== 'DELETE' && body.trim().length > 0) {
        if (!headers.has('Content-Type')) {
          headers.set('Content-Type', 'application/json')
        }
        init.body = body
      }

      const t0 = performance.now()
      const res = await fetch(url, init)
      const t1 = performance.now()
      const durationMs = Math.round(t1 - t0)

      const headerList: Array<[string, string]> = []
      res.headers.forEach((v, k) => headerList.push([k, v]))
      const ct = res.headers.get('Content-Type')
      let bodyText = await res.text()
      if (ct && ct.includes('application/json')) {
        try {
          bodyText = JSON.stringify(JSON.parse(bodyText), null, 2)
        } catch {
          /* not JSON despite header */
        }
      }

      setResponse({
        status: res.status,
        statusText: res.statusText,
        durationMs,
        headers: headerList,
        body: bodyText,
        contentType: ct,
      })

      const entry: HistoryEntry = {
        ts: Date.now(),
        method,
        path,
        status: res.status,
        durationMs,
      }
      const next = [entry, ...history.filter((h) => !(h.method === method && h.path === path))]
      setHistory(next)
      if (selectedId) saveHistory(selectedId, next)
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : t('web.integrations.proxy.requestFailed'),
      )
    } finally {
      setSending(false)
    }
  }

  if (integrations && integrations.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 text-center py-16 px-6">
        <Plug className="size-10 text-muted-foreground/40" strokeWidth={1.5} />
        <h2 className="text-[14px] font-semibold">
          {t('web.integrations.proxy.emptyTitle')}
        </h2>
        <p className="text-[12px] text-muted-foreground max-w-[360px]">
          {t('web.integrations.proxy.emptyDescription')}
        </p>
      </div>
    )
  }

  return (
    <div className="grid grid-cols-[280px_1fr] gap-4 h-full min-h-[540px]">
      {/* Left: integration list + history */}
      <aside className="flex flex-col gap-3 min-h-0">
        <div>
          <Label className="!text-[10px] !uppercase !tracking-wider">
            {t('web.integrations.proxy.targetLabel')}
          </Label>
          <Select
            value={selectedId ?? ''}
            onValueChange={(v) => setSelectedId(v)}
          >
            <SelectTrigger className="mt-1.5">
              <SelectValue placeholder={t('web.integrations.proxy.selectPlaceholder')} />
            </SelectTrigger>
            <SelectContent>
              {(integrations ?? []).map((i) => (
                <SelectItem key={i.id} value={i.id}>
                  {i.name} · /{i.route_prefix}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {selected && (
            <div className="mt-2 text-[11px] text-muted-foreground/80 font-mono">
              {t('web.integrations.proxy.baseLabel')} {selected.base_url}
            </div>
          )}
        </div>

        <div className="flex flex-col min-h-0 gap-1.5">
          <div className="flex items-center gap-1.5 text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
            <History className="size-3" />
            {t('web.integrations.proxy.history')}
          </div>
          <ScrollArea className="flex-1 min-h-0">
            <div className="flex flex-col gap-0.5 pr-1">
              {history.length === 0 && (
                <span className="text-[11px] text-muted-foreground/60 italic">
                  {t('web.integrations.proxy.historyEmpty')}
                </span>
              )}
              {history.map((h) => (
                <button
                  key={h.ts + h.method + h.path}
                  type="button"
                  onClick={() => {
                    setMethod(h.method)
                    setPath(h.path)
                  }}
                  className="flex items-center gap-2 px-2 py-1 rounded-md text-left text-[11px] hover:bg-card text-muted-foreground hover:text-foreground"
                >
                  <span
                    className={cn(
                      'font-mono w-[42px] shrink-0',
                      h.status >= 500
                        ? 'text-state-failed'
                        : h.status >= 400
                          ? 'text-state-idle'
                          : 'text-state-running',
                    )}
                  >
                    {h.method}
                  </span>
                  <span className="font-mono truncate flex-1">{h.path}</span>
                  <span className="font-mono text-[10px] tabular-nums text-muted-foreground/60 shrink-0">
                    {h.status}
                  </span>
                </button>
              ))}
            </div>
          </ScrollArea>
        </div>
      </aside>

      {/* Right: request + response */}
      <div className="flex flex-col gap-3 min-w-0 min-h-0">
        <section className="flex flex-col gap-2 border border-border rounded-md p-3 bg-card/30">
          <div className="flex items-center gap-2">
            <Select value={method} onValueChange={setMethod}>
              <SelectTrigger className="w-[100px] font-mono">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {METHODS.map((m) => (
                  <SelectItem key={m} value={m}>
                    {m}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <div className="flex-1 flex items-center border border-border rounded-md bg-input/40 h-9 overflow-hidden font-mono text-[12px]">
              <span className="px-2 text-muted-foreground border-r border-border">
                /api/v1/proxy/
                {selected?.route_prefix ??
                  t('web.integrations.proxy.prefixPlaceholder')}
              </span>
              <input
                type="text"
                value={path}
                onChange={(e) => setPath(e.target.value)}
                placeholder="/health"
                className="flex-1 bg-transparent px-2 outline-none text-foreground"
              />
            </div>
            <Button
              onClick={send}
              variant="accent"
              size="sm"
              disabled={!selected || sending}
            >
              {sending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Send className="size-3.5" />
              )}
              {sending
                ? t('web.integrations.proxy.sending')
                : t('web.integrations.proxy.send')}
            </Button>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div className="space-y-1">
              <Label className="!text-[10px] !uppercase !tracking-wider">
                {t('web.integrations.proxy.extraHeadersLabel')}
              </Label>
              <Textarea
                rows={3}
                value={headersText}
                onChange={(e) => setHeadersText(e.target.value)}
                placeholder={`X-Foo: bar`}
                className="font-mono"
              />
            </div>
            <div className="space-y-1">
              <Label className="!text-[10px] !uppercase !tracking-wider">
                {t('web.integrations.proxy.bodyLabel')}
              </Label>
              <Textarea
                rows={3}
                value={body}
                onChange={(e) => setBody(e.target.value)}
                placeholder={`{"hello":"world"}`}
                className="font-mono"
                disabled={method === 'GET' || method === 'DELETE'}
              />
            </div>
          </div>
        </section>

        <section className="flex-1 flex flex-col gap-2 min-h-0">
          {error && (
            <div className="text-[12px] text-destructive bg-destructive/10 border border-destructive/30 rounded-md px-3 py-2">
              {error}
            </div>
          )}

          {response && (
            <div className="border border-border rounded-md p-3 bg-card/30 flex flex-col gap-2 min-h-0 flex-1">
              <header className="flex items-center gap-2">
                <Badge
                  variant={
                    response.status >= 500
                      ? 'danger'
                      : response.status >= 400
                        ? 'warning'
                        : 'success'
                  }
                >
                  {response.status} {response.statusText}
                </Badge>
                <span className="text-[11px] text-muted-foreground/70 font-mono">
                  {response.durationMs} ms
                </span>
                {response.contentType && (
                  <span className="text-[11px] text-muted-foreground/70 font-mono truncate">
                    {response.contentType}
                  </span>
                )}
              </header>

              <ScrollArea className="flex-1 min-h-0">
                <div className="flex flex-col gap-3 pr-1">
                  <div>
                    <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium mb-1">
                      {t('web.integrations.proxy.headers')}
                    </div>
                    <Code className="text-[10px]">
                      {response.headers
                        .map(([k, v]) => `${k}: ${v}`)
                        .join('\n')}
                    </Code>
                  </div>
                  <div>
                    <div className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium mb-1">
                      {t('web.integrations.proxy.body')}
                    </div>
                    <Code>
                      {response.body || t('web.integrations.proxy.emptyBody')}
                    </Code>
                  </div>
                </div>
              </ScrollArea>
            </div>
          )}

          {!response && !error && (
            <div className="border border-dashed border-border rounded-md p-6 text-center text-[12px] text-muted-foreground flex-1 flex flex-col items-center justify-center gap-2">
              <TerminalIcon
                className="size-8 text-muted-foreground/40"
                strokeWidth={1.5}
              />
              <span>
                {t('web.integrations.proxy.stubText')}
                <br />
                <Trans
                  i18nKey="web.integrations.proxy.stubInjects"
                  components={{
                    1: <code className="font-mono text-foreground" />,
                    3: <code className="font-mono" />,
                  }}
                />
              </span>
              <ChevronRight className="size-4 opacity-0" />
            </div>
          )}
        </section>
      </div>
    </div>
  )
}
