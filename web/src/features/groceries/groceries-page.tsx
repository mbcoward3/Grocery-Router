import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent, type ReactNode, type RefObject } from 'react'
import {
  addGroceryLine,
  getGroceries,
  getGroceryContributions,
  getStoreSections,
  removeGroceryLine,
  updateGroceryLine,
  type Groceries,
  type GroceryLine,
  type StoreSection,
} from '../../api/client'
import { historyQueryOptions } from '../history/queries'
import { QuickAdd, SectionPicker, SectionOrderEditor } from './shopping-controls'
import { CheckIcon, ChevronIcon, CloseIcon, MoreIcon, SortIcon, TrashIcon } from '../../ui/icons'

const groceriesKey = ['week', 'groceries'] as const
const changeKey = ['groceries'] as const
// A checked row stays in place this long after the last check, so a mis-tap is easy to reverse.
const settleDelay = 1800
const undoDelay = 5000
// Motion stays short and only moves what changed: rows ease in, settle out, and checks pop.
const enterMotion = { duration: 180, easing: 'cubic-bezier(0, 0, .2, 1)' }
const leaveMotion = { duration: 200, easing: 'cubic-bezier(.4, 0, .2, 1)' }

type LineChange =
  | { type: 'complete'; lineId: number; completed: boolean }
  | { type: 'remove'; lineId: number }
  | { type: 'restore'; lineId: number }
  | { type: 'section'; lineId: number; sectionId: number }
  | { type: 'override'; lineId: number; value: string }

