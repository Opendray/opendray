import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Loader2,
  Pencil,
  Trash2,
  RotateCw,
  Plug,
  ExternalLink,
  Lock,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDistanceToNow } from 'date-fns'
import { Trans, useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'
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
import { EditIntegrationDialog } from '@/components/integrations/EditIntegrationDialog'
import { ProxyConsole } from '@/components/integrations/ProxyConsole'
import { ScopePicker } from '@/components/integrations/ScopePicker'
import {
  listIntegrations,
  registerIntegration,
  rotateIntegrationKey,
  updateIntegration,
  deleteIntegration,
} from '@/lib/integrations'
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
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [editing, setEditing] = useState<Integration | null>(null)
  const [revealKey, setRevealKey] = useState<string | null>(null)
  const [revealTitle, setRevealTitle] = useState<string>(
    t('web.integrations.reveal.titleIssued'),
  )

  const { data: integrations, isLoading } = useQuery({
    queryKey: ['integrations'],
    queryFn: listIntegrations,
    refetchInterval: 8_000,
  })

  const onRegistered = (key: string) => {
    setRevealTitle(t('web.integrations.reveal.titleIssued'))
    setRevealKey(key)
  }

  const onRotated = (key: string) => {
    setRevealTitle(t('web.integrations.reveal.titleRotated'))
    setRevealKey(key)
  }

  return (
    <div className="h-full flex flex-col bg-background">
      <header className="border-b border-border px-6 py-4 flex items-center gap-3">
        <div className="flex-1">
          <h1 className="text-[16px] font-semibold tracking-tight">
            {t('web.integrations.title')}
          </h1>
          <p className="text-[12px] text-muted-foreground">
            <Trans
              i18nKey="web.integrations.subtitle"
              components={{ 1: <code className="mx-1" /> }}
            />
          </p>
        </div>
        <Button
          variant="accent"
          size="sm"
          onClick={() => setCreateOpen(true)}
        >
          <Plus className="size-3.5" /> {t('web.integrations.register')}
        </Button>
      </header>

      <Tabs defaultValue="registered" className="flex-1 flex flex-col min-h-0">
        <div className="border-b border-border px-6 py-2">
          <TabsList>
            <TabsTrigger value="registered">
              {t('web.integrations.tabs.registered')}
            </TabsTrigger>
            <TabsTrigger value="console">
              {t('web.integrations.tabs.console')}
            </TabsTrigger>
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
                  {t('web.integrations.loading')}
                </div>
              )}
          {!isLoading && (integrations?.length ?? 0) === 0 && (
            <div className="flex flex-col items-center justify-center py-16 gap-3 text-center">
              <Plug className="size-10 text-muted-foreground/40" strokeWidth={1.5} />
              <h2 className="text-[14px] font-semibold">
                {t('web.integrations.empty.title')}
              </h2>
              <p className="text-[12px] text-muted-foreground max-w-[360px]">
                {t('web.integrations.empty.description')}
              </p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setCreateOpen(true)}
              >
                <Plus className="size-3.5" /> {t('web.integrations.empty.register')}
              </Button>
            </div>
          )}
              {(() => {
                const all = integrations ?? []
                const system = all.filter((i) => i.is_system)
                const operator = all.filter((i) => !i.is_system)
                const renderCard = (i: Integration) => (
                  <IntegrationCard
                    key={i.id}
                    integration={i}
                    onEdit={() => setEditing(i)}
                    onRotate={() => {
                      if (
                        !window.confirm(
                          t('web.integrations.card.rotateConfirm', { name: i.name }),
                        )
                      )
                        return
                      rotateIntegrationKey(i.id)
                        .then((res) => {
                          qc.invalidateQueries({ queryKey: ['integrations'] })
                          onRotated(res.api_key)
                        })
                        .catch((err: Error) => toast.error(err.message))
                    }}
                    onToggle={(enabled) =>
                      updateIntegration(i.id, { enabled })
                        .then(() =>
                          qc.invalidateQueries({ queryKey: ['integrations'] }),
                        )
                        .catch((err: Error) => toast.error(err.message))
                    }
                    onDelete={() => {
                      if (
                        !confirm(
                          t('web.integrations.card.deleteConfirm', { name: i.name }),
                        )
                      )
                        return
                      deleteIntegration(i.id)
                        .then(() => {
                          qc.invalidateQueries({ queryKey: ['integrations'] })
                          toast.success(t('web.integrations.card.removedToast'))
                        })
                        .catch((err: Error) => toast.error(err.message))
                    }}
                  />
                )
                return (
                  <>
                    {system.length > 0 && (
                      <>
                        <div className="text-[10px] uppercase tracking-wider text-muted-foreground/60 font-medium pt-1">
                          {t('web.integrations.groupSystem')}
                        </div>
                        {system.map(renderCard)}
                      </>
                    )}
                    {system.length > 0 && operator.length > 0 && (
                      <div className="text-[10px] uppercase tracking-wider text-muted-foreground/60 font-medium pt-3">
                        {t('web.integrations.groupOperator')}
                      </div>
                    )}
                    {operator.map(renderCard)}
                  </>
                )
              })()}
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

      <EditIntegrationDialog
        open={editing !== null}
        integration={editing}
        onOpenChange={(v) => {
          if (!v) setEditing(null)
        }}
      />
    </div>
  )
}

