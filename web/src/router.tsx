import type { QueryClient } from '@tanstack/react-query'
import {
  createRootRouteWithContext,
  createRoute,
  createRouter,
  Link,
} from '@tanstack/react-router'
import { AppShell } from './ui/app-shell'
import { GroceriesPage } from './features/groceries/groceries-page'
import { RecipePage } from './features/recipe/recipe-page'
import { RecipesPage } from './features/recipes/recipes-page'
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
  component: GroceriesPage,
})

const recipesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/recipes',
  validateSearch: (search: Record<string, unknown>): { q?: string } => (
    typeof search.q === 'string' && search.q.length > 0 ? { q: search.q } : {}
  ),
  component: RecipesPage,
})

const recipeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/recipes/$recipeId',
  validateSearch: (search: Record<string, unknown>): { from?: 'recipes'; q?: string } => ({
    ...(search.from === 'recipes' ? { from: 'recipes' as const } : {}),
    ...(typeof search.q === 'string' && search.q.length > 0 ? { q: search.q } : {}),
  }),
  component: RecipePage,
})

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

const routeTree = rootRoute.addChildren([weekRoute, groceriesRoute, recipesRoute, recipeRoute])

export const router = createRouter({ routeTree, context: { queryClient: undefined! } })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