export function GroceriesPage() {
  const query = useQuery({ queryKey: groceriesKey, queryFn: getGroceries })
  const sectionQuery = useQuery({ queryKey: ['store-sections'], queryFn: getStoreSections })
  // With no open week, a household that has finished a week is pointed at that week's list and
  // at planning the next one. It is only needed then, and never blocks the page.
  const historyQuery = useQuery({ ...historyQueryOptions, enabled: query.isSuccess && query.data === null })
  const [detailId, setDetailId] = useState<number | null>(null)
  const [placingId, setPlacingId] = useState<number | null>(null)
  const quick = useLineChange()
  const edit = useLineChange()
  const add = useAddLine(setPlacingId)
  const settling = useSettling(settleDelay, leaveMotion.duration)
  // Rows present when the list first loads appear without motion; later arrivals ease in.
  const ready = useRef(false)
  useEffect(() => { if (query.data) ready.current = true })
  const [ordering, setOrdering] = useState(false)
  const [doneOpen, setDoneOpen] = useState(false)
  const [removed, setRemoved] = useState<GroceryLine | null>(null)

  useEffect(() => {
    if (!removed) return
    const timer = setTimeout(() => setRemoved(null), undoDelay)
    return () => clearTimeout(timer)
  }, [removed])

  if (query.isLoading || (query.data === null && historyQuery.isLoading)) return <GroceriesSkeleton />
  if (query.error) return <ErrorPanel message={query.error.message} onRetry={() => void query.refetch()} />
  const lastWeek = query.data ? undefined : historyQuery.data?.weeks?.[0]
  if (lastWeek) {
    return (
      <section>
        <PageHeading />
        <div className="empty-week">
          <div className="empty-mark" aria-hidden="true">✦</div>
          <h2>Last week’s list is in History</h2>
          <p>A new grocery checklist starts when you plan next week.</p>
          <div className="empty-actions">
            <Link className="button" to="/history/$weekId" params={{ weekId: String(lastWeek.id) }}>See last week’s list</Link>
            <Link className="button primary" to="/">Plan next week</Link>
          </div>
        </div>
      </section>
    )
  }
  if (!query.data) {
    return (
      <section>
        <PageHeading />
        <div className="empty-week">
          <div className="empty-mark" aria-hidden="true">✦</div>
          <h2>Generate a week first</h2>
          <p>Your grocery checklist is built from the recipes in the current pool.</p>
          <Link className="button primary" to="/">Go to week</Link>
        </div>
      </section>
    )
  }

  const lines = query.data.lines.filter(line => !line.removed)
  const active = lines.filter(line => !line.completed || settling.ids.has(line.id))
  const done = lines.filter(line => line.completed && !settling.ids.has(line.id))
  const left = lines.filter(line => !line.completed).length
  const sections = sectionQuery.data?.sections ?? []
  const detail = lines.find(line => line.id === detailId)
  const placing = lines.find(line => line.id === placingId && line.origin === 'manual')
  const error = quick.error ?? add.error
  const leaving = (line: GroceryLine) => settling.leaving && line.completed && settling.ids.has(line.id)

  function toggle(line: GroceryLine) {
    const completed = !line.completed
    if (completed) settling.hold(line.id)
    else settling.release(line.id)
    quick.mutate({ type: 'complete', lineId: line.id, completed })
  }
  function remove(line: GroceryLine) {
    setDetailId(null)
    if (placingId === line.id) setPlacingId(null)
    quick.mutate({ type: 'remove', lineId: line.id })
    setRemoved(line)
  }
  function undoRemove() {
    if (!removed) return
    quick.mutate({ type: 'restore', lineId: removed.id })
    setRemoved(null)
  }

  return (
    <section aria-labelledby="groceries-heading">
      <PageHeading weekOf={query.data.startsOn} left={left} total={lines.length}>
        <button className="icon-button quiet" type="button" title="Reorder sections" aria-label="Reorder sections" disabled={!sectionQuery.data} onClick={() => setOrdering(true)}><SortIcon /></button>
      </PageHeading>

      <QuickAdd pending={add.isPending} onAdd={input => add.mutateAsync(input)} />
      {placing && (
        <div className="placement" role="group" aria-label={`Choose a section for ${placing.name}`}>
          <div className="placement-head">
            <span>Where will you find <strong>{placing.name}</strong>? <span className="muted">We’ll remember.</span></span>
            <button className="icon-button quiet" type="button" aria-label="Dismiss section picker" onClick={() => setPlacingId(null)}><CloseIcon /></button>
          </div>
          <SectionPicker sections={sections} selected={placing.sectionId} pending={edit.isPending} onPick={sectionId => {
            edit.mutate({ type: 'section', lineId: placing.id, sectionId }, { onSuccess: () => setPlacingId(null) })
          }} />
        </div>
      )}
      {sectionQuery.error && <p role="alert" className="inline-error">{sectionQuery.error.message} <button className="button" onClick={() => void sectionQuery.refetch()}>Retry sections</button></p>}
      {error && <p className="inline-error" role="alert">{error.message}</p>}

      {lines.length === 0 && <div className="grocery-empty">No grocery items this week.</div>}
      {lines.length > 0 && active.length === 0 && <p className="grocery-clear"><CheckIcon /> Everything’s in the cart.</p>}
      {groupBySection(active).map(([section, sectionLines]) => (
        <Animated as="section" className="grocery-section" key={section} aria-labelledby={`section-${slug(section)}`} enter={ready.current} leave={sectionLines.every(leaving)}>
          <header>
            <h2 id={`section-${slug(section)}`}>{section}</h2>
            <span className="grocery-count">{sectionLines.filter(line => !line.completed).length}</span>
          </header>
          <ul className="grocery-lines">
            {sectionLines.map(line => (
              <GroceryRow key={line.id} line={line} meta={line.recipeNames.join(' · ')} enter={ready.current} leave={leaving(line)} onToggle={() => toggle(line)} onOpen={() => setDetailId(line.id)} />
            ))}
          </ul>
        </Animated>
      ))}

      {done.length > 0 && (
        <Animated as="section" className="grocery-section done-group" aria-labelledby="done-heading" enter={ready.current}>
          <header>
            <h2 id="done-heading">
              <button className="done-toggle" type="button" aria-expanded={doneOpen} aria-controls="done-lines" onClick={() => setDoneOpen(open => !open)}>
                Done <span className="grocery-count">{done.length}</span><ChevronIcon />
              </button>
            </h2>
          </header>
          {doneOpen && (
            <ul className="grocery-lines" id="done-lines">
              {done.map(line => <GroceryRow key={line.id} line={line} meta={line.section} enter={ready.current} onToggle={() => toggle(line)} onOpen={() => setDetailId(line.id)} />)}
            </ul>
          )}
        </Animated>
      )}

      {removed && (
        <div className="grocery-toast" role="status">
          <span>Removed {removed.name}</span>
          <button className="toast-action" type="button" onClick={undoRemove}>Undo</button>
        </div>
      )}
      {ordering && <SectionOrderEditor sections={sections} onClose={() => setOrdering(false)} />}
      {detail && (
        <LineSheet
          key={detail.id}
          line={detail}
          sections={sections}
          pending={edit.isPending}
          error={edit.error?.message}
          onClose={() => { setDetailId(null); edit.reset() }}
          onChange={change => edit.mutate(change, { onSuccess: () => setDetailId(null) })}
          onRemove={() => remove(detail)}
        />
      )}
    </section>
  )
}

