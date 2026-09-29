import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useSearch } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import {
  addRecipe,
  listCatalogRecipes,
  setCatalogMembership,
  type CatalogRecipeSummary,
  type RecipeSummary,
  type Week,
} from '../../api/client'
import { ArrowIcon, PlusIcon } from '../../ui/icons'
import { currentWeekQueryOptions, recipesQueryOptions, weekQueryKey } from '../week/queries'

type Scope = 'family' | 'shared'

export function RecipesPage() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const search = useSearch({ strict: false }) as { q?: string; scope?: Scope }
  const scope: Scope = search.scope === 'shared' ? 'shared' : 'family'
  const query = search.q ?? ''
  const recipesQuery = useQuery(recipesQueryOptions)
  const catalogQuery = useQuery({ queryKey: ['catalog', 'recipes'], queryFn: () => listCatalogRecipes() })
  const weekQuery = useQuery(currentWeekQueryOptions)
  const [notice, setNotice] = useState<string | null>(null)

  const family = useMemo(() => filterRecipesByName(recipesQuery.data?.recipes ?? [], query), [query, recipesQuery.data])
  const shared = useMemo(() => filterCatalogRecipes(catalogQuery.data?.recipes ?? [], query), [query, catalogQuery.data])

  const addMutation = useMutation({
    mutationKey: ['week', 'add-from-recipes'],
    mutationFn: ({ recipeId }: { recipeId: number; recipeName: string }) => addRecipe(recipeId),
    onSuccess: (week: Week, variables) => {
      queryClient.setQueryData(weekQueryKey, week)
      void queryClient.invalidateQueries({ queryKey: ['week', 'groceries'] })
      setNotice(`Added ${variables.recipeName} to this week.`)
    },
  })
  const membershipMutation = useMutation({
    mutationKey: ['catalog', 'membership'],
    mutationFn: ({ catalogId, state }: { catalogId: number; name: string; state: 'trial' | 'adopted' }) => setCatalogMembership(catalogId, state),
    onSuccess: (_result, variables) => {
      void queryClient.invalidateQueries({ queryKey: ['catalog', 'recipes'] })
      void queryClient.invalidateQueries({ queryKey: ['recipes'] })
      setNotice(variables.state === 'trial' ? `Added ${variables.name} as a family trial.` : `Adopted ${variables.name}.`)
    },
  })

  if (recipesQuery.isLoading || catalogQuery.isLoading || weekQuery.isLoading) {
    return <div className="recipe-browser-skeleton" aria-label="Loading recipes" aria-busy="true" />
  }
  const error = recipesQuery.error ?? catalogQuery.error ?? weekQuery.error
  if (error) {
    return <div className="error-panel" role="alert"><div><strong>Couldn’t load recipes.</strong><p>{error.message}</p></div><button className="button" type="button" onClick={() => void queryClient.invalidateQueries()}>Try again</button></div>
  }

  const total = scope === 'family' ? family.length : shared.length
  const pending = addMutation.isPending || membershipMutation.isPending
  return (
    <section aria-labelledby="recipes-heading" aria-busy={pending}>
      <header className="page-heading recipe-browser-heading">
        <div><div className="eyebrow">Recipe library</div><h1 id="recipes-heading">Recipes</h1><p>Keep family favorites close and explore the shared catalog.</p></div>
        <span className="corpus-count">{total} {total === 1 ? 'recipe' : 'recipes'}</span>
      </header>

      <div className="recipe-scope-tabs" role="tablist" aria-label="Recipe collection">
        <button role="tab" aria-selected={scope === 'family'} onClick={() => void navigate({ to: '/recipes', search: { scope: 'family', q: query || undefined }, replace: true })}>Family <span>{recipesQuery.data?.recipes.length ?? 0}</span></button>
        <button role="tab" aria-selected={scope === 'shared'} onClick={() => void navigate({ to: '/recipes', search: { scope: 'shared', q: query || undefined }, replace: true })}>Shared <span>{catalogQuery.data?.recipes.length ?? 0}</span></button>
      </div>

      <label className="recipe-search">
        <span className="sr-only">Search recipes</span><span aria-hidden="true">⌕</span>
        <input type="search" value={query} placeholder={scope === 'family' ? 'Search family recipes…' : 'Search names or smart facets…'} autoComplete="off" onChange={(event) => { setNotice(null); void navigate({ to: '/recipes', search: { scope, q: event.target.value || undefined }, replace: true }) }} />
        {query && <span className="search-result-count">{total} found</span>}
      </label>

      {notice && <p className="browser-notice" role="status">{notice}</p>}
      {(addMutation.error || membershipMutation.error) && <p className="inline-error" role="alert">{(addMutation.error ?? membershipMutation.error)?.message}</p>}

      {scope === 'family' ? (
        family.length ? <div className="recipe-browser-list">{family.map((recipe) => <FamilyRow key={recipe.id} recipe={recipe} query={query} hasWeek={weekQuery.data !== null} pending={pending} onAdd={() => addMutation.mutate({ recipeId: recipe.id, recipeName: recipe.name })} />)}</div> : <Empty />
      ) : (
        shared.length ? <div className="recipe-browser-list">{shared.map((recipe) => <SharedRow key={recipe.catalogId} recipe={recipe} query={query} hasWeek={weekQuery.data !== null} pending={pending} onMembership={(state) => membershipMutation.mutate({ catalogId: recipe.catalogId, name: recipe.name, state })} onAdd={() => recipe.materializedRecipeId && addMutation.mutate({ recipeId: recipe.materializedRecipeId, recipeName: recipe.name })} />)}</div> : <Empty />
      )}
    </section>
  )
}

