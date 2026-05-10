import { useState } from 'react'
import { toast } from 'sonner'

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
      toast.success('Target created')
      await onCreated()
    } catch (err) {
      const msg =
        err instanceof APIError
          ? msgFromAPIError(err)
          : err instanceof Error
            ? err.message
            : 'Unknown error'
      toast.error('Create failed', { description: msg })
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
        <DialogTitle>New backup target</DialogTitle>
      </DialogHeader>

      <div className="flex flex-col gap-4">
        {/* Kind picker as cards */}
        <div className="flex flex-col gap-1.5">
          <Label className="text-[12px]">Where do you want to back up to?</Label>
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
          <Label className="text-[12px]">ID (optional)</Label>
          <Input
            value={id}
            onChange={(e) => setId(e.target.value)}
            placeholder="auto-generated if blank, e.g. tgt_xxx"
            className="h-8 font-mono"
          />
        </div>

        {/* Per-kind config */}
        <div className="flex flex-col gap-3 border-t border-border pt-3">
          {kind === 'local' && (
            <Field label="Root directory" hint="Empty = cfg.backup.local_dir (~/.opendray/backups)">
              <Input
                value={(config.root as string) ?? ''}
                onChange={(e) => patch({ root: e.target.value })}
                placeholder="~/backups/opendray  or  /mnt/external-hdd/opendray"
                className="h-8 font-mono"
              />
            </Field>
          )}

          {kind === 'smb' && (
            <>
              <FieldRow>
                <Field label="Host" className="flex-1">
                  <Input
                    value={(config.host as string) ?? ''}
                    onChange={(e) => patch({ host: e.target.value })}
                    placeholder="192.168.9.8"
                    className="h-8 font-mono"
                  />
                </Field>
                <Field label="Port" className="w-24">
                  <Input
                    type="number"
                    value={(config.port as number) ?? 445}
                    onChange={(e) => patch({ port: Number(e.target.value) || 445 })}
                    className="h-8"
                  />
                </Field>
              </FieldRow>
              <Field label="Share" hint="Top-level share name on the SMB server">
                <Input
                  value={(config.share as string) ?? ''}
                  onChange={(e) => patch({ share: e.target.value })}
                  placeholder="Claude_Workspace"
                  className="h-8"
                />
              </Field>
              <FieldRow>
                <Field label="User" className="flex-1">
                  <Input
                    value={(config.user as string) ?? ''}
                    onChange={(e) => patch({ user: e.target.value })}
                    className="h-8"
                  />
                </Field>
                <Field label="Password" className="flex-1">
                  <Input
                    type="password"
                    value={(config.password as string) ?? ''}
                    onChange={(e) => patch({ password: e.target.value })}
                    className="h-8"
                  />
                </Field>
              </FieldRow>
              <Field label="Path prefix" hint="Sub-folder under the share root (optional)">
                <Input
                  value={(config.path_prefix as string) ?? ''}
                  onChange={(e) => patch({ path_prefix: e.target.value })}
                  placeholder="opendray/backups"
                  className="h-8 font-mono"
                />
              </Field>
            </>
          )}

          {kind === 's3' && (
            <>
              <Field
                label="Endpoint"
                hint="Host (no protocol). AWS: s3.amazonaws.com · R2: <accountid>.r2.cloudflarestorage.com · MinIO: minio.local:9000"
              >
                <Input
                  value={(config.endpoint as string) ?? ''}
                  onChange={(e) => patch({ endpoint: e.target.value })}
                  placeholder="s3.amazonaws.com"
                  className="h-8 font-mono"
                />
              </Field>
              <FieldRow>
                <Field label="Region" className="flex-1" hint="AWS only; R2 use 'auto'">
                  <Input
                    value={(config.region as string) ?? ''}
                    onChange={(e) => patch({ region: e.target.value })}
                    placeholder="us-east-1 / auto"
                    className="h-8 font-mono"
                  />
                </Field>
                <Field label="Bucket" className="flex-1">
                  <Input
                    value={(config.bucket as string) ?? ''}
                    onChange={(e) => patch({ bucket: e.target.value })}
                    placeholder="opendray-backups"
                    className="h-8 font-mono"
                  />
                </Field>
              </FieldRow>
              <Field label="Access key">
                <Input
                  value={(config.access_key as string) ?? ''}
                  onChange={(e) => patch({ access_key: e.target.value })}
                  className="h-8 font-mono"
                  autoComplete="off"
                />
              </Field>
              <Field label="Secret key" hint="Stored AES-256-GCM encrypted; never echoed back">
                <Input
                  type="password"
                  value={(config.secret_key as string) ?? ''}
                  onChange={(e) => patch({ secret_key: e.target.value })}
                  className="h-8 font-mono"
                  autoComplete="off"
                />
              </Field>
              <Field label="Path prefix" hint="Object-key prefix (optional)">
                <Input
                  value={(config.path_prefix as string) ?? ''}
                  onChange={(e) => patch({ path_prefix: e.target.value })}
                  placeholder="opendray/backups"
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
                  Use HTTPS
                </label>
                <label className="flex items-center gap-2">
                  <Switch
                    checked={(config.path_style as boolean) ?? false}
                    onCheckedChange={(v) => patch({ path_style: v })}
                    className="scale-75"
                  />
                  Path-style addressing (legacy / MinIO)
                </label>
              </div>
            </>
          )}

          {kind === 'webdav' && (
            <>
              <Field
                label="Base URL"
                hint="Full URL including any path. Examples: https://cloud.example.com/remote.php/dav/files/me/ (Nextcloud), https://nas.local:5006/ (Synology), https://dav.jianguoyun.com/dav/ (Jianguoyun / 坚果云)"
              >
                <Input
                  value={(config.base_url as string) ?? ''}
                  onChange={(e) => patch({ base_url: e.target.value })}
                  placeholder="https://cloud.example.com/remote.php/dav/files/<user>/"
                  className="h-8 font-mono"
                />
              </Field>
              <FieldRow>
                <Field label="User" className="flex-1">
                  <Input
                    value={(config.user as string) ?? ''}
                    onChange={(e) => patch({ user: e.target.value })}
                    className="h-8"
                  />
                </Field>
                <Field label="Password" className="flex-1">
                  <Input
                    type="password"
                    value={(config.password as string) ?? ''}
                    onChange={(e) => patch({ password: e.target.value })}
                    className="h-8"
                  />
                </Field>
              </FieldRow>
              <Field label="Path prefix" hint="Sub-folder under the base URL (optional)">
                <Input
                  value={(config.path_prefix as string) ?? ''}
                  onChange={(e) => patch({ path_prefix: e.target.value })}
                  placeholder="opendray/backups"
                  className="h-8 font-mono"
                />
              </Field>
            </>
          )}

          {kind === 'sftp' && (
            <>
              <FieldRow>
                <Field label="Host" className="flex-1">
                  <Input
                    value={(config.host as string) ?? ''}
                    onChange={(e) => patch({ host: e.target.value })}
                    placeholder="vps.example.com"
                    className="h-8 font-mono"
                  />
                </Field>
                <Field label="Port" className="w-24">
                  <Input
                    type="number"
                    value={(config.port as number) ?? 22}
                    onChange={(e) => patch({ port: Number(e.target.value) || 22 })}
                    className="h-8"
                  />
                </Field>
              </FieldRow>
              <Field label="User">
                <Input
                  value={(config.user as string) ?? ''}
                  onChange={(e) => patch({ user: e.target.value })}
                  className="h-8 font-mono"
                />
              </Field>
              <Field
                label="Password"
                hint="Either password OR private key required. If both, password is treated as the key passphrase."
              >
                <Input
                  type="password"
                  value={(config.password as string) ?? ''}
                  onChange={(e) => patch({ password: e.target.value })}
                  className="h-8"
                />
              </Field>
              <Field
                label="Private key (PEM)"
                hint="Paste contents of an OpenSSH/PEM private key (e.g. ~/.ssh/id_ed25519). Leave blank for password-only auth."
              >
                <textarea
                  value={(config.private_key as string) ?? ''}
                  onChange={(e) => patch({ private_key: e.target.value })}
                  placeholder="-----BEGIN OPENSSH PRIVATE KEY-----..."
                  rows={4}
                  className="w-full px-2 py-1.5 rounded-md border border-border bg-card text-[11px] font-mono"
                />
              </Field>
              <Field
                label="Host key (pinning)"
                hint="OpenSSH-style server public key (run `ssh-keyscan host` to obtain). Leave blank to disable pinning (NOT recommended outside LAN)."
              >
                <textarea
                  value={(config.host_key as string) ?? ''}
                  onChange={(e) => patch({ host_key: e.target.value })}
                  placeholder="ssh-ed25519 AAAA..."
                  rows={2}
                  className="w-full px-2 py-1.5 rounded-md border border-border bg-card text-[11px] font-mono"
                />
              </Field>
              <Field label="Path prefix" hint="Absolute or relative to user home (optional)">
                <Input
                  value={(config.path_prefix as string) ?? ''}
                  onChange={(e) => patch({ path_prefix: e.target.value })}
                  placeholder="/var/backups/opendray  or  opendray-backups"
                  className="h-8 font-mono"
                />
              </Field>
            </>
          )}

          {kind === 'rclone' && (
            <>
              <div className="rounded-md border border-state-idle/40 bg-state-idle/10 p-2 text-[11px]">
                Requires the <code className="text-foreground">rclone</code>{' '}
                CLI installed on the opendray host. First configure your
                remote with{' '}
                <code className="text-foreground">rclone config</code>, then
                use the remote name below. opendray invokes{' '}
                <code className="text-foreground">rclone rcat / cat / lsd</code>{' '}
                under the hood.
              </div>
              <Field
                label="Remote name"
                hint="Name from `rclone config` (no colon). Example: gdrive, onedrive, dropbox-personal, baidu-pan"
              >
                <Input
                  value={(config.remote as string) ?? ''}
                  onChange={(e) => patch({ remote: e.target.value })}
                  placeholder="gdrive"
                  className="h-8 font-mono"
                />
              </Field>
              <Field label="Path prefix" hint="Sub-folder under the remote root (optional)">
                <Input
                  value={(config.path_prefix as string) ?? ''}
                  onChange={(e) => patch({ path_prefix: e.target.value })}
                  placeholder="opendray/backups"
                  className="h-8 font-mono"
                />
              </Field>
              <Field
                label="Binary path"
                hint="Override `which rclone`. Empty uses PATH lookup."
              >
                <Input
                  value={(config.binary_path as string) ?? ''}
                  onChange={(e) => patch({ binary_path: e.target.value })}
                  placeholder="/opt/homebrew/bin/rclone"
                  className="h-8 font-mono"
                />
              </Field>
              <Field
                label="Config path"
                hint="Override --config (default ~/.config/rclone/rclone.conf or ~/.rclone.conf)"
              >
                <Input
                  value={(config.config_path as string) ?? ''}
                  onChange={(e) => patch({ config_path: e.target.value })}
                  placeholder="leave blank for rclone default"
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
          Enable immediately (otherwise saved as disabled — useful for "configure
          now, turn on later")
        </label>
      </div>

      <DialogFooter>
        <Button onClick={submit} disabled={busy}>
          {busy ? 'Creating…' : 'Create target'}
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
