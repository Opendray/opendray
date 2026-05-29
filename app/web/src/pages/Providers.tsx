import { useState, useEffect } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2, Save, RotateCcw } from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { ConfigForm } from '@/components/providers/ConfigForm'
import { ProviderModelsSection } from '@/components/providers/ProviderModelsSection'
import { BrandAvatar } from '@/components/BrandAvatar'
import { providerIconKey } from '@/lib/providerIcons'
import { ClaudeAccountsPanel } from '@/components/providers/ClaudeAccountsPanel'
import { useConfirmDialog } from '@/components/ConfirmDialog'
import {
  listProviders,
  toggleProvider,
  updateProviderConfig,
  checkProviderUpdate,
  updateProvider,
} from '@/lib/catalog'
import type { Provider } from '@/lib/types'
import { cn } from '@/lib/utils'

export function ProvidersPage() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: providers, isLoading } = useQuery({
    queryKey: ['providers'],
    queryFn: listProviders,
  })

  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [draft, setDraft] = useState<Record<string, unknown>>({})

  // Default to first provider.
  useEffect(() => {
    if (!selectedId && providers && providers.length > 0) {
      setSelectedId(providers[0].manifest.id)
    }
  }, [providers, selectedId])

  const selected = providers?.find((p) => p.manifest.id === selectedId)

  // Reset draft when selection changes.
  useEffect(() => {
    setDraft(selected?.config ?? {})
  }, [selected?.manifest.id, selected?.manifest_hash])

  const saveConfig = useMutation({
    mutationFn: ({ id, cfg }: { id: string; cfg: Record<string, unknown> }) =>
      updateProviderConfig(id, cfg),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['providers'] })
      toast.success(t('web.providers.detail.savedToast'))
    },
    onError: (e: Error) =>
      toast.error(t('web.providers.detail.saveFailedToast'), {
        description: e.message,
      }),
  })

  const toggle = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      toggleProvider(id, enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['providers'] }),
    onError: (e: Error) =>
      toast.error(t('web.providers.detail.toggleFailedToast'), {
        description: e.message,
      }),
  })

  return (
    <div className="h-full flex">
      {/* Provider list */}
      <aside className="w-64 shrink-0 border-r border-border flex flex-col bg-background">
        <div className="h-9 px-3 flex items-center border-b border-border">
          <span className="text-[11px] font-semibold uppercase tracking-tight text-muted-foreground">
            {t('web.providers.list.title')}
          </span>
          <span className="ml-2 text-[10px] text-muted-foreground/60 font-mono">
            {providers?.length ?? 0}
          </span>
        </div>
        <ScrollArea className="flex-1">
          <div className="p-1.5 flex flex-col gap-0.5">
            {isLoading && (
              <div className="flex items-center gap-2 px-2 py-3 text-[12px] text-muted-foreground">
                <Loader2 className="size-3.5 animate-spin" />
                {t('web.providers.list.loading')}
              </div>
            )}
            {(providers ?? []).map((p) => (
              <button
                key={p.manifest.id}
                type="button"
                onClick={() => setSelectedId(p.manifest.id)}
                className={cn(
                  'w-full flex items-center gap-2.5 px-2.5 py-2 rounded-md text-left transition-colors border border-transparent',
                  selectedId === p.manifest.id
                    ? 'bg-card border-border'
                    : 'hover:bg-card/60',
                )}
              >
                <BrandAvatar
                  iconKey={providerIconKey(p.manifest.id)}
                  fallbackLetter={p.manifest.displayName?.charAt(0) ?? '?'}
                  size={20}
                  title={p.manifest.displayName}
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-1.5">
                    <span className="text-[12px] font-medium truncate">
                      {p.manifest.displayName}
                    </span>
                    {!p.enabled && (
                      <Badge variant="muted">
                        {t('web.providers.list.disabledBadge')}
                      </Badge>
                    )}
                  </div>
                  <div className="text-[10px] text-muted-foreground/70 font-mono truncate">
                    {p.manifest.executable}
                  </div>
                </div>
              </button>
            ))}
          </div>
        </ScrollArea>
      </aside>

      {/* Detail */}
      {selected ? (
        <ProviderDetail
          provider={selected}
          draft={draft}
          onChange={setDraft}
          dirty={JSON.stringify(draft) !== JSON.stringify(selected.config)}
          saving={saveConfig.isPending}
          onSave={() =>
            saveConfig.mutate({ id: selected.manifest.id, cfg: draft })
          }
          onReset={() => setDraft(selected.config ?? {})}
          onToggle={(enabled) =>
            toggle.mutate({ id: selected.manifest.id, enabled })
          }
        />
      ) : (
        <div className="flex-1 flex items-center justify-center text-[12px] text-muted-foreground">
          {isLoading ? '' : t('web.providers.list.noneSelected')}
        </div>
      )}
    </div>
  )
}

