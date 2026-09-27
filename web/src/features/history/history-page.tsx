import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from '@tanstack/react-router'
import { getHistoricalWeek, listWeekHistory, type GroceryLine } from '../../api/client'
import { ArrowIcon, CheckIcon } from '../../ui/icons'

const historyKey = ['weeks', 'history'] as const

export function HistoryPage() {
  const query = useQuery({ queryKey: historyKey, queryFn: listWeekHistory })

  if (query.isLoading) return <HistorySkeleton />
  if (query.error) return <ErrorPanel message={query.error.message} onRetry={() => void query.refetch()} />

  const weeks = query.data?.weeks ?? []
  return (
    <section aria-labelledby="history-heading">
      <header className="page-heading">
        <div>
          <div className="eyebrow">Past plans</div>
          <h1 id="history-heading">History</h1>
          <p>Your final meal pools and shopping progress, week by week.</p>
        </div>
        <span className="page-count">{weeks.length} weeks</span>
      </header>

      {weeks.length === 0 ? (
        <div className="empty-week history-empty">
          <div className="empty-mark" aria-hidden="true">↺</div>
          <h2>No past weeks yet</h2>
          <p>Once a new week begins, your previous plan and grocery checklist will appear here.</p>
          <Link className="button primary" to="/">Go to this week</Link>
        </div>
      ) : (
        <div className="history-list">
          {weeks.map((week) => (
            <Link className="history-row" key={week.id} to="/history/$weekId" params={{ weekId: String(week.id) }}>
              <div className="history-date">
                <strong>{formatWeekRange(week.startsOn)}</strong>
                <span>Week of {formatShortDate(week.startsOn)}</span>
              </div>
              <div className="history-stat"><strong>{week.recipeCount}</strong><span>meals</span></div>
              <div className="history-stat history-progress">
                <strong>{week.completedCount}<span> / {week.groceryCount}</span></strong>
                <span>groceries checked</span>
              </div>
              <span className="history-arrow" aria-hidden="true"><ArrowIcon /></span>
            </Link>
          ))}
        </div>
      )}
    </section>
  )
}

export function HistoricalWeekPage() {
  const { weekId } = useParams({ from: '/history/$weekId' })
  const id = Number(weekId)
  const query = useQuery({
    queryKey: ['weeks', 'history', id],
    queryFn: () => getHistoricalWeek(id),
    enabled: Number.isInteger(id) && id > 0,
  })

  if (query.isLoading) return <HistorySkeleton />
  if (query.error) return <ErrorPanel message={query.error.message} onRetry={() => void query.refetch()} />
  if (!query.data) return null

  const { week, groceries } = query.data
  const sections = groupLines(groceries.lines)
  const activeLines = groceries.lines.filter((line) => !line.removed)
  const completed = activeLines.filter((line) => line.completed).length

  return (
    <section aria-labelledby="historical-week-heading">
      <Link className="recipe-back" to="/history">← Back to history</Link>
      <header className="history-detail-heading">
        <div>
          <div className="eyebrow">Past week · Read only</div>
          <h1 id="historical-week-heading">{formatWeekRange(week.startsOn)}</h1>
          <p>{week.recipes.length} {week.recipes.length === 1 ? 'meal' : 'meals'} · {completed} of {activeLines.length} groceries checked</p>
        </div>
      </header>

      <section className="history-detail-section" aria-labelledby="historical-meals">
        <header><h2 id="historical-meals">Meals</h2><span>{week.recipes.length}</span></header>
        <div className="history-meals">
          {week.recipes.map((occurrence) => (
            <Link key={occurrence.id} to="/recipes/$recipeId" params={{ recipeId: String(occurrence.recipe.id) }}>
              <span>{occurrence.recipe.name}</span><ArrowIcon />
            </Link>
          ))}
        </div>
      </section>

      <section className="history-detail-section" aria-labelledby="historical-groceries">
        <header><h2 id="historical-groceries">Grocery checklist</h2><span>{completed} / {activeLines.length} checked</span></header>
        <div className="history-grocery-sections">
          {sections.map(([section, lines]) => (
            <section className="history-grocery-section" key={section}>
              <h3>{section}</h3>
              <div className="history-grocery-lines">
                {lines.map((line) => <HistoricalGroceryRow key={line.id} line={line} />)}
              </div>
            </section>
          ))}
        </div>
      </section>
    </section>
  )
}

function HistoricalGroceryRow({ line }: { line: GroceryLine }) {
  const state = line.removed ? 'Removed' : line.completed ? 'Checked' : 'Not checked'
  return (
    <div className={`historical-grocery-row${line.completed ? ' completed' : ''}${line.removed ? ' removed' : ''}`}>
      <span className="historical-check" aria-hidden="true">{line.completed && !line.removed && <CheckIcon />}</span>
      <div><strong>{line.name}</strong>{line.optional && <small>Optional</small>}</div>
      <span className="historical-quantity">{line.quantity ?? ''}{line.quantity !== line.generatedQuantity && <small>Adjusted</small>}</span>
      <span className="historical-state">{state}</span>
    </div>
  )
}

function groupLines(lines: GroceryLine[]): [string, GroceryLine[]][] {
  const groups = new Map<string, GroceryLine[]>()
  for (const line of lines) groups.set(line.section, [...(groups.get(line.section) ?? []), line])
  return [...groups.entries()]
}

function HistorySkeleton() {
  return <section aria-label="Loading history" aria-busy="true"><header className="page-heading"><div><div className="eyebrow">Past plans</div><h1>History</h1></div></header><div className="recipe-list">{Array.from({ length: 4 }, (_, index) => <div className="skeleton-row" key={index} />)}</div></section>
}

function ErrorPanel({ message, onRetry }: { message: string; onRetry: () => void }) {
  return <div className="error-panel" role="alert"><div><strong>Couldn’t load history.</strong><p>{message}</p></div><button className="button" onClick={onRetry}>Try again</button></div>
}

function formatShortDate(value: string) {
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(`${value}T00:00:00Z`))
}

function formatWeekRange(value: string) {
  const start = new Date(`${value}T00:00:00Z`)
  const end = new Date(start)
  end.setUTCDate(end.getUTCDate() + 6)
  const startText = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(start)
  const endText = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' }).format(end)
  return `${startText} – ${endText}`
}