function IntegrationCard({
  integration: i,
  onEdit,
  onRotate,
  onToggle,
  onDelete,
}: {
  integration: Integration
  onEdit: () => void
  onRotate: () => void
  onToggle: (enabled: boolean) => void
  onDelete: () => void
}) {
  const { t } = useTranslation()
  const managed = i.is_system
  return (
    <div
      className={cn(
        'border rounded-md p-4 bg-card/30 flex flex-col gap-3',
        managed
          ? 'border-accent/30 bg-accent/[0.04]'
          : 'border-border',
      )}
    >
      <div className="flex items-start gap-3">
        {managed ? (
          <Lock className="size-4 text-accent mt-0.5" />
        ) : (
          <Plug className="size-4 text-accent mt-0.5" />
        )}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-[13px] font-semibold">{i.name}</span>
            <Badge variant="outline" className="font-mono normal-case">
              {i.id}
            </Badge>
            {managed && (
              <Badge
                variant="outline"
                className="border-accent/40 text-accent normal-case"
                title={t('web.integrations.card.managedTooltip')}
              >
                <Lock className="size-2.5" />
                {t('web.integrations.card.managedBadge')}
              </Badge>
            )}
            {i.base_url ? (
              <Badge variant={HEALTH_VARIANT[i.health_status]}>
                {i.health_status}
              </Badge>
            ) : (
              <Badge
                variant="muted"
                title={t('web.integrations.card.consumerTooltip')}
              >
                {t('web.integrations.card.consumerBadge')}
              </Badge>
            )}
            {!i.enabled && (
              <Badge variant="muted">
                {t('web.integrations.card.disabledBadge')}
              </Badge>
            )}
          </div>
          {i.base_url ? (
            <div className="text-[11px] text-muted-foreground/80 font-mono mt-1 flex items-center gap-1.5 flex-wrap">
              <ExternalLink className="size-3 shrink-0" />
              <span className="truncate">{i.base_url}</span>
              <span>·</span>
              <span>/proxy/{i.route_prefix}/*</span>
            </div>
          ) : (
            <div className="text-[11px] text-muted-foreground/60 mt-1">
              {t('web.integrations.card.consumerOnlyHint')}
            </div>
          )}
          <div className="text-[11px] text-muted-foreground/70 mt-1.5 flex flex-wrap gap-1">
            {i.scopes.map((s) => (
              <Badge key={s} variant="outline" className="normal-case font-mono">
                {s}
              </Badge>
            ))}
          </div>
          {(i.health_last_seen || i.rotated_at) && (
            <div className="text-[10px] text-muted-foreground/60 font-mono mt-1.5">
              {i.health_last_seen && i.base_url &&
                t('web.integrations.card.lastProbed', {
                  relative: formatDistanceToNow(new Date(i.health_last_seen), {
                    addSuffix: true,
                  }),
                })}
              {i.health_last_seen && i.base_url && i.rotated_at && ' · '}
              {i.rotated_at &&
                t('web.integrations.card.rotated', {
                  relative: formatDistanceToNow(new Date(i.rotated_at), {
                    addSuffix: true,
                  }),
                })}
            </div>
          )}
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {managed ? (
            <span
              className="text-[10px] text-muted-foreground/70 italic max-w-[180px] text-right leading-tight"
              title={t('web.integrations.card.managedReadOnlyTooltip')}
            >
              {t('web.integrations.card.managedReadOnly')}
            </span>
          ) : (
            <>
              <Switch checked={i.enabled} onCheckedChange={onToggle} />
              <Button
                variant="ghost"
                size="icon"
                onClick={onEdit}
                aria-label={t('web.integrations.card.editAria')}
                title={t('web.integrations.card.editTooltip')}
                className="text-muted-foreground hover:text-foreground"
              >
                <Pencil className="size-3.5" />
              </Button>
              <Button variant="outline" size="sm" onClick={onRotate}>
                <RotateCw className="size-3.5" />
                {t('web.integrations.card.rotateKey')}
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={onDelete}
                aria-label={t('web.integrations.card.deleteAria')}
                className="text-muted-foreground hover:text-destructive"
              >
                <Trash2 className="size-3.5" />
              </Button>
            </>
          )}
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
  const { t } = useTranslation()
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
    if (!name.trim()) {
      setError(t('web.integrations.register_dialog.errorNameRequired'))
      return
    }
    const url = baseURL.trim()
    const prefix = routePrefix.trim()
    if ((url && !prefix) || (!url && prefix)) {
      setError(t('web.integrations.register_dialog.errorBothOrNeither'))
      return
    }
    register.mutate({
      name: name.trim(),
      base_url: url,
      route_prefix: prefix,
      scopes,
      version: version.trim() || undefined,
    })
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
          <DialogTitle>{t('web.integrations.register_dialog.title')}</DialogTitle>
          <DialogDescription>
            {t('web.integrations.register_dialog.description')}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="flex flex-col gap-4 mt-2">
          <div className="space-y-1.5">
            <Label htmlFor="name">
              {t('web.integrations.register_dialog.nameLabel')}
            </Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t('web.integrations.register_dialog.namePlaceholder')}
              required
              autoFocus
            />
          </div>
          <div className="rounded-md border border-border/60 bg-muted/10 p-3 text-[11px] text-muted-foreground leading-snug">
            <Trans
              i18nKey="web.integrations.register_dialog.modeHint"
              components={{ 1: <strong />, 3: <strong /> }}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="base_url">
              {t('web.integrations.register_dialog.baseUrlLabel')}{' '}
              <span className="text-muted-foreground/60">
                {t('web.integrations.register_dialog.optionalSuffix')}
              </span>
            </Label>
            <Input
              id="base_url"
              value={baseURL}
              onChange={(e) => setBaseURL(e.target.value)}
              placeholder={t('web.integrations.register_dialog.baseUrlPlaceholder')}
              className="font-mono"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="route_prefix">
              {t('web.integrations.register_dialog.routePrefixLabel')}{' '}
              <span className="text-muted-foreground/60">
                {t('web.integrations.register_dialog.optionalSuffix')}
              </span>
            </Label>
            <Input
              id="route_prefix"
              value={routePrefix}
              onChange={(e) => setRoutePrefix(e.target.value)}
              placeholder={t('web.integrations.register_dialog.routePrefixPlaceholder')}
              className="font-mono"
            />
            <p className="text-[11px] text-muted-foreground/80">
              <Trans
                i18nKey="web.integrations.register_dialog.routePrefixHint"
                values={{
                  prefix:
                    routePrefix ||
                    t('web.integrations.register_dialog.routePrefixPlaceholderToken'),
                }}
                components={{ 1: <code /> }}
              />
            </p>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="version">
              {t('web.integrations.register_dialog.versionLabel')}
            </Label>
            <Input
              id="version"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder={t('web.integrations.register_dialog.versionPlaceholder')}
              className="font-mono"
            />
          </div>
          <div className="space-y-1.5">
            <Label>{t('web.integrations.register_dialog.scopesLabel')}</Label>
            <ScopePicker
              selected={scopes}
              onChange={setScopes}
              intro={t('web.integrations.register_dialog.scopesIntro')}
            />
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
              {t('web.integrations.register_dialog.cancel')}
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
              {register.isPending
                ? t('web.integrations.register_dialog.submitting')
                : t('web.integrations.register_dialog.submit')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
