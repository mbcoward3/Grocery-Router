import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, Outlet, useRouterState } from '@tanstack/react-router'
import { getSession, logout } from '../api/client'
import { sessionQueryKey } from '../features/auth/auth-gate'
import { currentWeekQueryOptions } from '../features/week/queries'
import { BagIcon, BrandMark, CalendarIcon, HistoryIcon, RecipeIcon } from './icons'

export function AppShell() {
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const queryClient = useQueryClient()
  const session = useQuery({ queryKey: sessionQueryKey, queryFn: getSession, staleTime: 60_000 })
  const week = useQuery(currentWeekQueryOptions)
  const signOut = async () => {
    await logout()
    queryClient.clear()
    window.location.assign('/sign-in?signedOut=1')
  }
  const crumb = pathname === '/'
    ? 'Week'
    : pathname === '/groceries'
      ? 'Groceries'
      : pathname === '/recipes'
        ? 'Recipes'
        : pathname.startsWith('/history')
          ? 'History'
          : 'Recipe detail'

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <Link className="brand" to="/" aria-label="Grocery Router home">
          <span className="brand-mark" aria-hidden="true"><BrandMark /></span>
          <span>Grocery Router</span>
        </Link>
        <div className="workspace-label">{session.data?.households[0]?.householdName ?? 'Household'}</div>
        <nav className="navigation" aria-label="Primary navigation">
          <Link to="/" activeOptions={{ exact: true }} activeProps={{ className: 'active' }}>
            <CalendarIcon />
            <span>Week</span>
            <span className="navigation-count">{week.data?.recipes.length ?? 0}</span>
          </Link>
          <Link to="/groceries" activeProps={{ className: 'active' }}>
            <BagIcon />
            <span>Groceries</span>
          </Link>
          <Link to="/recipes" activeOptions={{ exact: true }} activeProps={{ className: 'active' }}>
            <RecipeIcon />
            <span>Recipes</span>
          </Link>
          <Link to="/history" activeOptions={{ exact: false }} activeProps={{ className: 'active' }}>
            <HistoryIcon />
            <span>History</span>
          </Link>
        </nav>
        <div className="sidebar-footer">
          <span>{session.data?.user.displayName}</span>
          <button type="button" className="sign-out" onClick={() => void signOut()}>Sign out</button>
        </div>
      </aside>
      <main className="main-canvas">
        <header className="topbar">
          <span>Home</span><span className="crumb-separator">/</span><strong>{crumb}</strong>
        </header>
        <div className="content"><Outlet /></div>
      </main>
    </div>
  )
}
