import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams, useSearch } from '@tanstack/react-router'
import { getCatalogRecipe, setCatalogMembership, type CatalogRecipeSummary } from '../../api/client'
import { ExternalIcon } from '../../ui/icons'

export function SharedRecipePage() {
  const { catalogRecipeId } = useParams({ strict: false }) as { catalogRecipeId: string }
  const search = useSearch({ strict: false }) as { q?: string }
  const queryClient = useQueryClient()
  const id = Number(catalogRecipeId)
  const query = useQuery({ queryKey: ['catalog', 'recipes', id], queryFn: () => getCatalogRecipe(id), enabled: Number.isInteger(id) && id > 0 })
  const membership = useMutation({ mutationFn: (state: 'trial' | 'adopted') => setCatalogMembership(id, state), onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ['catalog', 'recipes'] }); void queryClient.invalidateQueries({ queryKey: ['recipes'] }); void query.refetch() } })
  if (query.isLoading) return <div className="recipe-detail-skeleton" aria-label="Loading recipe" aria-busy="true" />
  if (query.error || !query.data) return <div className="error-panel" role="alert"><div><strong>Couldn’t load this recipe.</strong><p>{query.error?.message ?? 'The recipe ID is invalid.'}</p></div><Link className="button" to="/recipes" search={{ scope: 'shared', q: search.q }}>Back to Explore</Link></div>
  const recipe = query.data
  const primary = recipe.sources.find((source) => source.primary) ?? recipe.sources[0]
  return <article className="recipe-detail">
    <Link className="recipe-back" to="/recipes" search={{ scope: 'shared', q: search.q }}><span aria-hidden="true">←</span> Explore</Link>
    <header className="recipe-detail-header">
      <h1>{recipe.name}</h1>
      <div className="recipe-facts">
        <Fact label="Hands-on" value={duration(recipe.handsOn)} />
        <Fact label="Unattended" value={duration(recipe.unattended)} />
        <Fact label="Yield" value={recipe.yield ?? 'Unknown'} />
      </div>
      {recipe.status !== 'reviewable' && <div className="shared-detail-actions">{recipe.householdState === null && <button className="button primary" disabled={membership.isPending} onClick={() => membership.mutate('trial')}>Try with family</button>}{recipe.householdState === 'trial' && <button className="button primary" disabled={membership.isPending} onClick={() => membership.mutate('adopted')}>Adopt recipe</button>}</div>}
      {membership.error && <p className="inline-error" role="alert">{membership.error.message}</p>}
      {primary && <div className="recipe-source"><span>Source</span>{primary.url ? <a href={primary.url} target="_blank" rel="noreferrer">{primary.attribution} <ExternalIcon /></a> : <span>{primary.attribution}</span>}</div>}
    </header>
    <div className="recipe-detail-grid">
      <aside className="ingredients-panel"><div className="section-title"><span>Ingredients</span><span>{recipe.ingredientSections.reduce((sum, section) => sum + section.ingredients.length, 0)}</span></div>{recipe.ingredientSections.map((section) => <section className="ingredient-section" key={section.name}><h2>{section.name}</h2><ul>{section.ingredients.map((ingredient) => <li key={ingredient.id}><span>{ingredient.sourceText}</span>{ingredient.optional && <span className="optional-label">Optional</span>}{ingredient.displayNote && <small>{ingredient.displayNote}</small>}</li>)}</ul></section>)}</aside>
      <main className="instructions-panel"><div className="section-title"><span>Instructions</span></div>{recipe.instructionSections.map((section) => <section className="instruction-section" key={section.name}>{recipe.instructionSections.length > 1 && <h2>{section.name}</h2>}<ol>{section.steps.map((step) => <li key={step.id}><span>{step.instruction}</span></li>)}</ol></section>)}</main>
    </div>
  </article>
}

function Fact({ label, value }: { label: string; value: string }) {
  return <div className="recipe-fact"><span>{label}</span><strong>{value}</strong></div>
}

function duration(value: CatalogRecipeSummary['handsOn']): string {
  const { minimumMinutes: min, maximumMinutes: max } = value
  if (min === null && max === null) return 'Unknown'
  if (min === max || max === null) return `${min} min`
  if (min === null) return `Up to ${max} min`
  return `${min}–${max} min`
}
