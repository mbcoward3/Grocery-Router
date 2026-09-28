import { useQuery } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { ApiRequestError, getSession, setHouseholdId } from '../../api/client'
import { BrandMark } from '../../ui/icons'

export const sessionQueryKey = ['session'] as const

export function AuthGate({ children }: { children: ReactNode }) {
  const session = useQuery({
    queryKey: sessionQueryKey,
    queryFn: getSession,
    retry: false,
    staleTime: 60_000,
  })

  if (session.isPending) return <AuthState title="Opening Grocery Router…" />
  if (session.error) {
    if (session.error instanceof ApiRequestError && session.error.status === 401) {
      const returnTo = window.location.pathname === '/sign-in'
        ? '/'
        : `${window.location.pathname}${window.location.search}`
      return (
        <AuthState title="Your household groceries, together.">
          <a className="button primary" href={`/api/v2/auth/google/start?returnTo=${encodeURIComponent(returnTo)}`}>
            Continue with Google
          </a>
        </AuthState>
      )
    }
    return <AuthState title="Grocery Router is unavailable."><p>Try again in a moment.</p></AuthState>
  }

  const household = session.data.households[0]
  if (!household) return <AuthState title="No household access."><p>Ask a household owner for access.</p></AuthState>
  setHouseholdId(household.householdID)
  return children
}

function AuthState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <main className="auth-shell">
      <section className="auth-card">
        <span className="auth-mark" aria-hidden="true"><BrandMark /></span>
        <div className="eyebrow">Grocery Router</div>
        <h1>{title}</h1>
        {children && <div className="auth-actions">{children}</div>}
      </section>
    </main>
  )
}
