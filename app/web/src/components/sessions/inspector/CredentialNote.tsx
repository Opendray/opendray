import { KeyRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import type { GitRemote } from '@/lib/githost'

// Which identity a PR/issue call authenticated as used to be invisible:
// the panel said nothing while a token was working, and on failure
// showed the forge's error verbatim. That error never mentions the
// credential — GitHub answers a token that lacks the repo with a 403
// reading "Write access to repository not granted", even for a read —
// so the one fact that would explain it was the one fact not on screen.
//
// Deliberately quiet: with an owner-scoped credential matching exactly
// there is nothing to explain, and a line on every panel would be noise
// that trains people to ignore it. It speaks only when the answer is
// non-obvious — a host-wide credential standing in for an owner that
// has none, or a call that actually failed.
export function CredentialNote({
  remote,
  failed,
}: {
  remote: GitRemote | undefined
  /** True when the call errored — then the credential is always worth
   * naming, fallback or not. */
  failed?: boolean
}) {
  const { t } = useTranslation()
  if (!remote?.has_token) return null
  if (!remote.token_is_fallback && !failed) return null

  const scope = remote.token_owner
    ? `${remote.host}/${remote.token_owner}`
    : remote.host

  return (
    <div className="flex items-start gap-1.5 px-1 py-1 text-[11px] text-muted-foreground/80">
      <KeyRound className="mt-0.5 size-3 shrink-0 text-muted-foreground/60" />
      <span>
        {remote.token_is_fallback
          ? t('web.sessions.inspector.git.credentialFallback', {
              owner: remote.owner,
              scope,
            })
          : t('web.sessions.inspector.git.credentialUsed', { scope })}
        {remote.token_name && ` (${remote.token_name})`}
      </span>
    </div>
  )
}
