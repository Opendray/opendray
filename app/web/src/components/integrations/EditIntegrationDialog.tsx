import { useEffect, useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { updateIntegration } from '@/lib/integrations'
import type { Integration } from '@/lib/types'

import { ScopePicker } from './ScopePicker'

interface EditIntegrationDialogProps {
  open: boolean
  integration: Integration | null
  onOpenChange: (v: boolean) => void
}

// EditIntegrationDialog allows the operator to change the editable
// fields on an existing integration row: scopes, base_url, version,
// enabled. Name and route_prefix are immutable — both columns are
// UNIQUE in the DB and changing them would require coordinated
// updates of token caches and proxy mounts.
export function EditIntegrationDialog({
  open,
  integration,
  onOpenChange,
}: EditIntegrationDialogProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [baseURL, setBaseURL] = useState('')
  const [version, setVersion] = useState('')
  const [scopes, setScopes] = useState<string[]>([])
  const [error, setError] = useState<string | null>(null)

  // Reset form when the integration prop changes (different row
  // selected, or dialog reopened on the same row).
  useEffect(() => {
    if (!integration) return
    setBaseURL(integration.base_url ?? '')
    setVersion(integration.version ?? '')
    setScopes(integration.scopes ?? [])
    setError(null)
  }, [integration, open])

  const update = useMutation({
    mutationFn: (patch: Parameters<typeof updateIntegration>[1]) =>
      updateIntegration(integration!.id, patch),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['integrations'] })
      toast.success(t('web.integrations.edit_dialog.updatedToast'))
      onOpenChange(false)
    },
    onError: (err: Error) => setError(err.message),
  })

  if (!integration) return null

  const isConsumer = !integration.base_url
  const url = baseURL.trim()
  const baseURLChanged = url !== (integration.base_url ?? '')
  // Switching consumer → proxy or proxy → consumer would require
  // a new route_prefix too, which we can't change here. Block it.
  const baseURLLooksLikeModeSwitch =
    isConsumer && url !== ''
      ? true
      : !isConsumer && url === ''
        ? true
        : false

  const submit = (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    if (baseURLLooksLikeModeSwitch) {
      setError(t('web.integrations.edit_dialog.errorModeSwitch'))
      return
    }
    const patch: Parameters<typeof updateIntegration>[1] = {}
    if (baseURLChanged) patch.base_url = url
    if (version.trim() !== (integration.version ?? '')) {
      patch.version = version.trim()
    }
    if (
      JSON.stringify(scopes) !== JSON.stringify(integration.scopes ?? [])
    ) {
      patch.scopes = scopes
    }
    if (Object.keys(patch).length === 0) {
      onOpenChange(false)
      return
    }
    update.mutate(patch)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[560px] max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {t('web.integrations.edit_dialog.title', { name: integration.name })}
          </DialogTitle>
          <DialogDescription>
            {t('web.integrations.edit_dialog.description')}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={submit} className="flex flex-col gap-5 mt-3">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-[11px] text-muted-foreground/70">
                {t('web.integrations.edit_dialog.nameLabel')}
              </Label>
              <Input
                value={integration.name}
                readOnly
                disabled
                className="font-mono opacity-70"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-[11px] text-muted-foreground/70">
                {t('web.integrations.edit_dialog.routePrefixLabel')}
              </Label>
              <Input
                value={
                  integration.route_prefix ||
                  t('web.integrations.edit_dialog.consumerOnlyLabel')
                }
                readOnly
                disabled
                className="font-mono opacity-70"
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="edit_base_url">
              {t('web.integrations.edit_dialog.baseUrlLabel')}{' '}
              <span className="text-muted-foreground/60">
                {isConsumer
                  ? t('web.integrations.edit_dialog.baseUrlConsumerSuffix')
                  : t('web.integrations.edit_dialog.baseUrlProxySuffix')}
              </span>
            </Label>
            <Input
              id="edit_base_url"
              value={baseURL}
              onChange={(e) => setBaseURL(e.target.value)}
              placeholder={
                isConsumer
                  ? t('web.integrations.edit_dialog.baseUrlConsumerPlaceholder')
                  : t('web.integrations.edit_dialog.baseUrlProxyPlaceholder')
              }
              className="font-mono"
              disabled={isConsumer}
            />
            {isConsumer && (
              <p className="text-[10.5px] text-muted-foreground/60">
                {t('web.integrations.edit_dialog.consumerHint')}
              </p>
            )}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="edit_version">
              {t('web.integrations.edit_dialog.versionLabel')}
            </Label>
            <Input
              id="edit_version"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
              placeholder={t('web.integrations.edit_dialog.versionPlaceholder')}
              className="font-mono"
            />
          </div>

          <div className="space-y-1.5">
            <Label>{t('web.integrations.edit_dialog.scopesLabel')}</Label>
            <ScopePicker
              selected={scopes}
              onChange={setScopes}
              intro={t('web.integrations.edit_dialog.scopesIntro')}
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
              disabled={update.isPending}
            >
              {t('web.integrations.edit_dialog.cancel')}
            </Button>
            <Button
              type="submit"
              variant="accent"
              size="sm"
              disabled={update.isPending}
            >
              {update.isPending && <Loader2 className="size-3.5 animate-spin" />}
              {t('web.integrations.edit_dialog.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
