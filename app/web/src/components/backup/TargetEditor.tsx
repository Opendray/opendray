import { useState } from 'react'
import { toast } from 'sonner'
import { Trans, useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import {
  TARGET_KINDS,
  type TargetKind,
  type TargetSpec,
  createTarget,
} from '@/lib/backup'
import { APIError } from '@/lib/api'

// TargetEditor is the create-form for any backup target kind. It
// owns its own state; the caller passes onCreated to refresh
// upstream lists. Used from both /backups → Targets and
// /settings → Backup.
export function TargetEditor({
  onCreated,
}: {
  onCreated: () => void | Promise<void>
}) {
  const { t } = useTranslation()
  const [kind, setKind] = useState<TargetKind>('local')
  const [id, setId] = useState('')
  const [enabled, setEnabled] = useState(true)
  const [busy, setBusy] = useState(false)
  const [config, setConfig] = useState<Record<string, unknown>>({})

  function patch(p: Record<string, unknown>) {
    setConfig((c) => ({ ...c, ...p }))
  }

  async function submit() {
    setBusy(true)
    try {
      // Strip empty strings so backend uses defaults.
      const cleaned: Record<string, unknown> = {}
      for (const [k, v] of Object.entries(config)) {
        if (v === '' || v === undefined || v === null) continue
        cleaned[k] = v
      }
      await createTarget({
        id: id || undefined,
        kind,
        config: cleaned,
        enabled,
      })
      toast.success(t('web.backups.targetEditor.createdToast'))
      await onCreated()
    } catch (err) {
      const msg =
        err instanceof APIError
          ? msgFromAPIError(err)
          : err instanceof Error
            ? err.message
            : 'Unknown error'
      toast.error(t('web.backups.targetEditor.createFailedToast'), { description: msg })
    } finally {
      setBusy(false)
    }
  }

  function changeKind(k: TargetKind) {
    setKind(k)
    setConfig({}) // reset kind-specific state
  }

  return (
    <DialogContent className="max-w-xl max-h-[85vh] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{t('web.backups.targetEditor.title')}</DialogTitle>
      </DialogHeader>

      <div className="flex flex-col gap-4">
        {/* Kind picker as cards */}
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">{t('web.backups.targetEditor.kindPicker')}</Label>
          <div className="grid grid-cols-2 gap-2">
            {TARGET_KINDS.map((k) => (
              <button
                key={k.kind}
                type="button"
                onClick={() => changeKind(k.kind)}
                className={
                  'text-left p-2.5 rounded-md border transition-colors ' +
                  (kind === k.kind
                    ? 'border-accent bg-accent/5'
                    : 'border-border hover:bg-card/40')
                }
              >
                <div className="text-[12px] font-medium">{k.label}</div>
                <div className="text-[10.5px] text-muted-foreground mt-0.5">
                  {k.description}
                </div>
                <div className="text-[10px] text-muted-foreground/70 mt-1 leading-tight">
                  {k.examples}
                </div>
              </button>
            ))}
          </div>
        </div>

        {/* ID (optional) */}
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">{t('web.backups.targetEditor.idLabel')}</Label>
          <Input
            value={id}
            onChange={(e) => setId(e.target.value)}
            placeholder={t('web.backups.targetEditor.idPlaceholder')}
            className="h-8 font-mono"
          />
        </div>

        {/* Per-kind config */}
        <div className="flex flex-col gap-3 border-t border-border pt-3">
          {kind === 'local' && (
            <Field
              label={t('web.backups.targetEditor.local.rootLabel')}
              hint={t('web.backups.targetEditor.local.rootHint')}
            >
              <Input
                value={(config.root as string) ?? ''}
                onChange={(e) => patch({ root: e.target.value })}
                placeholder={t('web.backups.targetEditor.local.rootPlaceholder')}
                className="h-8 font-mono"
              />
            </Field>
          )}

          {kind === 'smb' && (
            <>
              <FieldRow>
                <Field label={t('web.backups.targetEditor.smb.hostLabel')} className="flex-1">
                  <Input
                    value={(config.host as string) ?? ''}
                    onChange={(e) => patch({ host: e.target.value })}
                    placeholder={t('web.backups.targetEditor.smb.hostPlaceholder')}
                    className="h-8 font-mono"
                  />
                </Field>
                <Field label={t('web.backups.targetEditor.smb.portLabel')} className="w-24">
                  <Input
                    type="number"
                    value={(config.port as number) ?? 445}
                    onChange={(e) => patch({ port: Number(e.target.value) || 445 })}
                    className="h-8"
                  />
                </Field>
              </FieldRow>
              <Field
                label={t('web.backups.targetEditor.smb.shareLabel')}
                hint={t('web.backups.targetEditor.smb.shareHint')}
              >
                <Input
                  value={(config.share as string) ?? ''}
                  onChange={(e) => patch({ share: e.target.value })}
                  placeholder={t('web.backups.targetEditor.smb.sharePlaceholder')}
                  className="h-8"
                />
              </Field>
              <FieldRow>
                <Field label={t('web.backups.targetEditor.smb.userLabel')} className="flex-1">
                  <Input
                    value={(config.user as string) ?? ''}
                    onChange={(e) => patch({ user: e.target.value })}
                    className="h-8"
                  />
                </Field>
                <Field label={t('web.backups.targetEditor.smb.passwordLabel')} className="flex-1">
                  <Input
                    type="password"
                    value={(config.password as string) ?? ''}
                    onChange={(e) => patch({ password: e.target.value })}
                    className="h-8"
                  />
                </Field>
              </FieldRow>
              <Field
                label={t('web.backups.targetEditor.smb.pathPrefixLabel')}
                hint={t('web.backups.targetEditor.smb.pathPrefixHint')}
              >
                <Input
                  value={(config.path_prefix as string) ?? ''}
                  onChange={(e) => patch({ path_prefix: e.target.value })}
                  placeholder={t('web.backups.targetEditor.smb.pathPrefixPlaceholder')}
                  className="h-8 font-mono"
                />
              </Field>
            </>
          )}

          {kind === 's3' && (
            <>
              <Field
                label={t('web.backups.targetEditor.s3.endpointLabel')}
                hint={t('web.backups.targetEditor.s3.endpointHint')}
              >
                <Input
                  value={(config.endpoint as string) ?? ''}
                  onChange={(e) => patch({ endpoint: e.target.value })}
                  placeholder={t('web.backups.targetEditor.s3.endpointPlaceholder')}
                  className="h-8 font-mono"
                />
              </Field>
              <FieldRow>
                <Field
                  label={t('web.backups.targetEditor.s3.regionLabel')}
                  className="flex-1"
                  hint={t('web.backups.targetEditor.s3.regionHint')}
                >
                  <Input
                    value={(config.region as string) ?? ''}
                    onChange={(e) => patch({ region: e.target.value })}
                    placeholder={t('web.backups.targetEditor.s3.regionPlaceholder')}
                    className="h-8 font-mono"
                  />
                </Field>
                <Field label={t('web.backups.targetEditor.s3.bucketLabel')} className="flex-1">
                  <Input
                    value={(config.bucket as string) ?? ''}
                    onChange={(e) => patch({ bucket: e.target.value })}
                    placeholder={t('web.backups.targetEditor.s3.bucketPlaceholder')}
                    className="h-8 font-mono"
                  />
                </Field>
              </FieldRow>
              <Field label={t('web.backups.targetEditor.s3.accessKeyLabel')}>
                <Input
                  value={(config.access_key as string) ?? ''}
                  onChange={(e) => patch({ access_key: e.target.value })}
                  className="h-8 font-mono"
                  autoComplete="off"
                />
              </Field>
              <Field
                label={t('web.backups.targetEditor.s3.secretKeyLabel')}
                hint={t('web.backups.targetEditor.s3.secretKeyHint')}
              >
                <Input
                  type="password"
                  value={(config.secret_key as string) ?? ''}
                  onChange={(e) => patch({ secret_key: e.target.value })}
                  className="h-8 font-mono"
                  autoComplete="off"
                />
              </Field>
              <Field
                label={t('web.backups.targetEditor.s3.pathPrefixLabel')}
                hint={t('web.backups.targetEditor.s3.pathPrefixHint')}
              >
                <Input
                  value={(config.path_prefix as string) ?? ''}
                  onChange={(e) => patch({ path_prefix: e.target.value })}
                  placeholder={t('web.backups.targetEditor.s3.pathPrefixPlaceholder')}
                  className="h-8 font-mono"
                />
              </Field>
              <div className="flex gap-4 text-[12px]">
                <label className="flex items-center gap-2">
                  <Switch
                    checked={(config.use_ssl as boolean) ?? true}
                    onCheckedChange={(v) => patch({ use_ssl: v })}
                    className="scale-75"
                  />
                  {t('web.backups.targetEditor.s3.useHttps')}
                </label>
                <label className="flex items-center gap-2">
                  <Switch
                    checked={(config.path_style as boolean) ?? false}
                    onCheckedChange={(v) => patch({ path_style: v })}
                    className="scale-75"
                  />
                  {t('web.backups.targetEditor.s3.pathStyle')}
                </label>
              </div>
            </>
          )}

          {kind === 'webdav' && (
            <>
              <Field
                label={t('web.backups.targetEditor.webdav.baseUrlLabel')}
                hint={t('web.backups.targetEditor.webdav.baseUrlHint')}
              >
                <Input
                  value={(config.base_url as string) ?? ''}
                  onChange={(e) => patch({ base_url: e.target.value })}
                  placeholder={t('web.backups.targetEditor.webdav.baseUrlPlaceholder')}
                  className="h-8 font-mono"
                />
              </Field>
              <FieldRow>
                <Field label={t('web.backups.targetEditor.webdav.userLabel')} className="flex-1">
                  <Input
                    value={(config.user as string) ?? ''}
                    onChange={(e) => patch({ user: e.target.value })}
                    className="h-8"
                  />
                </Field>
                <Field label={t('web.backups.targetEditor.webdav.passwordLabel')} className="flex-1">
                  <Input
                    type="password"
                    value={(config.password as string) ?? ''}
                    onChange={(e) => patch({ password: e.target.value })}
                    className="h-8"
                  />
                </Field>
              </FieldRow>
              <Field
                label={t('web.backups.targetEditor.webdav.pathPrefixLabel')}
                hint={t('web.backups.targetEditor.webdav.pathPrefixHint')}
              >
                <Input
                  value={(config.path_prefix as string) ?? ''}
                  onChange={(e) => patch({ path_prefix: e.target.value })}
                  placeholder={t('web.backups.targetEditor.webdav.pathPrefixPlaceholder')}
                  className="h-8 font-mono"
                />
              </Field>
            </>
          )}

          {kind === 'sftp' && (
            <>
              <FieldRow>
                <Field label={t('web.backups.targetEditor.sftp.hostLabel')} className="flex-1">
                  <Input
                    value={(config.host as string) ?? ''}
                    onChange={(e) => patch({ host: e.target.value })}
                    placeholder={t('web.backups.targetEditor.sftp.hostPlaceholder')}
                    className="h-8 font-mono"
                  />
                </Field>
                <Field label={t('web.backups.targetEditor.sftp.portLabel')} className="w-24">
                  <Input
                    type="number"
                    value={(config.port as number) ?? 22}
                    onChange={(e) => patch({ port: Number(e.target.value) || 22 })}
                    className="h-8"
                  />
                </Field>
              </FieldRow>
              <Field label={t('web.backups.targetEditor.sftp.userLabel')}>
                <Input
                  value={(config.user as string) ?? ''}
                  onChange={(e) => patch({ user: e.target.value })}
                  className="h-8 font-mono"
                />
              </Field>
              <Field
                label={t('web.backups.targetEditor.sftp.passwordLabel')}
                hint={t('web.backups.targetEditor.sftp.passwordHint')}
              >
                <Input
                  type="password"
                  value={(config.password as string) ?? ''}
                  onChange={(e) => patch({ password: e.target.value })}
                  className="h-8"
                />
              </Field>
              <Field
                label={t('web.backups.targetEditor.sftp.privateKeyLabel')}
                hint={t('web.backups.targetEditor.sftp.privateKeyHint')}
              >
                <textarea
                  value={(config.private_key as string) ?? ''}
                  onChange={(e) => patch({ private_key: e.target.value })}
                  placeholder={t('web.backups.targetEditor.sftp.privateKeyPlaceholder')}
                  rows={4}
                  className="w-full px-2 py-1.5 rounded-md border border-border bg-card text-[11px] font-mono"
                />
              </Field>
              <Field
                label={t('web.backups.targetEditor.sftp.hostKeyLabel')}
                hint={t('web.backups.targetEditor.sftp.hostKeyHint')}
              >
                <textarea
                  value={(config.host_key as string) ?? ''}
                  onChange={(e) => patch({ host_key: e.target.value })}
                  placeholder={t('web.backups.targetEditor.sftp.hostKeyPlaceholder')}
                  rows={2}
                  className="w-full px-2 py-1.5 rounded-md border border-border bg-card text-[11px] font-mono"
                />
              </Field>
              <Field
                label={t('web.backups.targetEditor.sftp.pathPrefixLabel')}
                hint={t('web.backups.targetEditor.sftp.pathPrefixHint')}
              >
                <Input
                  value={(config.path_prefix as string) ?? ''}
                  onChange={(e) => patch({ path_prefix: e.target.value })}
                  placeholder={t('web.backups.targetEditor.sftp.pathPrefixPlaceholder')}
                  className="h-8 font-mono"
                />
              </Field>
            </>
          )}

          {kind === 'rclone' && (
            <>
              <div className="rounded-md border border-state-idle/40 bg-state-idle/10 p-2 text-[11px]">
                <Trans
                  i18nKey="web.backups.targetEditor.rclone.rcloneHint"
                  components={{
                    1: <code className="text-foreground" />,
                    3: <code className="text-foreground" />,
                    5: <code className="text-foreground" />,
                  }}
                />
              </div>
              <Field
                label={t('web.backups.targetEditor.rclone.remoteLabel')}
                hint={t('web.backups.targetEditor.rclone.remoteHint')}
              >
                <Input
                  value={(config.remote as string) ?? ''}
                  onChange={(e) => patch({ remote: e.target.value })}
                  placeholder={t('web.backups.targetEditor.rclone.remotePlaceholder')}
                  className="h-8 font-mono"
                />
              </Field>
              <Field
                label={t('web.backups.targetEditor.rclone.pathPrefixLabel')}
                hint={t('web.backups.targetEditor.rclone.pathPrefixHint')}
              >
                <Input
                  value={(config.path_prefix as string) ?? ''}
                  onChange={(e) => patch({ path_prefix: e.target.value })}
                  placeholder={t('web.backups.targetEditor.rclone.pathPrefixPlaceholder')}
                  className="h-8 font-mono"
                />
              </Field>
              <Field
                label={t('web.backups.targetEditor.rclone.binaryPathLabel')}
                hint={t('web.backups.targetEditor.rclone.binaryPathHint')}
              >
                <Input
                  value={(config.binary_path as string) ?? ''}
                  onChange={(e) => patch({ binary_path: e.target.value })}
                  placeholder={t('web.backups.targetEditor.rclone.binaryPathPlaceholder')}
                  className="h-8 font-mono"
                />
              </Field>
              <Field
                label={t('web.backups.targetEditor.rclone.configPathLabel')}
                hint={t('web.backups.targetEditor.rclone.configPathHint')}
              >
                <Input
                  value={(config.config_path as string) ?? ''}
                  onChange={(e) => patch({ config_path: e.target.value })}
                  placeholder={t('web.backups.targetEditor.rclone.configPathPlaceholder')}
                  className="h-8 font-mono"
                />
              </Field>
            </>
          )}
        </div>

        <label className="flex items-center gap-2 text-[12px] border-t border-border pt-3">
          <Switch
            checked={enabled}
            onCheckedChange={setEnabled}
            className="scale-75"
          />
          {t('web.backups.targetEditor.enableImmediately')}
        </label>
      </div>

      <DialogFooter>
        <Button onClick={submit} disabled={busy}>
          {busy
            ? t('web.backups.targetEditor.creating')
            : t('web.backups.targetEditor.create')}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

// targetSummary returns a human-readable one-liner for table rows.
export function targetSummary(t: TargetSpec): string {
  if (t.kind === 'local') {
    return String(t.config?.root ?? '(default local dir)')
  }
  if (t.kind === 'smb') {
    const host = t.config?.host
    const share = t.config?.share
    const user = t.config?.user
    const prefix = t.config?.path_prefix
    return `//${host ?? '?'}/${share ?? '?'} as ${user ?? '?'}${prefix ? ` → ${prefix}/` : ''}`
  }
  if (t.kind === 's3') {
    const ep = t.config?.endpoint
    const bucket = t.config?.bucket
    const prefix = t.config?.path_prefix
    return `s3://${bucket ?? '?'}@${ep ?? '?'}${prefix ? `/${prefix}` : ''}`
  }
  if (t.kind === 'webdav') {
    const url = t.config?.base_url
    const user = t.config?.user
    const prefix = t.config?.path_prefix
    return `${url ?? '?'} as ${user ?? '?'}${prefix ? ` → ${prefix}/` : ''}`
  }
  if (t.kind === 'sftp') {
    const host = t.config?.host
    const user = t.config?.user
    const prefix = t.config?.path_prefix
    return `${user ?? '?'}@${host ?? '?'}${prefix ? `:${prefix}` : ''}`
  }
  if (t.kind === 'rclone') {
    const remote = t.config?.remote
    const prefix = t.config?.path_prefix
    return `rclone:${remote ?? '?'}${prefix ? `:${prefix}` : ''}`
  }
  return JSON.stringify(t.config)
}

// ── small helpers ────────────────────────────────────────────────

function Field({
  label,
  hint,
  className,
  children,
}: {
  label: string
  hint?: string
  className?: string
  children: React.ReactNode
}) {
  return (
    <div className={'flex flex-col gap-1.5 ' + (className ?? '')}>
      <Label className="text-[12px]">{label}</Label>
      {children}
      {hint && <p className="text-[10.5px] text-muted-foreground">{hint}</p>}
    </div>
  )
}

function FieldRow({ children }: { children: React.ReactNode }) {
  return <div className="flex gap-2">{children}</div>
}

function msgFromAPIError(err: APIError): string {
  if (
    err.body &&
    typeof err.body === 'object' &&
    'error' in err.body &&
    typeof (err.body as { error: unknown }).error === 'string'
  ) {
    return (err.body as { error: string }).error
  }
  return err.message
}
