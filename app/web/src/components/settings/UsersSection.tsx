import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { APIError } from '@/lib/api'
import {
  createUser,
  deleteUser,
  listUsers,
  updateUser,
  type UserRow,
} from '@/lib/users'
import type { Role } from '@/stores/auth'
import { useAuth } from '@/stores/auth'
import { cn } from '@/lib/utils'

const errText = (e: unknown) =>
  e instanceof APIError || e instanceof Error ? e.message : String(e)

/**
 * Admin-only operator logins. Roles gate the admin pages and their APIs;
 * they do not isolate users from each other — every session runs as the
 * same OS user and every login sees every project.
 */
export function UsersSection() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const me = useAuth((s) => s.username)
  const { data: users, isLoading, error } = useQuery({
    queryKey: ['users'],
    queryFn: listUsers,
  })
  const refresh = () => qc.invalidateQueries({ queryKey: ['users'] })

  const patch = useMutation({
    mutationFn: ({ id, body }: { id: string; body: Parameters<typeof updateUser>[1] }) =>
      updateUser(id, body),
    onSuccess: refresh,
    onError: (e) => toast.error(errText(e)),
  })
  const remove = useMutation({
    mutationFn: deleteUser,
    onSuccess: refresh,
    onError: (e) => toast.error(errText(e)),
  })

  const resetPassword = (u: UserRow) => {
    const pw = window.prompt(t('web.settings.users.resetPrompt', { user: u.username }))
    if (!pw) return
    patch.mutate(
      { id: u.id, body: { password: pw } },
      { onSuccess: () => toast.success(t('web.settings.users.passwordReset')) },
    )
  }

  return (
    <div>
      <header className="mb-5 pb-3 border-b border-border">
        <h2 className="text-[15px] font-semibold tracking-tight">
          {t('web.settings.users.title')}
        </h2>
        <p className="text-[12px] text-muted-foreground mt-0.5">
          {t('web.settings.users.description')}
        </p>
      </header>

      <CreateUserForm onCreated={refresh} />

      {isLoading && <p className="text-[12px] text-muted-foreground">…</p>}
      {error && <p className="text-[12px] text-rose-400">{errText(error)}</p>}

      <div className="mt-5 border border-border rounded-md divide-y divide-border">
        {users?.map((u) => {
          const self = u.username.toLowerCase() === (me ?? '').toLowerCase()
          return (
            <div key={u.id} className="flex flex-wrap items-center gap-3 px-3 py-2.5">
              <div className="min-w-0 flex-1">
                <div className={cn('text-[13px] font-medium', u.disabled && 'line-through opacity-60')}>
                  {u.username}
                  {u.display_name && (
                    <span className="ml-2 text-muted-foreground font-normal">{u.display_name}</span>
                  )}
                </div>
                <div className="text-[11px] text-muted-foreground">
                  {u.primary
                    ? t('web.settings.users.primary')
                    : u.last_login_at
                      ? t('web.settings.users.lastLogin', {
                          when: new Date(u.last_login_at).toLocaleString(),
                        })
                      : t('web.settings.users.neverLoggedIn')}
                </div>
              </div>
              {u.primary ? (
                <span className="text-[11px] px-2 py-0.5 rounded border border-border">
                  {t('web.settings.users.roles.admin')}
                </span>
              ) : (
                <>
                  <select
                    className="text-[12px] bg-background border border-border rounded px-2 py-1"
                    value={u.role}
                    disabled={self}
                    onChange={(e) =>
                      patch.mutate({ id: u.id, body: { role: e.target.value as Role } })
                    }
                  >
                    <option value="member">{t('web.settings.users.roles.member')}</option>
                    <option value="admin">{t('web.settings.users.roles.admin')}</option>
                  </select>
                  <button
                    type="button"
                    className="text-[12px] px-2 py-1 rounded border border-border hover:bg-card disabled:opacity-50"
                    disabled={self}
                    onClick={() => patch.mutate({ id: u.id, body: { disabled: !u.disabled } })}
                  >
                    {u.disabled ? t('web.settings.users.enable') : t('web.settings.users.disable')}
                  </button>
                  <button
                    type="button"
                    className="text-[12px] px-2 py-1 rounded border border-border hover:bg-card"
                    onClick={() => resetPassword(u)}
                  >
                    {t('web.settings.users.resetPassword')}
                  </button>
                  <button
                    type="button"
                    className="text-[12px] px-2 py-1 rounded border border-border text-rose-400 hover:bg-card disabled:opacity-50"
                    disabled={self}
                    onClick={() => {
                      if (window.confirm(t('web.settings.users.deleteConfirm', { user: u.username })))
                        remove.mutate(u.id)
                    }}
                  >
                    {t('web.settings.users.delete')}
                  </button>
                </>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

function CreateUserForm({ onCreated }: { onCreated: () => void }) {
  const { t } = useTranslation()
  const [username, setUsername] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<Role>('member')
  const create = useMutation({
    mutationFn: createUser,
    onSuccess: (u) => {
      toast.success(t('web.settings.users.created', { user: u.username }))
      setUsername('')
      setDisplayName('')
      setPassword('')
      setRole('member')
      onCreated()
    },
    onError: (e) => toast.error(errText(e)),
  })
  const submit = (e: FormEvent) => {
    e.preventDefault()
    create.mutate({ username, password, role, display_name: displayName })
  }
  const input =
    'text-[12px] bg-background border border-border rounded px-2 py-1.5 min-w-0'
  return (
    <form onSubmit={submit} className="flex flex-wrap items-end gap-2">
      <input
        className={cn(input, 'w-36')}
        placeholder={t('web.settings.users.username')}
        value={username}
        onChange={(e) => setUsername(e.target.value)}
        autoComplete="off"
        required
      />
      <input
        className={cn(input, 'w-36')}
        placeholder={t('web.settings.users.displayName')}
        value={displayName}
        onChange={(e) => setDisplayName(e.target.value)}
      />
      <input
        className={cn(input, 'w-40')}
        type="password"
        placeholder={t('web.settings.users.password')}
        value={password}
        onChange={(e) => setPassword(e.target.value)}
        autoComplete="new-password"
        required
      />
      <select
        className={input}
        value={role}
        onChange={(e) => setRole(e.target.value as Role)}
      >
        <option value="member">{t('web.settings.users.roles.member')}</option>
        <option value="admin">{t('web.settings.users.roles.admin')}</option>
      </select>
      <button
        type="submit"
        className="text-[12px] px-3 py-1.5 rounded-md border border-border hover:bg-card disabled:opacity-50"
        disabled={create.isPending}
      >
        {t('web.settings.users.add')}
      </button>
    </form>
  )
}
