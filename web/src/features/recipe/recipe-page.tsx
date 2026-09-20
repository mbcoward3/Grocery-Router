import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from '@tanstack/react-router'
import { getRecipe, type RecipeSummary } from '../../api/client'
import { ArrowIcon, ExternalIcon } from '../../ui/icons'

export function RecipePage() {
  const { recipeId } = useParams({ strict: false }) as { recipeId: string }
  const id = Number(recipeId)
  const query = useQuery({
    queryKey: ['recipes', id],
    queryFn: () => getRecipe(id),
    enabled: Number.isInteger(id) && id > 0,
  })

  if (query.isLoading) return <div className="recipe-detail-skeleton" aria-label="Loading recipe" aria-busy="true" />
  if (query.error || !query.data) {
    return (
      <div className="error-panel" role="alert">
        <div><strong>Couldn’t load this recipe.</strong><p>{query.error?.message ?? 'The recipe ID is invalid.'}</p></div>
        <Link className="button" to="/">Back to week</Link>
      </div>
    )
  }

  const recipe = query.data
  const primarySource = recipe.sources.find((source) => source.primary) ?? recipe.sources[0]
  return (
    <article className="recipe-detail">
      <Link className="recipe-back" to="/"><span aria-hidden="true">←</span> This week</Link>
      <header className="recipe-detail-header">
        <div className="eyebrow">Recipe</div>
        <h1>{recipe.name}</h1>
        <div className="recipe-facts">
          <Fact label="Hands-on" value={duration(recipe.handsOn)} />
          <Fact label="Unattended" value={duration(recipe.unattended)} />
          <Fact label="Yield" value={recipe.yield ?? 'Unknown'} />
        </div>
        {primarySource && (
          <div className="recipe-source">
            <span>{primarySource.relationship === 'adapted-from' ? 'Adapted from' : 'Source'}: {primarySource.attribution}</span>
            {primarySource.url && <a href={primarySource.url} target="_blank" rel="noreferrer">View source <ExternalIcon /></a>}
          </div>
        )}
      </header>

      <div className="recipe-detail-grid">
        <aside className="ingredients-panel">
          <div className="section-title"><span>Ingredients</span><span>{recipe.ingredientSections.reduce((sum, section) => sum + section.ingredients.length, 0)}</span></div>
          {recipe.ingredientSections.map((section) => (
            <section className="ingredient-section" key={section.name}>
              <h2>{section.name}</h2>
              <ul>
                {section.ingredients.map((ingredient) => (
                  <li key={ingredient.id}>
                    <span>{ingredient.sourceText}</span>
                    {ingredient.optional && <span className="optional-label">Optional</span>}
                    {ingredient.displayNote && <small>{ingredient.displayNote}</small>}
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </aside>

        <main className="instructions-panel">
          <div className="section-title"><span>Instructions</span></div>
          {recipe.instructionSections.map((section) => (
            <section className="instruction-section" key={section.name}>
              {recipe.instructionSections.length > 1 && <h2>{section.name}</h2>}
              <ol>
                {section.steps.map((step) => <li key={step.id}><span>{step.instruction}</span></li>)}
              </ol>
            </section>
          ))}
          <Link className="button recipe-end-link" to="/groceries">Open grocery list <ArrowIcon /></Link>
        </main>
      </div>
    </article>
  )
}

function Fact({ label, value }: { label: string; value: string }) {
  return <div className="recipe-fact"><span>{label}</span><strong>{value}</strong></div>
}

function duration(value: RecipeSummary['handsOn']): string {
  const { minimumMinutes: min, maximumMinutes: max } = value
  if (min === null && max === null) return 'Unknown'
  if (min === max || max === null) return `${min} min`
  if (min === null) return `Up to ${max} min`
  return `${min}–${max} min`
}
