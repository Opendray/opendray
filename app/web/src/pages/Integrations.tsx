import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Loader2,
  Trash2,
  RotateCw,
  Plug,
  ExternalLink,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDistanceToNow } from 'date-fns'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { APIKeyRevealDialog } from '@/components/integrations/APIKeyRevealDialog'
import { ProxyConsole } from '@/components/integrations/ProxyConsole'
import {
  listIntegrations,
  registerIntegration,
  rotateIntegrationKey,
  updateIntegration,
  deleteIntegration,
} from '@/lib/integrations'
import { ALL_SCOPES } from '@/lib/types'
import type { Integration, IntegrationHealth } from '@/lib/types'

const HEALTH_VARIANT: Record<
  IntegrationHealth,
  'success' | 'warning' | 'danger' | 'muted'
> = {
  healthy: 'success',
  degraded: 'warning',
  unhealthy: 'danger',
  unknown: 'muted',
}

export function IntegrationsPage() {
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [revealKey, setRevealKey] = useState<string | null>(null)
  const [revealTitle, setRevealTitle] = useState<string>('API key issued')

  const { data: integrations, isLoading } = useQuery({
    queryKey: ['integrations'],
    queryFn: listIntegrations,
    refetchInterval: 8_000,
  })

  const onRegistered = (key: string) => {
    setRevealTitle('API key issued')
    setRevealKey(key)
  }

  const onRotated = (key: string) => {
    setRevealTitle('API key rotated')
    setRevealKey(key)
  }

  return (
    <div className="h-full flex flex-col bg-background">
      <header className="border-b border-border px-6 py-4 flex items-center gap-3">
        <div className="flex-1">
          <h1 className="text-[16px] font-semibold tracking-tight">
            Integrations
          </h1>
          <p className="text-[12px] text-muted-foreground">
            External apps that consume opendray. Reverse-proxy through
            <code className="mx-1">/api/v1/proxy/&lt;prefix&gt;/…</code>
            and subscribe to events via the WS endpoint.
          </p>
        </div>
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreateOpen(true)}
        >
          <Plus className="size-3.5" /> Register
        </Button>
      </header>

      <Tabs defaultValue="registered" className="flex-1 flex flex-col min-h-0">
        <div className="border-b border-border px-6 py-2">
          <TabsList>
            <TabsTrigger value="registered">Registered</TabsTrigger>
            <TabsTrigger value="console">Reverse proxy</TabsTrigger>
          </TabsList>
        </div>

        <TabsContent
          value="registered"
          className="flex-1 min-h-0 mt-0 data-[state=inactive]:hidden"
        >
          <ScrollArea className="h-full">
            <div className="p-6 max-w-[960px] flex flex-col gap-3">
              {isLoading && (
                <div className="flex items-center gap-2 text-[12px] text-muted-foreground">
                  <Loader2 className="size-3.5 animate-spin" />
                  Loading…
                </div>
              )}
          {!isLoading && (integrations?.length ?? 0) === 0 && (
            <div className="flex flex-col items-center justify-center py-16 gap-3 text-center">
              <Plug className="size-10 text-muted-foreground/40" strokeWidth={1.5} />
              <h2 className="text-[14px] font-semibold">No integrations yet</h2>
              <p className="text-[12px] text-muted-foreground max-w-[360px]">
                Register an external app to give it a scoped API key. Its
                code stays out of this repo.
              </p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCreateOpen(true)}
              >
                <Plus className="size-3.5" /> Register integration
              </Button>
            </div>
          )}
              {(integrations ?? []).map((i) => (
                <IntegrationCard
                  key={i.id}
                  integration={i}
                  onRotate={() =>
                    rotateIntegrationKey(i.id)
                      .then((res) => {
                        qc.invalidateQueries({ queryKey: ['integrations'] })
                        onRotated(res.api_key)
                      })
                      .catch((err: Error) => toast.error(err.message))
                  }
                  onToggle={(enabled) =>
                    updateIntegration(i.id, { enabled })
                      .then(() =>
                        qc.invalidateQueries({ queryKey: ['integrations'] }),
                      )
                      .catch((err: Error) => toast.error(err.message))
                  }
                  onDelete={() => {
                    if (!confirm(`Delete integration ${i.name}?`)) return
                    deleteIntegration(i.id)
                      .then(() => {
                        qc.invalidateQueries({ queryKey: ['integrations'] })
                        toast.success('Integration removed')
                      })
                      .catch((err: Error) => toast.error(err.message))
                  }}
                />
              ))}
            </div>
          </ScrollArea>
        </TabsContent>

        <TabsContent
          value="console"
          className="flex-1 min-h-0 mt-0 data-[state=inactive]:hidden"
        >
          <div className="px-6 py-4 h-full">
            <ProxyConsole />
          </div>
        </TabsContent>
      </Tabs>

      <RegisterDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onRegistered={(apiKey) => {
          onRegistered(apiKey)
        }}
      />

      <APIKeyRevealDialog
        open={revealKey !== null}
        apiKey={revealKey ?? ''}
        title={revealTitle}
        onClose={() => setRevealKey(null)}
      />
    </div>
  )
}

