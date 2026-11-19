import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'

import { getMe } from '@/lib/users'
import { useAuth } from '@/stores/auth'

/** Keeps the persisted role in step with the server. A role change
 * revokes the token, but a session persisted before roles existed has
 * none until /auth/me answers. */
export function useIdentitySync() {
  const token = useAuth((s) => s.token)
  const setIdentity = useAuth((s) => s.setIdentity)
  const { data } = useQuery({
    queryKey: ['auth-me', token],
    queryFn: getMe,
    enabled: !!token,
    staleTime: 5 * 60 * 1000,
  })
  useEffect(() => {
    if (data?.role) setIdentity(data.role, data.primary)
  }, [data, setIdentity])
}