// Line changes update the screen immediately and save in the background, so quick taps while
// shopping are never blocked. Overlapping saves only apply the last response in flight.
function useLineChange() {
  const client = useQueryClient()
  return useMutation({
    mutationKey: changeKey,
    mutationFn: saveChange,
    onMutate: async (change: LineChange) => {
      const patch = optimisticPatch(change)
      if (!patch) return
      await client.cancelQueries({ queryKey: groceriesKey })
      client.setQueryData<Groceries | null>(groceriesKey, data => data && {
        ...data,
        lines: data.lines.map(line => line.id === change.lineId ? { ...line, ...patch } : line),
      })
    },
    onSuccess: (data, change) => {
      if (client.isMutating({ mutationKey: changeKey }) <= 1) client.setQueryData(groceriesKey, data)
      // Placing an unknown item teaches the household's suggestions.
      if (change.type === 'section') void client.invalidateQueries({ queryKey: ['grocery-items'] })
    },
    onError: () => void client.invalidateQueries({ queryKey: groceriesKey }),
  })
}

function useAddLine(onUnknown: (lineId: number) => void) {
  const client = useQueryClient()
  return useMutation({
    mutationKey: [...changeKey, 'add'],
    mutationFn: addGroceryLine,
    onSuccess: data => {
      const before = new Set(client.getQueryData<Groceries | null>(groceriesKey)?.lines.map(line => line.id))
      client.setQueryData(groceriesKey, data)
      void client.invalidateQueries({ queryKey: ['grocery-items'] })
      const added = data.lines.find(line => line.origin === 'manual' && !before.has(line.id))
      if (added?.section === 'Other') onUnknown(added.id)
    },
  })
}

// Checked rows hold their place until checks pause, then collapse out together.
function useSettling(delay: number, leaveDuration: number) {
  const [ids, setIds] = useState<ReadonlySet<number>>(() => new Set())
  const [leaving, setLeaving] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined)
  useEffect(() => () => clearTimeout(timer.current), [])
  const hold = useCallback((id: number) => {
    setIds(previous => new Set(previous).add(id))
    setLeaving(false)
    clearTimeout(timer.current)
    timer.current = setTimeout(() => {
      setLeaving(true)
      timer.current = setTimeout(() => { setIds(new Set()); setLeaving(false) }, leaveDuration)
    }, delay)
  }, [delay, leaveDuration])
  const release = useCallback((id: number) => {
    setIds(previous => {
      if (!previous.has(id)) return previous
      const next = new Set(previous); next.delete(id); return next
    })
  }, [])
  return { ids, leaving, hold, release }
}

function reducedMotion() { return window.matchMedia('(prefers-reduced-motion: reduce)').matches }

// Height, opacity, and bottom margin animate together so neighbours glide instead of jumping.
function useEnterLeave(ref: RefObject<HTMLElement | null>, enter: boolean, leave: boolean) {
  const shouldEnter = useRef(enter)
  useLayoutEffect(() => {
    const element = ref.current
    if (!shouldEnter.current || !element || reducedMotion()) return
    const { height, marginBottom } = getComputedStyle(element)
    clip(element, element.animate([{ height: '0px', marginBottom: '0px', opacity: 0 }, { height, marginBottom, opacity: 1 }], enterMotion))
  }, [ref])
  useLayoutEffect(() => {
    const element = ref.current
    if (!leave || !element || reducedMotion()) return
    const { height, marginBottom } = getComputedStyle(element)
    const animation = element.animate([{ height, marginBottom, opacity: 1 }, { height: '0px', marginBottom: '0px', opacity: 0 }], { ...leaveMotion, fill: 'forwards' })
    clip(element, animation)
    return () => animation.cancel()
  }, [ref, leave])
}

