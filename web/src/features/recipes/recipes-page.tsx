import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useSearch } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { addRecipe, type RecipeSummary, type Week } from '../../api/client'
import { ArrowIcon, PlusIcon } from '../../ui/icons'
import { currentWeekQueryOptions, recipesQueryOptions, weekQueryKey } from '../week/queries'

export function RecipesPage() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const search = useSearch({ strict: false }) as { q?: string }
  const recipesQuery = useQuery(recipesQueryOptions)
  const weekQuery = useQuery(currentWeekQueryOptions)
  const [addedRecipe, setAddedRecipe] = useState<string | null>(null)
  const query = search.q ?? ''

  const recipes = useMemo(
    () => filterRecipesByName(recipesQuery.data?.recipes ?? [], query),
    [query, recipesQuery.data],
  )

  const addMutation = useMutation({
    mutationKey: ['week', 'add-from-recipes'],
    mutationFn: ({ recipeId }: { recipeId: number; recipeName: string }) => addRecipe(recipeId),
    onSuccess: (week: Week, variables) => {
      queryClient.setQueryData(weekQueryKey, week)
      void queryClient.invalidateQueries({ queryKey: ['week', 'groceries'] })
      setAddedRecipe(variables.recipeName)
    },
  })

  if (recipesQuery.isLoading || weekQuery.isLoading) {
    return <div className="recipe-browser-skeleton" aria-label="Loading recipes" aria-busy="true" />
  }

  if (recipesQuery.error || weekQuery.error) {
    const error = recipesQuery.error ?? weekQuery.error
    return (
      <div className="error-panel" role="alert">
        <div><strong>Couldn’t load recipes.</strong><p>{error?.message}</p></div>
        <button className="button" type="button" onClick={() => void queryClient.invalidateQueries()}>Try again</button>
      </div>
    )
  }

  const total = recipesQuery.data?.recipes.length ?? 0
  const hasWeek = weekQuery.data !== null

  return (
    <section aria-labelledby="recipes-heading" aria-busy={addMutation.isPending}>
      <header className="page-heading recipe-browser-heading">
        <div>
          <div className="eyebrow">Verified corpus</div>
          <h1 id="recipes-heading">Recipes</h1>
          <p>Browse the household collection or find a recipe by name.</p>
        </div>
        <span className="corpus-count">{total} {total === 1 ? 'recipe' : 'recipes'}</span>
      </header>

      <label className="recipe-search">
        <span className="sr-only">Search recipes by name</span>
        <span aria-hidden="true">⌕</span>
        <input
          type="search"
          value={query}
          placeholder="Search recipes by name…"
          autoComplete="off"
          onChange={(event) => {
            setAddedRecipe(null)
            void navigate({ to: '/recipes', search: { q: event.target.value || undefined }, replace: true })
          }}
        />
        {query && <span className="search-result-count">{recipes.length} found</span>}
      </label>

      {!hasWeek && (
        <p className="browser-week-note">
          <Link to="/">Generate a week</Link> before adding recipes to it.
        </p>
      )}
      {addedRecipe && <p className="browser-notice" role="status">Added {addedRecipe} to this week.</p>}
      {addMutation.error && <p className="inline-error" role="alert">{addMutation.error.message}</p>}

      {recipes.length > 0 ? (
        <div className="recipe-browser-list">
          {recipes.map((recipe) => (
            <RecipeBrowserRow
              key={recipe.id}
              recipe={recipe}
              query={query}
              hasWeek={hasWeek}
              pending={addMutation.isPending}
              onAdd={() => {
                setAddedRecipe(null)
                addMutation.mutate({ recipeId: recipe.id, recipeName: recipe.name })
              }}
            />
          ))}
        </div>
      ) : (
        <div className="recipe-browser-empty">
          <strong>No recipes found</strong>
          <span>Try a different name.</span>
        </div>
      )}
    </section>
  )
}

function RecipeBrowserRow({
  recipe,
  query,
  hasWeek,
  pending,
  onAdd,
}: {
  recipe: RecipeSummary
  query: string
  hasWeek: boolean
  pending: boolean
  onAdd: () => void
}) {
  return (
    <article className="recipe-browser-row">
      <Link
        className="recipe-browser-main"
        to="/recipes/$recipeId"
        params={{ recipeId: String(recipe.id) }}
        search={{ from: 'recipes', q: query || undefined }}
      >
        <div>
          <h2>{recipe.name}</h2>
          <div className="recipe-browser-meta">
            <span>{duration('Hands-on', recipe.handsOn)}</span>
            <span>{duration('Unattended', recipe.unattended)}</span>
            {recipe.yield && <span>{recipe.yield}</span>}
          </div>
        </div>
        <ArrowIcon />
      </Link>
      <button
        className="button recipe-browser-add"
        type="button"
        disabled={!hasWeek || pending}
        title={hasWeek ? `Add ${recipe.name} to this week` : 'Generate a week before adding recipes'}
        onClick={onAdd}
      >
        <PlusIcon /> <span>Add to week</span>
      </button>
    </article>
  )
}

function duration(label: string, value: RecipeSummary['handsOn']): string {
  const { minimumMinutes: min, maximumMinutes: max } = value
  if (min === null && max === null) return `${label} unknown`
  if (min === max || max === null) return `${label} ${formatMinutes(min!)}`
  if (min === null) return `${label} up to ${formatMinutes(max)}`
  return `${label} ${formatMinutes(min)}–${formatMinutes(max)}`
}

function formatMinutes(minutes: number): string {
  if (minutes < 60) return `${minutes} min`
  const hours = Math.floor(minutes / 60)
  const remainder = minutes % 60
  return remainder === 0 ? `${hours} hr` : `${hours} hr ${remainder} min`
}

export function filterRecipesByName(recipes: RecipeSummary[], query: string): RecipeSummary[] {
  const normalizedQuery = query.trim().toLocaleLowerCase()
  if (!normalizedQuery) return recipes
  return recipes.filter((recipe) => recipe.name.toLocaleLowerCase().includes(normalizedQuery))
}
