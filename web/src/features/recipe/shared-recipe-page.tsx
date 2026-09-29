import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams, useSearch } from '@tanstack/react-router'
import { getCatalogRecipe, setCatalogMembership } from '../../api/client'
import { ExternalIcon } from '../../ui/icons'

export function SharedRecipePage() {
  const { catalogRecipeId } = useParams({ strict: false }) as { catalogRecipeId: string }
  const search = useSearch({ strict: false }) as { q?: string }
  const queryClient = useQueryClient()
  const id = Number(catalogRecipeId)
  const query = useQuery({ queryKey: ['catalog', 'recipes', id], queryFn: () => getCatalogRecipe(id), enabled: Number.isInteger(id) && id > 0 })
  const membership = useMutation({ mutationFn: (state: 'trial' | 'adopted') => setCatalogMembership(id, state), onSuccess: () => { void queryClient.invalidateQueries({ queryKey: ['catalog', 'recipes'] }); void queryClient.invalidateQueries({ queryKey: ['recipes'] }); void query.refetch() } })
  if (query.isLoading) return <div className="recipe-detail-skeleton" aria-label="Loading shared recipe" aria-busy="true" />
  if (query.error || !query.data) return <div className="error-panel" role="alert"><div><strong>Couldn’t load this shared recipe.</strong><p>{query.error?.message ?? 'The recipe ID is invalid.'}</p></div><Link className="button" to="/recipes" search={{ scope: 'shared', q: search.q }}>Back to shared recipes</Link></div>
  const recipe = query.data
  const primary = recipe.sources.find((source) => source.primary) ?? recipe.sources[0]
  return <article className="recipe-detail">
    <Link className="recipe-back" to="/recipes" search={{ scope: 'shared', q: search.q }}><span aria-hidden="true">←</span> Shared recipes</Link>
    <header className="recipe-detail-header">
      <div className="eyebrow">Shared catalog</div>
      <div className="shared-detail-title"><h1>{recipe.name}</h1><span className={`catalog-state ${recipe.status === 'reviewable' ? 'review' : recipe.householdState ?? 'shared'}`}>{recipe.status === 'reviewable' ? 'Needs review' : recipe.householdState ?? 'Shared'}</span></div>
      <div className="semantic-facets">{recipe.facets.map((facet) => <span className="pill" key={facet}>{facet}</span>)}</div>
      {recipe.status === 'reviewable' ? <p className="review-callout">This generated candidate is available for inspection but cannot enter family planning until a human approves it.</p>
        : <div className="shared-detail-actions">{recipe.householdState === null && <button className="button primary" disabled={membership.isPending} onClick={() => membership.mutate('trial')}>Try with family</button>}{recipe.householdState === 'trial' && <button className="button primary" disabled={membership.isPending} onClick={() => membership.mutate('adopted')}>Adopt recipe</button>}</div>}
      {membership.error && <p className="inline-error" role="alert">{membership.error.message}</p>}
      {primary && <div className="recipe-source"><span>Source: {primary.attribution}</span>{primary.url && <a href={primary.url} target="_blank" rel="noreferrer">View source <ExternalIcon /></a>}</div>}
    </header>
    <div className="recipe-detail-grid">
      <aside className="ingredients-panel"><div className="section-title"><span>Ingredients</span><span>{recipe.ingredientSections.reduce((sum, section) => sum + section.ingredients.length, 0)}</span></div>{recipe.ingredientSections.map((section) => <section className="ingredient-section" key={section.name}><h2>{section.name}</h2><ul>{section.ingredients.map((ingredient) => <li key={ingredient.id}><span>{ingredient.sourceText}</span>{ingredient.optional && <span className="optional-label">Optional</span>}{ingredient.displayNote && <small>{ingredient.displayNote}</small>}</li>)}</ul></section>)}</aside>
      <main className="instructions-panel"><div className="section-title"><span>Instructions</span></div>{recipe.instructionSections.map((section) => <section className="instruction-section" key={section.name}>{recipe.instructionSections.length > 1 && <h2>{section.name}</h2>}<ol>{section.steps.map((step) => <li key={step.id}><span>{step.instruction}</span></li>)}</ol></section>)}</main>
    </div>
  </article>
}