function clip(element: HTMLElement, animation: Animation) {
  element.style.overflow = 'hidden'
  const restore = () => { element.style.overflow = '' }
  animation.addEventListener('finish', restore)
  animation.addEventListener('cancel', restore)
}

function Animated({ as: Tag, enter, leave = false, children, ...props }: { as: 'section'; enter: boolean; leave?: boolean; className: string; children: ReactNode; 'aria-labelledby': string }) {
  const element = useRef<HTMLElement>(null)
  useEnterLeave(element, enter, leave)
  return <Tag ref={element} {...props}>{children}</Tag>
}

function GroceryRow({ line, meta, enter, leave = false, onToggle, onOpen }: { line: GroceryLine; meta: string; enter: boolean; leave?: boolean; onToggle: () => void; onOpen: () => void }) {
  const row = useRef<HTMLLIElement>(null)
  const check = useRef<HTMLSpanElement>(null)
  const wasCompleted = useRef(line.completed)
  useEnterLeave(row, enter, leave)
  // The check pops only when it is ticked here, not when a done row appears.
  useLayoutEffect(() => {
    if (line.completed && !wasCompleted.current && !reducedMotion()) {
      check.current?.animate([{ transform: 'scale(.6)' }, { transform: 'scale(1.1)', offset: .6 }, { transform: 'scale(1)' }], { duration: 240, easing: 'ease-out' })
    }
    wasCompleted.current = line.completed
  }, [line.completed])
  const adjusted = line.quantity !== line.generatedQuantity
  return (
    <li ref={row} className={`grocery-row${line.completed ? ' completed' : ''}`}>
      <button className="grocery-toggle" type="button" onClick={onToggle} aria-pressed={line.completed} aria-label={`${line.completed ? 'Mark incomplete' : 'Mark complete'}: ${line.name}`}>
        <span ref={check} className="grocery-check" aria-hidden="true">{line.completed && <CheckIcon />}</span>
        <span className="grocery-text">
          <span className="grocery-name">{line.name}{line.optional && <span className="optional-label">Optional</span>}</span>
          {meta && <span className="grocery-meta">{meta}</span>}
        </span>
        {line.quantity && <span className={`grocery-quantity${adjusted ? ' adjusted' : ''}`} title={adjusted ? 'Adjusted for this week' : undefined}>{line.quantity}</span>}
      </button>
      <button className="row-more" type="button" onClick={onOpen} aria-label={`Details for ${line.name}`}><MoreIcon /></button>
    </li>
  )
}

function LineSheet({ line, sections, pending, error, onClose, onChange, onRemove }: {
  line: GroceryLine
  sections: StoreSection[]
  pending: boolean
  error?: string
  onClose: () => void
  onChange: (change: LineChange) => void
  onRemove: () => void
}) {
  const dialog = useRef<HTMLDialogElement>(null)
  const [value, setValue] = useState(line.quantity ?? '')
  useEffect(() => { const element = dialog.current; element?.showModal(); return () => element?.close() }, [])
  const manual = line.origin === 'manual'
  function save(event: FormEvent) {
    event.preventDefault()
    onChange({ type: 'override', lineId: line.id, value })
  }
  return (
    <dialog ref={dialog} className="line-sheet" aria-labelledby="line-sheet-title"
      onCancel={event => { event.preventDefault(); onClose() }}
      onClick={event => { if (event.target === event.currentTarget) onClose() }}>
      <header>
        <div><div className="sheet-section">{line.section}</div><h2 id="line-sheet-title">{line.name}</h2></div>
        <button className="icon-button quiet" type="button" onClick={onClose} aria-label="Close"><CloseIcon /></button>
      </header>
      {manual ? (
        <div className="sheet-field">
          <span className="sheet-label">Section</span>
          <SectionPicker sections={sections} selected={line.sectionId} pending={pending} onPick={sectionId => onChange({ type: 'section', lineId: line.id, sectionId })} />
        </div>
      ) : (
        <form className="sheet-field" onSubmit={save}>
          <label className="sheet-label" htmlFor="line-quantity">Quantity this week</label>
          <div className="sheet-input">
            <input id="line-quantity" value={value} onChange={event => setValue(event.target.value)} placeholder={line.generatedQuantity ?? 'No quantity'} autoComplete="off" />
            <button className="button" type="submit" disabled={pending || value === (line.quantity ?? '')}>{pending ? 'Saving…' : 'Save'}</button>
          </div>
          <p className="sheet-note">Recipes call for {line.generatedQuantity ?? 'no set amount'}. Clear to restore.</p>
        </form>
      )}
      {line.hasContributions && <Contributions lineId={line.id} />}
      {error && <p className="inline-error sheet-error" role="alert">{error}</p>}
      <footer>
        <button className="sheet-remove" type="button" onClick={onRemove}><TrashIcon /> Remove from list</button>
      </footer>
    </dialog>
  )
}

