import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { FormEvent, useMemo, useState } from 'react'
import {
  addGroceryLine,
  getGroceries,
  getGroceryContributions,
  removeGroceryLine,
  updateGroceryLine,
  type Groceries,
  type GroceryLine,
} from '../../api/client'
import { CheckIcon, ChevronIcon, CloseIcon, EditIcon, PlusIcon, TrashIcon } from '../../ui/icons'

const groceriesKey = ['week', 'groceries'] as const

type LineAction =
  | { type: 'complete'; line: GroceryLine }
  | { type: 'remove'; line: GroceryLine }
  | { type: 'override'; line: GroceryLine; value: string }

export function GroceriesPage() {
  const queryClient = useQueryClient()
  const query = useQuery({ queryKey: groceriesKey, queryFn: getGroceries })
  const [expanded, setExpanded] = useState<number | null>(null)
  const [newItem, setNewItem] = useState('')
  const [editing, setEditing] = useState<GroceryLine | null>(null)

  const mutation = useMutation({
    mutationKey: ['groceries'],
    mutationFn: runLineAction,
    onSuccess: (data) => {
      queryClient.setQueryData(groceriesKey, data)
      setEditing(null)
    },
  })
  const addMutation = useMutation({
    mutationKey: ['groceries', 'add'],
    mutationFn: addGroceryLine,
    onSuccess: (data) => {
      queryClient.setQueryData(groceriesKey, data)
      setNewItem('')
    },
  })

  const visibleLines = query.data?.lines.filter((line) => !line.removed) ?? []
  const sections = useMemo(() => groupLines(visibleLines), [visibleLines])
  const completed = visibleLines.filter((line) => line.completed).length

  if (query.isLoading) return <GroceriesSkeleton />
  if (query.error) return <ErrorPanel message={query.error.message} onRetry={() => void query.refetch()} />
  if (!query.data) {
    return (
      <section>
        <PageHeading count={0} completed={0} />
        <div className="empty-week">
          <div className="empty-mark" aria-hidden="true">✦</div>
          <h2>Generate a week first</h2>
          <p>Your grocery checklist is built from the recipes in the current pool.</p>
          <Link className="button primary" to="/">Go to week</Link>
        </div>
      </section>
    )
  }

  function addItem(event: FormEvent) {
    event.preventDefault()
    if (newItem.trim()) addMutation.mutate(newItem)
  }

  return (
    <section aria-labelledby="groceries-heading" aria-busy={mutation.isPending || addMutation.isPending}>
      <PageHeading count={visibleLines.length} completed={completed} />
      <form className="quick-add" onSubmit={addItem}>
        <PlusIcon />
        <label className="sr-only" htmlFor="new-grocery">Add grocery item</label>
        <input id="new-grocery" value={newItem} onChange={(event) => setNewItem(event.target.value)} placeholder="Add an item…" autoComplete="off" />
        <button className="button" type="submit" disabled={!newItem.trim() || addMutation.isPending}>{addMutation.isPending ? 'Adding…' : 'Add'}</button>
      </form>

      {sections.length === 0 ? (
        <div className="grocery-empty">No grocery items in this week.</div>
      ) : sections.map(([section, lines]) => (
        <section className="grocery-section" key={section} aria-labelledby={`section-${slug(section)}`}>
          <header><h2 id={`section-${slug(section)}`}>{section}</h2><span>{lines.length}</span></header>
          <div className="grocery-lines">
            {lines.map((line) => (
              <GroceryRow
                key={line.id}
                line={line}
                expanded={expanded === line.id}
                pending={mutation.isPending}
                onExpand={() => setExpanded(expanded === line.id ? null : line.id)}
                onComplete={() => mutation.mutate({ type: 'complete', line })}
                onEdit={() => setEditing(line)}
                onRemove={() => mutation.mutate({ type: 'remove', line })}
              />
            ))}
          </div>
        </section>
      ))}
      {(mutation.error || addMutation.error) && <p className="inline-error" role="alert">{(mutation.error ?? addMutation.error)?.message}</p>}
      <div className="grocery-footer">Week of {formatDate(query.data.startsOn)} · {completed} of {visibleLines.length} complete</div>

      {editing && (
        <OverrideDialog
          line={editing}
          pending={mutation.isPending}
          onClose={() => setEditing(null)}
          onSave={(value) => mutation.mutate({ type: 'override', line: editing, value })}
        />
      )}
    </section>
  )
}

