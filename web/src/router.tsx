import type { QueryClient } from '@tanstack/react-query'
import {
  createRootRouteWithContext,
  createRoute,
  createRouter,
  Link,
} from '@tanstack/react-router'
import { AppShell } from './ui/app-shell'
import { WeekPage } from './features/week/week-page'

interface RouterContext {
  queryClient: QueryClient
}

const rootRoute = createRootRouteWithContext<RouterContext>()({
  component: AppShell,
  notFoundComponent: () => (
    <StatePage eyebrow="Not found" title="That page does not exist.">
      <Link className="button" to="/">Return to this week</Link>
    </StatePage>
  ),
})

const weekRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: WeekPage,
})

const groceriesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/groceries',
  component: () => (
    <StatePage eyebrow="Groceries" title="Your consolidated checklist is next.">
      Week planning is ready. Grocery checklist wiring follows the same compact Atlas shell.
    </StatePage>
  ),
})

const recipeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/recipes/$recipeId',
  component: RecipePlaceholder,
})

function RecipePlaceholder() {
  return (
    <StatePage eyebrow="Recipe detail" title="Recipe details are being prepared.">
      <Link className="button" to="/">Return to this week</Link>
    </StatePage>
  )
}

function StatePage({
  eyebrow,
  title,
  children,
}: {
  eyebrow: string
  title: string
  children: React.ReactNode
}) {
  return (
    <section className="state-page">
      <div className="eyebrow">{eyebrow}</div>
      <h1>{title}</h1>
      <div className="state-page-copy">{children}</div>
    </section>
  )
}

const routeTree = rootRoute.addChildren([weekRoute, groceriesRoute, recipeRoute])

export const router = createRouter({ routeTree, context: { queryClient: undefined! } })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
