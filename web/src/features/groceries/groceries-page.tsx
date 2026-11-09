import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import {
  addGroceryLine,
  getGroceries,
  getGroceryContributions,
  removeGroceryLine,
  updateGroceryLine,
  getStoreSections,
  type Groceries,
  type GroceryLine,
} from '../../api/client'
import { QuickAdd, SectionPicker, SectionOrderEditor } from './shopping-controls'
import { CheckIcon, ChevronIcon, CloseIcon, EditIcon, TrashIcon } from '../../ui/icons'

const groceriesKey = ['week', 'groceries'] as const

type LineAction =
  | { type: 'complete'; line: GroceryLine }
  | { type: 'remove'; line: GroceryLine }
  | { type: 'section'; line: GroceryLine; sectionId: number }
  | { type: 'override'; line: GroceryLine; value: string }

export function GroceriesPage() {
  const queryClient = useQueryClient()
  const query = useQuery({ queryKey: groceriesKey, queryFn: getGroceries })
  const [expanded, setExpanded] = useState<number | null>(null)
  const sectionQuery = useQuery({ queryKey: ['store-sections'], queryFn: getStoreSections })
  const [ordering, setOrdering] = useState(false)
  const [placing, setPlacing] = useState<GroceryLine | null>(null)
  const [moving, setMoving] = useState<number | null>(null)
  const [hideDone, setHideDone] = useState(() => { try { return localStorage.getItem('groceries-hide-done') === 'true' } catch { return false } })
  const [openSections, setOpenSections] = useState<Set<string>>(new Set())
  const [held, setHeld] = useState<Set<number>>(new Set())
  const [undo, setUndo] = useState<GroceryLine | null>(null)
  const settleTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(() => () => { if (settleTimer.current) clearTimeout(settleTimer.current) }, [])
  function complete(line: GroceryLine) {
    if (!line.completed) {
      setHeld(previous => new Set([...previous, line.id]))
      setUndo(line)
      if (settleTimer.current) clearTimeout(settleTimer.current)
      settleTimer.current = setTimeout(() => setHeld(new Set()), 1800)
    } else { setHeld(previous => { const next = new Set(previous); next.delete(line.id); return next }) }
    mutation.mutate({ type: 'complete', line })
  }
  const [editing, setEditing] = useState<GroceryLine | null>(null)

  const mutation = useMutation({
    mutationKey: ['groceries'],
    mutationFn: runLineAction,
    onSuccess: (data) => {
      queryClient.setQueryData(groceriesKey, data)
      setEditing(null)
    },
    onError: (_error, action) => {
      if (action.type === 'complete') {
        setUndo(null)
        setHeld(previous => { const next = new Set(previous); next.delete(action.line.id); return next })
      }
    },
  })
  const addMutation = useMutation({
    mutationKey: ['groceries', 'add'],
    mutationFn: addGroceryLine,
    onSuccess: (data) => {
      queryClient.setQueryData(groceriesKey, data)
      const previousIds = new Set(query.data?.lines.map(line => line.id))
      const added = data.lines.find(line => line.origin === 'manual' && !previousIds.has(line.id))
      if (added?.section === 'Other') setPlacing(added)
    },
  })

  const visibleLines = query.data?.lines.filter((line) => !line.removed) ?? []
  const sections = groupLines(visibleLines)
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

  return (
    <section aria-labelledby="groceries-heading" aria-busy={mutation.isPending || addMutation.isPending}>
      <PageHeading count={visibleLines.length} completed={completed} />
      <div className="shopping-toolbar">
        <button className="button" onClick={() => setOrdering(true)} disabled={!sectionQuery.data}>Reorder sections</button>
        <button className={`button hide-done-toggle${hideDone ? ' active' : ''}`} aria-pressed={hideDone} onClick={() => {
          const next = !hideDone; setHideDone(next)
          try { localStorage.setItem('groceries-hide-done', String(next)) } catch { /* Storage is optional. */ }
        }}>{hideDone ? `${completed} done · show` : 'Hide done'}</button>
      </div>
      <QuickAdd pending={addMutation.isPending} onAdd={input => addMutation.mutateAsync(input)} />
      {placing && <div className="placement-panel"><div><strong>Where will you find {placing.name}?</strong><button className="icon-button" aria-label="Dismiss section picker" onClick={() => setPlacing(null)}><CloseIcon /></button></div><SectionPicker sections={sectionQuery.data?.sections ?? []} selected={placing.sectionId} pending={mutation.isPending} onPick={sectionId => {
        mutation.mutate({ type: 'section', line: placing, sectionId }, { onSuccess: () => setPlacing(null) })
      }} /></div>}
      {sectionQuery.error && <p role="alert" className="inline-error">{sectionQuery.error.message} <button className="button" onClick={() => void sectionQuery.refetch()}>Retry sections</button></p>}

      {sections.length === 0 ? (
        <div className="grocery-empty">No grocery items in this week.</div>
      ) : sections.map(([section, lines]) => {
        const allDone = lines.every(line => line.completed && !held.has(line.id))
        const collapsed = allDone && !openSections.has(section)
        const sorted = [...lines].sort((a, b) => Number(a.completed && !held.has(a.id)) - Number(b.completed && !held.has(b.id)))
        const shown = sorted.filter(line => !hideDone || !line.completed || held.has(line.id))
        return (
        <section className="grocery-section" key={section} aria-labelledby={`section-${slug(section)}`}>
          <header><h2 id={`section-${slug(section)}`}>{section}</h2>{allDone ? <button className="section-collapse" aria-expanded={!collapsed} aria-controls={`lines-${slug(section)}`} onClick={() => setOpenSections(previous => { const next = new Set(previous); if (next.has(section)) next.delete(section); else next.add(section); return next })}><CheckIcon /> {lines.length} done <ChevronIcon /></button> : <span>{lines.filter(line => !line.completed).length} left</span>}</header>
          <div className="grocery-lines" id={`lines-${slug(section)}`} hidden={collapsed}>
            {shown.map((line) => (
              <GroceryRow
                key={line.id}
                line={line}
                expanded={expanded === line.id}
                pending={mutation.isPending}
                onExpand={() => setExpanded(expanded === line.id ? null : line.id)}
                onComplete={() => complete(line)}
                onEdit={() => line.origin === 'manual' ? setMoving(moving === line.id ? null : line.id) : setEditing(line)}
                sectionPicker={moving === line.id && <div className="row-section-picker"><div><strong>Move to section</strong><button className="icon-button" aria-label="Close section picker" onClick={() => setMoving(null)}><CloseIcon /></button></div><SectionPicker sections={sectionQuery.data?.sections ?? []} selected={line.sectionId} pending={mutation.isPending} onPick={sectionId => mutation.mutate({ type: 'section', line, sectionId }, { onSuccess: () => setMoving(null) })} /></div>}
                onRemove={() => mutation.mutate({ type: 'remove', line })}
              />
            ))}
          </div>
        </section>
      )})}
      {undo && <div className="shopping-undo" role="status"><span>{undo.name} checked</span><button className="button" disabled={mutation.isPending} onClick={() => { mutation.mutate({ type: 'complete', line: { ...undo, completed: true } }); setUndo(null) }}>Undo</button><button className="icon-button" aria-label="Dismiss undo" onClick={() => setUndo(null)}><CloseIcon /></button></div>}
      {(mutation.error || addMutation.error) && <p className="inline-error" role="alert">{(mutation.error ?? addMutation.error)?.message}</p>}
      <div className="grocery-footer">Week of {formatDate(query.data.startsOn)} · {completed} of {visibleLines.length} complete</div>

      {ordering && <SectionOrderEditor sections={sectionQuery.data?.sections ?? []} onClose={() => setOrdering(false)} />}
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

function GroceryRow({ line, expanded, pending, onExpand, onComplete, onEdit, onRemove, sectionPicker }: {
  line: GroceryLine
  expanded: boolean
  pending: boolean
  onExpand: () => void
  onComplete: () => void
  onEdit: () => void
  onRemove: () => void
  sectionPicker?: ReactNode
}) {
  const row = useRef<HTMLElement>(null)
  const previousTop = useRef<number | null>(null)
  useLayoutEffect(() => {
    if (!row.current) return
    const top = row.current.getBoundingClientRect().top + window.scrollY
    if (previousTop.current !== null && top !== previousTop.current && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      row.current.animate([{ transform: `translateY(${previousTop.current - top}px)` }, { transform: 'translateY(0)' }], { duration: 180, easing: 'ease-out' })
    }
    previousTop.current = top
  })
  return (
    <article ref={row} className={`grocery-row${line.completed ? ' completed' : ''}`}>
      <button className="check-button" type="button" onClick={onComplete} disabled={pending} aria-label={`${line.completed ? 'Mark incomplete' : 'Mark complete'}: ${line.name}`} aria-pressed={line.completed}>
        {line.completed && <CheckIcon />}
      </button>
      <div className="grocery-main">
        <div className="grocery-name">{line.name}{line.optional && <span className="optional-label">Optional</span>}</div>
        {line.recipeNames.length > 0 && <small className="recipe-associations" title={line.recipeNames.join(', ')}>For {line.recipeNames.join(' · ')}</small>}
        {line.origin === 'manual' && <small>Manual item</small>}
      </div>
      <div className="grocery-quantity">
        {line.quantity ?? ''}
        {line.quantity !== line.generatedQuantity && <small>Adjusted</small>}
      </div>
      <div className="grocery-actions">
        {line.hasContributions && <button className={`icon-button expand-button${expanded ? ' open' : ''}`} type="button" onClick={onExpand} aria-expanded={expanded} aria-label={`Show recipes for ${line.name}`}><ChevronIcon /></button>}
        <button className="icon-button" type="button" onClick={onEdit} aria-label={line.origin === 'manual' ? `Change section for ${line.name}` : `Adjust quantity for ${line.name}`}><EditIcon /></button>
        <button className="icon-button danger" type="button" onClick={onRemove} disabled={pending} aria-label={`Remove ${line.name}`}><TrashIcon /></button>
      </div>
      {sectionPicker}
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
  if (action.type === 'section') return updateGroceryLine(action.line.id, { storeSectionId: action.sectionId })
  if (action.type === 'remove') return removeGroceryLine(action.line.id)
  return updateGroceryLine(action.line.id, { overrideText: action.value })
}

function slug(value: string) { return value.toLowerCase().replace(/[^a-z0-9]+/g, '-') }
function formatDate(value: string) { return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(`${value}T00:00:00Z`)) }
