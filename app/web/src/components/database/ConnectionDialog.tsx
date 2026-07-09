import { useEffect, useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, CheckCircle2, Loader2, XCircle } from 'lucide-react'
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
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  createConnection,
  updateConnection,
  testConnectionParams,
  DB_DRIVERS,
  DB_DEFAULT_PORTS,
  driverUsesServer,
  type DBConnection,
  type DBConnectionInput,
  type DBDriver,
  type DBPingResult,
} from '@/lib/database'

interface ConnectionDialogProps {
  cwd: string
  open: boolean
  // connection set = edit mode; null = create mode.
  connection: DBConnection | null
  onOpenChange: (v: boolean) => void
}

const SSL_MODES = ['disable', 'prefer', 'require', 'verify-ca', 'verify-full']

interface FormState {
  name: string
  driver: DBDriver
  host: string
  port: string
  db_name: string
  username: string
  password: string
  ssl_mode: string
  read_only: boolean
}

const EMPTY: FormState = {
  name: '',
  driver: 'postgres',
  host: '',
  port: '5432',
  db_name: '',
  username: '',
  password: '',
  ssl_mode: 'prefer',
  read_only: false,
}

// ConnectionDialog registers or edits a database connection. Password is
// write-only: in edit mode it starts blank and an empty submit keeps the
// stored secret. A Test button probes connectivity before saving and
// surfaces a superuser warning (operator rule: never run project work as
// the PG superuser).
export function ConnectionDialog({
  cwd,
  open,
  connection,
  onOpenChange,
}: ConnectionDialogProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const editing = !!connection
  const [form, setForm] = useState<FormState>(EMPTY)
  const [ping, setPing] = useState<DBPingResult | null>(null)

  useEffect(() => {
    if (!open) return
    setPing(null)
    if (connection) {
      setForm({
        name: connection.name,
        driver: connection.driver,
        host: connection.host,
        port: String(connection.port),
        db_name: connection.db_name,
        username: connection.username,
        password: '',
        ssl_mode: connection.ssl_mode,
        read_only: connection.read_only,
      })
    } else {
      setForm(EMPTY)
    }
  }, [open, connection])

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm((f) => ({ ...f, [key]: value }))

  // Switching driver resets the port to that engine's default (file-based
  // SQLite has none) and clears the last ping.
  const onDriverChange = (driver: DBDriver) => {
    setPing(null)
    setForm((f) => ({
      ...f,
      driver,
      port: driverUsesServer(driver) ? String(DB_DEFAULT_PORTS[driver]) : '',
    }))
  }

  const isSqlite = form.driver === 'sqlite'

  const toInput = (): DBConnectionInput => ({
    cwd,
    name: form.name.trim(),
    driver: form.driver,
    host: isSqlite ? '' : form.host.trim(),
    port: isSqlite ? 0 : Number(form.port) || DB_DEFAULT_PORTS[form.driver],
    db_name: form.db_name.trim(),
    username: isSqlite ? '' : form.username.trim(),
    password: isSqlite ? '' : form.password,
    ssl_mode: isSqlite ? '' : form.ssl_mode,
    read_only: form.read_only,
  })

  const testMut = useMutation({
    mutationFn: () => testConnectionParams(toInput()),
    onSuccess: (res) => {
      setPing(res)
      if (!res.ok) toast.error(t('web.database.dialog.testFailed'))
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const saveMut = useMutation({
    mutationFn: () => {
      if (connection) {
        // Omit password when left blank so the stored secret is kept.
        const patch = isSqlite
          ? {
              name: form.name.trim(),
              db_name: form.db_name.trim(),
              read_only: form.read_only,
            }
          : {
              name: form.name.trim(),
              host: form.host.trim(),
              port: Number(form.port) || DB_DEFAULT_PORTS[form.driver],
              db_name: form.db_name.trim(),
              username: form.username.trim(),
              ssl_mode: form.ssl_mode,
              read_only: form.read_only,
              ...(form.password ? { password: form.password } : {}),
            }
        return updateConnection(connection.id, patch)
      }
      return createConnection(toInput())
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['db-connections', cwd] })
      toast.success(
        editing
          ? t('web.database.dialog.savedEdit')
          : t('web.database.dialog.savedCreate'),
      )
      onOpenChange(false)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (
      !form.name.trim() ||
      !form.db_name.trim() ||
      (!isSqlite && (!form.host.trim() || !form.username.trim()))
    ) {
      toast.error(t('web.database.dialog.missingFields'))
      return
    }
    saveMut.mutate()
  }

  const superuserWarn =
    ping?.is_superuser || form.username.trim() === 'linivek'

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {editing
              ? t('web.database.dialog.editTitle')
              : t('web.database.dialog.createTitle')}
          </DialogTitle>
          <DialogDescription>
            {t('web.database.dialog.description')}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={onSubmit} className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div className="col-span-2">
              <Label htmlFor="db-name">{t('web.database.dialog.name')}</Label>
              <Input
                id="db-name"
                value={form.name}
                onChange={(e) => set('name', e.target.value)}
                placeholder="prod-db"
              />
            </div>
            <div className="col-span-2">
              <Label>{t('web.database.dialog.driver')}</Label>
              <Select
                value={form.driver}
                onValueChange={(v) => onDriverChange(v as DBDriver)}
                disabled={editing}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {DB_DRIVERS.map((d) => (
                    <SelectItem key={d} value={d}>
                      {t(`web.database.dialog.drivers.${d}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {driverUsesServer(form.driver) && (
              <>
                <div className="col-span-2 sm:col-span-1">
                  <Label htmlFor="db-host">
                    {t('web.database.dialog.host')}
                  </Label>
                  <Input
                    id="db-host"
                    value={form.host}
                    onChange={(e) => set('host', e.target.value)}
                    placeholder="192.168.3.88"
                  />
                </div>
                <div>
                  <Label htmlFor="db-port">
                    {t('web.database.dialog.port')}
                  </Label>
                  <Input
                    id="db-port"
                    value={form.port}
                    onChange={(e) => set('port', e.target.value)}
                    inputMode="numeric"
                  />
                </div>
              </>
            )}
            <div className={isSqlite ? 'col-span-2' : ''}>
              <Label htmlFor="db-database">
                {isSqlite
                  ? t('web.database.dialog.filePath')
                  : t('web.database.dialog.database')}
              </Label>
              <Input
                id="db-database"
                value={form.db_name}
                onChange={(e) => set('db_name', e.target.value)}
                placeholder={isSqlite ? 'data/app.db' : ''}
              />
              {isSqlite && (
                <p className="mt-1 text-xs text-muted-foreground">
                  {t('web.database.dialog.filePathHint')}
                </p>
              )}
            </div>
            {driverUsesServer(form.driver) && (
              <>
                <div>
                  <Label htmlFor="db-user">
                    {t('web.database.dialog.username')}
                  </Label>
                  <Input
                    id="db-user"
                    value={form.username}
                    onChange={(e) => set('username', e.target.value)}
                  />
                </div>
                <div className="col-span-2 sm:col-span-1">
                  <Label htmlFor="db-pass">
                    {t('web.database.dialog.password')}
                  </Label>
                  <Input
                    id="db-pass"
                    type="password"
                    value={form.password}
                    onChange={(e) => set('password', e.target.value)}
                    placeholder={
                      editing && connection?.has_password
                        ? t('web.database.dialog.passwordKept')
                        : ''
                    }
                  />
                </div>
                <div>
                  <Label>{t('web.database.dialog.sslMode')}</Label>
                  <Select
                    value={form.ssl_mode}
                    onValueChange={(v) => set('ssl_mode', v)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {SSL_MODES.map((m) => (
                        <SelectItem key={m} value={m}>
                          {m}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </>
            )}
          </div>

          <div className="flex items-center gap-2">
            <Switch
              id="db-readonly"
              checked={form.read_only}
              onCheckedChange={(v) => set('read_only', v)}
            />
            <Label htmlFor="db-readonly" className="cursor-pointer">
              {t('web.database.dialog.readOnly')}
            </Label>
          </div>

          {superuserWarn && (
            <div className="flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 p-2 text-xs text-amber-700 dark:text-amber-400">
              <AlertTriangle className="mt-0.5 h-3.5 w-3.5 flex-none" />
              <span>{t('web.database.dialog.superuserWarning')}</span>
            </div>
          )}

          {ping && (
            <div
              className={`flex items-start gap-2 rounded-md border p-2 text-xs ${
                ping.ok
                  ? 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                  : 'border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-400'
              }`}
            >
              {ping.ok ? (
                <CheckCircle2 className="mt-0.5 h-3.5 w-3.5 flex-none" />
              ) : (
                <XCircle className="mt-0.5 h-3.5 w-3.5 flex-none" />
              )}
              <span>
                {ping.ok
                  ? t('web.database.dialog.testOk', {
                      version: ping.server_version ?? '?',
                      ms: ping.latency_ms,
                    })
                  : ping.error}
              </span>
            </div>
          )}

          <DialogFooter className="gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => testMut.mutate()}
              disabled={testMut.isPending}
            >
              {testMut.isPending && (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              )}
              {t('web.database.dialog.test')}
            </Button>
            <Button type="submit" disabled={saveMut.isPending}>
              {saveMut.isPending && (
                <Loader2 className="mr-1 h-3 w-3 animate-spin" />
              )}
              {t('web.database.dialog.save')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