function FamilyRow({ recipe, query, hasWeek, pending, onAdd }: { recipe: RecipeSummary; query: string; hasWeek: boolean; pending: boolean; onAdd: () => void }) {
  return <article className="recipe-browser-row"><Link className="recipe-browser-main" to="/recipes/$recipeId" params={{ recipeId: String(recipe.id) }} search={{ from: 'recipes', q: query || undefined }}><div><div className="shared-name-line"><h2>{recipe.name}</h2>{recipe.collectionState && <span className={`catalog-state ${recipe.collectionState}`}>{recipe.collectionState}</span>}</div><div className="recipe-browser-meta"><span>{duration('Hands-on', recipe.handsOn)}</span><span>{duration('Unattended', recipe.unattended)}</span>{recipe.yield && <span>{recipe.yield}</span>}</div></div><ArrowIcon /></Link><button className="button recipe-browser-add" type="button" disabled={!hasWeek || pending} onClick={onAdd}><PlusIcon /><span>Add to week</span></button></article>
}

function SharedRow({ recipe, query, hasWeek, pending, onMembership, onAdd }: { recipe: CatalogRecipeSummary; query: string; hasWeek: boolean; pending: boolean; onMembership: (state: 'trial' | 'adopted') => void; onAdd: () => void }) {
  const reviewable = recipe.status === 'reviewable'
  return <article className="recipe-browser-row shared-recipe-row">
    <Link className="recipe-browser-main" to="/recipes/shared/$catalogRecipeId" params={{ catalogRecipeId: String(recipe.catalogId) }} search={{ q: query || undefined }}>
      <div><div className="shared-name-line"><h2>{recipe.name}</h2><span className={`catalog-state ${reviewable ? 'review' : recipe.householdState ?? 'shared'}`}>{reviewable ? 'Needs review' : recipe.householdState ?? 'Shared'}</span></div><div className="recipe-browser-meta">{recipe.facets.slice(0, 3).map((facet) => <span key={facet}>{facet}</span>)}{recipe.yield && <span>{recipe.yield}</span>}</div></div><ArrowIcon />
    </Link>
    {reviewable ? <button className="button recipe-browser-add" disabled title="Human approval is required">Reviewing</button>
      : recipe.householdState === null ? <button className="button recipe-browser-add" disabled={pending} onClick={() => onMembership('trial')}>Try with family</button>
      : recipe.householdState === 'trial' ? <button className="button recipe-browser-add" disabled={pending} onClick={() => onMembership('adopted')}>Adopt</button>
      : <button className="button recipe-browser-add" disabled={!hasWeek || pending} onClick={onAdd}><PlusIcon /><span>Add to week</span></button>}
  </article>
}

function Empty() { return <div className="recipe-browser-empty"><strong>No recipes found</strong><span>Try a different search.</span></div> }

function duration(label: string, value: RecipeSummary['handsOn']): string {
  const { minimumMinutes: min, maximumMinutes: max } = value
  if (min === null && max === null) return `${label} unknown`
  if (min === max || max === null) return `${label} ${formatMinutes(min!)}`
  if (min === null) return `${label} up to ${formatMinutes(max)}`
  return `${label} ${formatMinutes(min)}–${formatMinutes(max)}`
}
function formatMinutes(minutes: number): string { if (minutes < 60) return `${minutes} min`; const hours = Math.floor(minutes / 60); const remainder = minutes % 60; return remainder === 0 ? `${hours} hr` : `${hours} hr ${remainder} min` }
export function filterRecipesByName(recipes: RecipeSummary[], query: string): RecipeSummary[] { const q = query.trim().toLocaleLowerCase(); return q ? recipes.filter((recipe) => recipe.name.toLocaleLowerCase().includes(q)) : recipes }
export function filterCatalogRecipes(recipes: CatalogRecipeSummary[], query: string): CatalogRecipeSummary[] { const q = query.trim().toLocaleLowerCase(); return q ? recipes.filter((recipe) => recipe.name.toLocaleLowerCase().includes(q) || recipe.facets.some((facet) => facet.toLocaleLowerCase().includes(q))) : recipes }
