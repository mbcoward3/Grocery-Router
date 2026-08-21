import { useQuery } from '@tanstack/react-query'
import { Link, Outlet, useRouterState } from '@tanstack/react-router'
import { currentWeekQueryOptions } from '../features/week/queries'
import { BagIcon, CalendarIcon } from './icons'

export function AppShell() {
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  const week = useQuery(currentWeekQueryOptions)
  const crumb = pathname === '/' ? 'Week' : pathname === '/groceries' ? 'Groceries' : 'Recipe detail'

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <Link className="brand" to="/" aria-label="Grocery Router home">
          <span className="brand-mark" aria-hidden="true"><span /><span /><span /></span>
          <span>Grocery Router</span>
        </Link>
        <div className="workspace-label">Household</div>
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
        </nav>
        <div className="sidebar-footer"><span>Local only</span><span>v1</span></div>
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