function Contributions({ lineId }: { lineId: number }) {
  const query = useQuery({ queryKey: ['groceries', lineId, 'contributions'], queryFn: () => getGroceryContributions(lineId) })
  return (
    <div className="sheet-field">
      <span className="sheet-label">For this week’s recipes</span>
      {query.isLoading && <span className="muted">Loading…</span>}
      {query.error && <span className="contribution-error">{query.error.message}</span>}
      {query.data && (
        <ul className="contributions">
          {query.data.contributions.map((item, index) => (
            <li key={`${item.recipeId}-${index}`}>
              <Link to="/recipes/$recipeId" params={{ recipeId: String(item.recipeId) }}>{item.recipeName}</Link>
              <span>{item.sourceText}{item.optional && ' · optional'}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function PageHeading({ weekOf, left, total, children }: { weekOf?: string; left?: number; total?: number; children?: ReactNode }) {
  const progress = total ? (total - (left ?? 0)) / total : 0
  return (
    <header className="page-heading grocery-heading">
      <div>
        <div className="eyebrow">Shopping list</div>
        <h1 id="groceries-heading">Groceries</h1>
        {weekOf && <p>Week of {formatDate(weekOf)}</p>}
      </div>
      {total ? (
        <div className="grocery-heading-meta">
          <span className="grocery-progress" aria-label={`${total - (left ?? 0)} of ${total} done`}>
            <ProgressRing value={progress} />{left ? `${left} left` : 'All done'}
          </span>
          {children}
        </div>
      ) : null}
    </header>
  )
}

function ProgressRing({ value }: { value: number }) {
  const circumference = 2 * Math.PI * 6
  return (
    <svg className="progress-ring" viewBox="0 0 16 16" aria-hidden="true">
      <circle cx="8" cy="8" r="6" />
      <circle cx="8" cy="8" r="6" strokeDasharray={circumference} strokeDashoffset={circumference * (1 - value)} />
    </svg>
  )
}

function GroceriesSkeleton() {
  return <section aria-label="Loading groceries" aria-busy="true"><PageHeading /><div className="recipe-list">{Array.from({ length: 6 }, (_, index) => <div className="skeleton-row" key={index} />)}</div></section>
}

function ErrorPanel({ message, onRetry }: { message: string; onRetry: () => void }) {
  return <div className="error-panel" role="alert"><div><strong>Couldn’t load groceries.</strong><p>{message}</p></div><button className="button" onClick={onRetry}>Try again</button></div>
}

function groupBySection(lines: GroceryLine[]): [string, GroceryLine[]][] {
  const groups = new Map<string, GroceryLine[]>()
  for (const line of lines) groups.set(line.section, [...(groups.get(line.section) ?? []), line])
  return [...groups.entries()]
}

function optimisticPatch(change: LineChange): Partial<GroceryLine> | null {
  if (change.type === 'complete') return { completed: change.completed }
  if (change.type === 'remove') return { removed: true }
  if (change.type === 'restore') return { removed: false }
  return null
}

function saveChange(change: LineChange): Promise<Groceries> {
  if (change.type === 'complete') return updateGroceryLine(change.lineId, { completed: change.completed })
  if (change.type === 'remove') return removeGroceryLine(change.lineId)
  if (change.type === 'restore') return updateGroceryLine(change.lineId, { removed: false })
  if (change.type === 'section') return updateGroceryLine(change.lineId, { storeSectionId: change.sectionId })
  return updateGroceryLine(change.lineId, { overrideText: change.value })
}

function slug(value: string) { return value.toLowerCase().replace(/[^a-z0-9]+/g, '-') }
function formatDate(value: string) { return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(new Date(`${value}T00:00:00Z`)) }
