import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import {
  addRecipe,
  generateWeek,
  randomSwapRecipe,
  removeRecipe,
  swapRecipe,
  type RecipeSummary,
  type Week,
} from '../../api/client'
import { ArrowIcon, CloseIcon, PlusIcon, SwapIcon, TrashIcon } from '../../ui/icons'
import { currentWeekQueryOptions, recipesQueryOptions, weekQueryKey } from './queries'

type WeekAction =
  | { type: 'generate'; recipeCount: number }
  | { type: 'add'; recipeId: number }
  | { type: 'remove'; occurrenceId: number }
  | { type: 'random-swap'; occurrenceId: number }
  | { type: 'swap'; occurrenceId: number; recipeId: number }

type PickerState = { mode: 'add' } | { mode: 'swap'; occurrenceId: number }

export function WeekPage() {
  const queryClient = useQueryClient()
  const weekQuery = useQuery(currentWeekQueryOptions)
  const recipesQuery = useQuery(recipesQueryOptions)
  const [recipeCount, setRecipeCount] = useState(5)
  const [picker, setPicker] = useState<PickerState | null>(null)

  const mutation = useMutation({
    mutationKey: ['week'],
    mutationFn: runWeekAction,
    onSuccess: (week) => {
      queryClient.setQueryData(weekQueryKey, week)
      setPicker(null)
    },
  })

  const week = weekQuery.data
  const recipes = recipesQuery.data?.recipes ?? []
  const maxCount = Math.max(1, recipes.length)
  const selectedCount = Math.min(recipeCount, maxCount)
  const isLoading = weekQuery.isLoading || recipesQuery.isLoading
  const readError = weekQuery.error ?? recipesQuery.error

  if (isLoading) return <WeekSkeleton />
  if (readError) {
    return <ErrorPanel message={readError.message} onRetry={() => void queryClient.invalidateQueries()} />
  }

  if (!week) {
    return (
      <section aria-labelledby="week-heading">
        <PageHeading count={0} />
        <div className="empty-week">
          <div className="empty-mark" aria-hidden="true">✦</div>
          <h2 id="week-heading">Build this week’s recipe pool</h2>
          <p>Choose a starting size. You can add, remove, or swap any recipe afterward.</p>
          <label className="count-field">
            Recipes
            <select
              value={selectedCount}
              onChange={(event) => setRecipeCount(Number(event.target.value))}
              disabled={mutation.isPending}
            >
              {Array.from({ length: maxCount }, (_, index) => index + 1).map((count) => (
                <option key={count} value={count}>{count}</option>
              ))}
            </select>
          </label>
          <button
            className="button primary"
            type="button"
            disabled={mutation.isPending || recipes.length === 0}
            onClick={() => mutation.mutate({ type: 'generate', recipeCount: selectedCount })}
          >
            {mutation.isPending ? 'Generating…' : 'Generate week'}
          </button>
          <MutationError error={mutation.error} />
        </div>
      </section>
    )
  }

  return (
    <section aria-labelledby="week-heading" aria-busy={mutation.isPending}>
      <PageHeading count={week.recipes.length}>
        <button className="button" type="button" onClick={() => mutation.mutate({ type: 'generate', recipeCount: week.recipes.length })} disabled={mutation.isPending || week.recipes.length === 0}>
          <SwapIcon /> Refresh pool
        </button>
        <button className="button primary" type="button" onClick={() => setPicker({ mode: 'add' })} disabled={mutation.isPending}>
          <PlusIcon /> Add recipe
        </button>
      </PageHeading>

      <div className="recipe-list">
        {week.recipes.map((occurrence) => (
          <article className="recipe-row" key={occurrence.id}>
            <div className="recipe-grip" aria-hidden="true">⠿</div>
            <div className="recipe-main">
              <h2>{occurrence.recipe.name}</h2>
              <div className="recipe-meta">
                {durationLabel(occurrence.recipe) && <span>{durationLabel(occurrence.recipe)}</span>}
                {occurrence.recipe.yield && <span>{occurrence.recipe.yield}</span>}
              </div>
            </div>
            <div className="recipe-pills">
              {durationLabel(occurrence.recipe) && <span className="pill">{durationLabel(occurrence.recipe)}</span>}
              {occurrence.recipe.yield && <span className="pill">{occurrence.recipe.yield}</span>}
            </div>
            <div className="recipe-actions">
              <Link className="icon-button detail-action" to="/recipes/$recipeId" params={{ recipeId: String(occurrence.recipe.id) }} aria-label={`Open ${occurrence.recipe.name}`}>
                <ArrowIcon />
              </Link>
              <button className="icon-button" type="button" onClick={() => mutation.mutate({ type: 'random-swap', occurrenceId: occurrence.id })} disabled={mutation.isPending} aria-label={`Randomly swap ${occurrence.recipe.name}`} title="Random swap">
                <SwapIcon />
              </button>
              <button className="icon-button specific-swap" type="button" onClick={() => setPicker({ mode: 'swap', occurrenceId: occurrence.id })} disabled={mutation.isPending} aria-label={`Choose a replacement for ${occurrence.recipe.name}`} title="Choose replacement">
                <span aria-hidden="true">•••</span>
              </button>
              <button className="icon-button danger" type="button" onClick={() => mutation.mutate({ type: 'remove', occurrenceId: occurrence.id })} disabled={mutation.isPending} aria-label={`Remove ${occurrence.recipe.name}`} title="Remove">
                <TrashIcon />
              </button>
            </div>
          </article>
        ))}
      </div>

      <div className="week-footer">
        <span>Week of {formatWeekDate(week.startsOn)}</span>
        <span>{week.recipes.length} {week.recipes.length === 1 ? 'recipe' : 'recipes'}</span>
      </div>
      <MutationError error={mutation.error} />

      {picker && (
        <RecipePicker
          key={`${picker.mode}-${picker.mode === 'swap' ? picker.occurrenceId : 'new'}`}
          picker={picker}
          recipes={recipes}
          pending={mutation.isPending}
          onClose={() => setPicker(null)}
          onChoose={(recipeId) => {
            if (picker.mode === 'add') mutation.mutate({ type: 'add', recipeId })
            else mutation.mutate({ type: 'swap', occurrenceId: picker.occurrenceId, recipeId })
          }}
        />
      )}
    </section>
  )
}