function GroceryRow({ line, expanded, pending, onExpand, onComplete, onEdit, onRemove }: {
  line: GroceryLine
  expanded: boolean
  pending: boolean
  onExpand: () => void
  onComplete: () => void
  onEdit: () => void
  onRemove: () => void
}) {
  return (
    <article className={`grocery-row${line.completed ? ' completed' : ''}`}>
      <button className="check-button" type="button" onClick={onComplete} disabled={pending} aria-label={`${line.completed ? 'Mark incomplete' : 'Mark complete'}: ${line.name}`} aria-pressed={line.completed}>
        {line.completed && <CheckIcon />}
      </button>
      <div className="grocery-main">
        <div className="grocery-name">{line.name}{line.optional && <span className="optional-label">Optional</span>}</div>
        {line.origin === 'manual' && <small>Manual item</small>}
      </div>
      <div className="grocery-quantity">
        {line.quantity ?? ''}
        {line.quantity !== line.generatedQuantity && <small>Adjusted</small>}
      </div>
      <div className="grocery-actions">
        {line.hasContributions && <button className={`icon-button expand-button${expanded ? ' open' : ''}`} type="button" onClick={onExpand} aria-expanded={expanded} aria-label={`Show recipes for ${line.name}`}><ChevronIcon /></button>}
        {line.origin === 'generated' && <button className="icon-button" type="button" onClick={onEdit} aria-label={`Adjust quantity for ${line.name}`}><EditIcon /></button>}
        <button className="icon-button danger" type="button" onClick={onRemove} disabled={pending} aria-label={`Remove ${line.name}`}><TrashIcon /></button>
      </div>
      {expanded && <ContributionPanel lineId={line.id} />}
    </article>
  )
}

function ContributionPanel({ lineId }: { lineId: number }) {
  const query = useQuery({ queryKey: ['groceries', lineId, 'contributions'], queryFn: () => getGroceryContributions(lineId) })
  return (
    <div className="contribution-panel">
      <div className="contribution-heading">From this week’s recipes</div>
      {query.isLoading && <span className="muted">Loading…</span>}
      {query.error && <span className="contribution-error">{query.error.message}</span>}
      {query.data?.contributions.map((item, index) => (
        <div className="contribution" key={`${item.recipeId}-${index}`}>
          <Link to="/recipes/$recipeId" params={{ recipeId: String(item.recipeId) }}>{item.recipeName}</Link>
          <span>{item.sourceText}</span>
          {item.optional && <small>Optional</small>}
        </div>
      ))}
    </div>
  )
}

function OverrideDialog({ line, pending, onClose, onSave }: { line: GroceryLine; pending: boolean; onClose: () => void; onSave: (value: string) => void }) {
  const [value, setValue] = useState(line.quantity ?? '')
  return (
    <div className="dialog-layer" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <dialog className="picker-dialog" open aria-labelledby="override-title">
        <header><div><div className="eyebrow">Week-only edit</div><h2 id="override-title">Adjust {line.name}</h2></div><button className="icon-button" onClick={onClose} aria-label="Close"><CloseIcon /></button></header>
        <label>Displayed quantity<input value={value} onChange={(event) => setValue(event.target.value)} placeholder={line.generatedQuantity ?? 'No quantity'} autoFocus /></label>
        <p className="dialog-note">Generated requirement: {line.generatedQuantity ?? 'presence only'}. Clear the field to restore it.</p>
        <footer><button className="button" onClick={onClose}>Cancel</button><button className="button primary" disabled={pending} onClick={() => onSave(value)}>{pending ? 'Saving…' : 'Save'}</button></footer>
      </dialog>
    </div>
  )
}

function PageHeading({ count, completed }: { count: number; completed: number }) {
  return <header className="page-heading grocery-heading"><div><div className="eyebrow">Shopping list</div><h1 id="groceries-heading">Groceries</h1><p>{count ? `${count - completed} items left to pick up.` : 'Everything required for this week.'}</p></div><div className="completion-count"><strong>{completed}</strong><span>of {count} done</span></div></header>
}

function GroceriesSkeleton() {
  return <section aria-label="Loading groceries" aria-busy="true"><PageHeading count={0} completed={0} /><div className="recipe-list">{Array.from({ length: 6 }, (_, index) => <div className="skeleton-row" key={index} />)}</div></section>
}

function ErrorPanel({ message, onRetry }: { message: string; onRetry: () => void }) {
  return <div className="error-panel" role="alert"><div><strong>Couldn’t load groceries.</strong><p>{message}</p></div><button className="button" onClick={onRetry}>Try again</button></div>
}

function groupLines(lines: GroceryLine[]): [string, GroceryLine[]][] {
  const groups = new Map<string, GroceryLine[]>()
  for (const line of lines) groups.set(line.section, [...(groups.get(line.section) ?? []), line])
  return [...groups.entries()]
}

function runLineAction(action: LineAction): Promise<Groceries> {
  if (action.type === 'complete') return updateGroceryLine(action.line.id, { completed: !action.line.completed })
  if (action.type === 'remove') return removeGroceryLine(action.line.id)
  return updateGroceryLine(action.line.id, { overrideText: action.value })
}

function slug(value: string) { return value.toLowerCase().replace(/[^a-z0-9]+/g, '-') }
function formatDate(value: string) { return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(`${value}T00:00:00Z`)) }