function ProviderDetail({
  provider,
  draft,
  onChange,
  dirty,
  saving,
  onSave,
  onReset,
  onToggle,
}: {
  provider: Provider
  draft: Record<string, unknown>
  onChange: (v: Record<string, unknown>) => void
  dirty: boolean
  saving: boolean
  onSave: () => void
  onReset: () => void
  onToggle: (enabled: boolean) => void
}) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const m = provider.manifest
  const rt = provider.runtime
  // Latest-version check is a network (npm) call, so it runs on its own
  // endpoint and only for installed CLI providers. staleTime aligns with
  // the server-side npm cache (1h) and refetch-on-focus/mount is off so
  // toggling between tabs doesn't keep round-tripping for data the
  // backend would just return from cache anyway.
  const { data: upd } = useQuery({
    queryKey: ['provider-update', m.id],
    queryFn: () => checkProviderUpdate(m.id),
    enabled: m.kind === 'cli' && !!rt?.installed,
    staleTime: 60 * 60_000,
    refetchOnWindowFocus: false,
    refetchOnMount: false,
  })
  const { confirm, dialog: confirmDialog } = useConfirmDialog()
  const updateMut = useMutation({
    mutationFn: () => updateProvider(m.id),
    onSuccess: (res) => {
      if (!res.available) {
        // Can't update here (e.g. read-only npm prefix) — guidance, not error.
        toast.info(t('web.providers.detail.updateUnavailable'), {
          description: res.reason,
        })
        return
      }
      if (res.changed) {
        toast.success(
          t('web.providers.detail.updatedToast', {
            from: res.beforeVersion,
            to: res.afterVersion,
          }),
        )
      } else {
        toast.info(t('web.providers.detail.alreadyLatestToast'))
      }
      qc.invalidateQueries({ queryKey: ['providers'] })
      qc.invalidateQueries({ queryKey: ['provider-update', m.id] })
    },
    onError: (e: Error) =>
      toast.error(t('web.providers.detail.updateFailedToast'), {
        description: e.message,
      }),
  })
  const caps: { key: keyof typeof m.capabilities; labelKey: string }[] = [
    { key: 'supportsResume', labelKey: 'web.providers.detail.caps.resume' },
    { key: 'supportsStream', labelKey: 'web.providers.detail.caps.stream' },
    { key: 'supportsImages', labelKey: 'web.providers.detail.caps.images' },
    { key: 'supportsMcp', labelKey: 'web.providers.detail.caps.mcp' },
  ]
  return (
    <main className="flex-1 flex flex-col min-w-0 bg-background">
      <div className="border-b border-border px-6 py-4 flex items-start gap-4">
        <BrandAvatar
          iconKey={providerIconKey(m.id)}
          fallbackLetter={m.displayName?.charAt(0) ?? '?'}
          size={36}
          title={m.displayName}
          className="mt-1"
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <h1 className="text-[16px] font-semibold tracking-tight">
              {m.displayName}
            </h1>
            <Badge variant="outline" className="font-mono normal-case">
              {m.kind}
            </Badge>
            {rt && !rt.installed ? (
              <Badge
                variant="outline"
                className="font-mono normal-case text-muted-foreground"
              >
                {t('web.providers.detail.notInstalled')}
              </Badge>
            ) : (
              <Badge variant="outline" className="font-mono normal-case">
                {/* Real installed CLI version when known; falls back to
                    the manifest's schema version only if unprobed. */}
                {rt?.installedVersion ?? `v${m.version}`}
              </Badge>
            )}
            {upd?.updateAvailable && upd.latestVersion ? (
              <>
                <Badge variant="accent" className="font-mono normal-case">
                  {t('web.providers.detail.updateAvailable', {
                    version: upd.latestVersion,
                  })}
                </Badge>
                <Button
                  size="sm"
                  variant="outline"
                  className="h-6 px-2 text-[11px]"
                  disabled={updateMut.isPending}
                  onClick={async () => {
                    const n = upd?.activeSessions ?? 0
                    if (n > 0) {
                      const label = m.displayName ?? m.id
                      const ok = await confirm({
                        title: `Upgrade ${label} CLI?`,
                        description:
                          `${n} session${n === 1 ? '' : 's'} currently running on ${m.id}. ` +
                          `${n === 1 ? "It'll" : "They'll"} keep using the old version in memory — new sessions (and any on-demand code loads) pick up the new one. Continue?`,
                        confirmLabel: 'Upgrade',
                        cancelLabel: 'Cancel',
                      })
                      if (!ok) return
                    }
                    updateMut.mutate()
                  }}
                >
                  {updateMut.isPending ? (
                    <Loader2 className="h-3 w-3 animate-spin" />
                  ) : null}
                  {updateMut.isPending
                    ? t('web.providers.detail.updating')
                    : t('web.providers.detail.update', {
                        version: upd.latestVersion,
                      })}
                </Button>
              </>
            ) : null}
          </div>
          <p className="text-[12px] text-muted-foreground mt-1 max-w-[60ch]">
            {m.description}
          </p>
          <div className="flex items-center gap-1.5 mt-2 flex-wrap">
            {caps.map(({ key, labelKey }) =>
              m.capabilities[key] ? (
                <Badge key={key} variant="accent">
                  {t(labelKey)}
                </Badge>
              ) : null,
            )}
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <span className="text-[11px] text-muted-foreground">
            {provider.enabled
              ? t('web.providers.detail.enabled')
              : t('web.providers.detail.disabled')}
          </span>
          <Switch
            checked={provider.enabled}
            onCheckedChange={onToggle}
            aria-label={t('web.providers.detail.toggleAria', {
              name: m.displayName,
            })}
          />
        </div>
      </div>

      <ScrollArea className="flex-1">
        <div className="px-6 py-6 max-w-[640px]">
          <h2 className="text-[12px] font-semibold uppercase tracking-wider text-muted-foreground/80 mb-4">
            {t('web.providers.detail.configuration')}
          </h2>
          {m.configSchema && m.configSchema.length > 0 ? (
            <ConfigForm
              schema={m.configSchema}
              initial={provider.config}
              onChange={onChange}
            />
          ) : (
            <p className="text-[12px] text-muted-foreground italic">
              {t('web.providers.detail.noConfig')}
            </p>
          )}
          {m.modelFlag ? (
            <>
              <Separator className="my-6" />
              <ProviderModelsSection
                manifest={m}
                value={draft}
                onChange={onChange}
              />
            </>
          ) : null}
          {m.id === 'claude' && (
            <>
              <Separator className="my-6" />
              <ClaudeAccountsPanel />
            </>
          )}
          <Separator className="my-6" />
          <div className="text-[10px] text-muted-foreground/70 font-mono">
            {t('web.providers.detail.executable')} {m.executable}
            <br />
            {t('web.providers.detail.manifestHash')} {provider.manifest_hash}
          </div>
        </div>
      </ScrollArea>

      <div className="border-t border-border px-6 py-3 flex items-center justify-end gap-2 bg-background">
        <Button
          variant="ghost"
          size="sm"
          onClick={onReset}
          disabled={!dirty || saving}
        >
          <RotateCcw className="size-3.5" />
          {t('web.providers.detail.reset')}
        </Button>
        <Button
          variant="accent"
          size="sm"
          onClick={onSave}
          disabled={!dirty || saving}
        >
          {saving ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <Save className="size-3.5" />
          )}
          {saving
            ? t('web.providers.detail.saving')
            : t('web.providers.detail.save')}
        </Button>
      </div>
      {confirmDialog}
    </main>
  )
}