function PageHeading({ count, children }: { count: number; children?: React.ReactNode }) {
  return (
    <header className="page-heading">
      <div>
        <div className="eyebrow">Current pool</div>
        <h1 id="week-heading">Recipes for this week</h1>
        <p>A flexible set of meals, ready when you are.</p>
      </div>
      <div className="page-actions">{children}</div>
      <span className="page-count">{count} {count === 1 ? 'recipe' : 'recipes'}</span>
    </header>
  )
}

function RecipePicker({
  picker,
  recipes,
  pending,
  onClose,
  onChoose,
}: {
  picker: PickerState
  recipes: RecipeSummary[]
  pending: boolean
  onClose: () => void
  onChoose: (recipeId: number) => void
}) {
  const sortedRecipes = useMemo(() => [...recipes].sort((a, b) => a.name.localeCompare(b.name)), [recipes])
  const [recipeId, setRecipeId] = useState(sortedRecipes[0]?.id ?? 0)
  const title = picker.mode === 'add' ? 'Add a recipe' : 'Choose a replacement'

  return (
    <div className="dialog-layer" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <dialog className="picker-dialog" open aria-labelledby="picker-title">
        <header>
          <div><div className="eyebrow">Recipe pool</div><h2 id="picker-title">{title}</h2></div>
          <button className="icon-button" type="button" onClick={onClose} aria-label="Close"><CloseIcon /></button>
        </header>
        <label>
          Recipe
          <select value={recipeId} onChange={(event) => setRecipeId(Number(event.target.value))} autoFocus>
            {sortedRecipes.map((recipe) => <option key={recipe.id} value={recipe.id}>{recipe.name}</option>)}
          </select>
        </label>
        <footer>
          <button className="button" type="button" onClick={onClose}>Cancel</button>
          <button className="button primary" type="button" disabled={pending || recipeId === 0} onClick={() => onChoose(recipeId)}>
            {pending ? 'Saving…' : picker.mode === 'add' ? 'Add recipe' : 'Replace recipe'}
          </button>
        </footer>
      </dialog>
    </div>
  )
}

function WeekSkeleton() {
  return (
    <section aria-label="Loading this week" aria-busy="true">
      <PageHeading count={0} />
      <div className="recipe-list skeleton-list">
        {Array.from({ length: 4 }, (_, index) => <div className="skeleton-row" key={index} />)}
      </div>
    </section>
  )
}

function ErrorPanel({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="error-panel" role="alert">
      <div><strong>Couldn’t load this week.</strong><p>{message}</p></div>
      <button className="button" type="button" onClick={onRetry}>Try again</button>
    </div>
  )
}

function MutationError({ error }: { error: Error | null }) {
  if (!error) return null
  return <p className="inline-error" role="alert">{error.message}</p>
}

function durationLabel(recipe: RecipeSummary): string | null {
  const minimum = recipe.handsOn.minimumMinutes
  const maximum = recipe.handsOn.maximumMinutes
  if (minimum === null && maximum === null) return null
  if (minimum === maximum || maximum === null) return `${minimum} min hands-on`
  if (minimum === null) return `Up to ${maximum} min hands-on`
  return `${minimum}–${maximum} min hands-on`
}

function formatWeekDate(date: string): string {
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(`${date}T00:00:00Z`))
}

function runWeekAction(action: WeekAction): Promise<Week> {
  switch (action.type) {
    case 'generate': return generateWeek(action.recipeCount)
    case 'add': return addRecipe(action.recipeId)
    case 'remove': return removeRecipe(action.occurrenceId)
    case 'random-swap': return randomSwapRecipe(action.occurrenceId)
    case 'swap': return swapRecipe(action.occurrenceId, action.recipeId)
  }
}