function IntegrationCard({
  integration: i,
  onRotate,
  onToggle,
  onDelete,
}: {
  integration: Integration
  onRotate: () => void
  onToggle: (enabled: boolean) => void
  onDelete: () => void
}) {
  return (
    <div className="border border-border rounded-md p-4 bg-card/30 flex flex-col gap-3">
      <div className="flex items-start gap-3">
        <Plug className="size-4 text-accent mt-0.5" />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-[13px] font-semibold">{i.name}</span>
            <Badge variant="outline" className="font-mono normal-case">
              {i.id}
            </Badge>
            <Badge variant={HEALTH_VARIANT[i.health_status]}>
              {i.health_status}
            </Badge>
            {!i.enabled && <Badge variant="muted">disabled</Badge>}
          </div>
          <div className="text-[11px] text-muted-foreground/80 font-mono mt-1 flex items-center gap-1.5 flex-wrap">
            <ExternalLink className="size-3 shrink-0" />
            <span className="truncate">{i.base_url}</span>
            <span>·</span>
            <span>/proxy/{i.route_prefix}/*</span>
          </div>
          <div className="text-[11px] text-muted-foreground/70 mt-1.5 flex flex-wrap gap-1">
            {i.scopes.map((s) => (
              <Badge key={s} variant="outline" className="normal-case font-mono">
                {s}
              </Badge>
            ))}
          </div>
          {i.health_last_seen && (
            <div className="text-[10px] text-muted-foreground/60 font-mono mt-1.5">
              last probed {formatDistanceToNow(new Date(i.health_last_seen), { addSuffix: true })}
              {i.rotated_at && (
                <>
                  {' '}· rotated {formatDistanceToNow(new Date(i.rotated_at), { addSuffix: true })}
                </>
              )}
            </div>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Switch checked={i.enabled} onCheckedChange={onToggle} />
          <Button variant="outline" size="sm" onClick={onRotate}>
            <RotateCw className="size-3.5" />
            Rotate key
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={onDelete}
            aria-label="Delete integration"
            className="text-muted-foreground hover:text-destructive"
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </div>
    </div>
  )
}

function RegisterDialog({
  open,
  onOpenChange,
  onRegistered,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
  onRegistered: (apiKey: string) => void
}) {
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [baseURL, setBaseURL] = useState('')
  const [routePrefix, setRoutePrefix] = useState('')
  const [version, setVersion] = useState('')
  const [scopes, setScopes] = useState<string[]>([
    'session:read',
    'event:subscribe:session.*',
  ])
  const [error, setError] = useState<string | null>(null)

  const reset = () => {
    setName('')
    setBaseURL('')
    setRoutePrefix('')
    setVersion('')
    setScopes(['session:read', 'event:subscribe:session.*'])
    setError(null)
  }

  const register = useMutation({
    mutationFn: registerIntegration,
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['integrations'] })
      reset()
      onOpenChange(false)
      onRegistered(res.api_key)
    },
    onError: (e: Error) => setError(e.message),
  })

  const submit = (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    if (!name.trim() || !baseURL.trim() || !routePrefix.trim()) {
      setError('name, base_url, and route_prefix are required.')
      return
    }
    register.mutate({
      name: name.trim(),
      base_url: baseURL.trim(),
      route_prefix: routePrefix.trim(),
      scopes,
      version: version.trim() || undefined,
    })
  }

  const toggleScope = (s: string) => {
    setScopes((prev) =>
      prev.includes(s) ? prev.filter((x) => x !== s) : [...prev, s],
    )
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (!v) reset()
        onOpenChange(v)
      }}
    >
      <DialogContent className="max-w-[520px]">
        <DialogHeader>
          <DialogTitle>Register integration</DialogTitle>
          <DialogDescription>
            Issues a one-time API key. Copy it before closing — opendray
            never displays the plaintext again.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-4 mt-2">
          <div className="space-y-1.5">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="PetTracker"
              required
              autoFocus
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="base_url">Base URL</Label>
            <Input
              id="base_url"
              value={baseURL}
              onChange={(e) => setBaseURL(e.target.value)}
              placeholder="http://192.168.3.42:8080"
              required
              className="font-mono"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="route_prefix">Route prefix</Label>
            <Input
              id="route_prefix"
              value={routePrefix}
              onChange={(e) => setRoutePrefix(e.target.value)}
              placeholder="pet-tracker"
              required
              className="font-mono"
            />
            <p className="text-[11px] text-muted-foreground/80">
              Reachable at <code>/api/v1/proxy/{routePrefix || '<prefix>'}/*</code>.
            </p>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="version">Version (optional)</Label>
            <Input
              id="version"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder="0.1.0"
              className="font-mono"
            />
          </div>
          <div className="space-y-1.5">
            <Label>Scopes</Label>
            <div className="flex flex-wrap gap-1.5">
              {ALL_SCOPES.map((s) => (
                <button
                  key={s}
                  type="button"
                  onClick={() => toggleScope(s)}
                  className={`px-2 py-0.5 rounded-md border text-[11px] font-mono transition-colors ${
                    scopes.includes(s)
                      ? 'border-accent bg-accent/15 text-foreground'
                      : 'border-border bg-transparent text-muted-foreground hover:bg-card'
                  }`}
                >
                  {s}
                </button>
              ))}
            </div>
          </div>

          {error && (
            <div className="text-[12px] text-destructive bg-destructive/10 border border-destructive/30 rounded-md px-3 py-2">
              {error}
            </div>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => onOpenChange(false)}
              disabled={register.isPending}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="accent"
              size="sm"
              disabled={register.isPending}
            >
              {register.isPending && (
                <Loader2 className="size-3.5 animate-spin" />
              )}
              {register.isPending ? 'Registering…' : 'Register'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
