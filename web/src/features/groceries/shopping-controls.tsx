import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { searchGroceryItems, setStoreSections, type GroceryItemMatch, type StoreSection } from '../../api/client'
import { CloseIcon, PlusIcon } from '../../ui/icons'

type Addition = { name: string; groceryItemId?: number }

export function QuickAdd({ pending, onAdd }: { pending: boolean; onAdd: (input: Addition) => Promise<unknown> }) {
  const [text, setText] = useState('')
  const [term, setTerm] = useState('')
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(-1)
  const input = useRef<HTMLInputElement>(null)
  useEffect(() => { const timer = setTimeout(() => setTerm(text.trim()), 150); return () => clearTimeout(timer) }, [text])
  const query = useQuery({ queryKey: ['grocery-items', term], queryFn: () => searchGroceryItems(term), enabled: term.length > 0 })
  const items = term === text.trim() ? query.data?.items ?? [] : []
  const show = open && text.trim().length > 0 && items.length > 0
  async function add(item?: GroceryItemMatch) {
    if (pending || !text.trim()) return
    try {
      await onAdd({ name: item?.name ?? text.trim(), ...(item ? { groceryItemId: item.id } : {}) })
      setText(''); setTerm(''); setOpen(false); setActive(-1); input.current?.focus()
    } catch { /* The parent presents the mutation error and retains the input. */ }
  }
  function submit(event: FormEvent) { event.preventDefault(); void add(show && active >= 0 ? items[active] : undefined) }
  return <div className="quick-add-wrap">
    <form className="quick-add" onSubmit={submit}>
      <PlusIcon /><label className="sr-only" htmlFor="new-grocery">Add grocery item</label>
      <input ref={input} id="new-grocery" role="combobox" aria-autocomplete="list" aria-expanded={show} aria-controls="grocery-suggestions" aria-activedescendant={show && active >= 0 ? `grocery-suggestion-${active}` : undefined}
        value={text} placeholder="Add an item…" autoComplete="off" onFocus={() => setOpen(true)} onBlur={() => setOpen(false)}
        onChange={event => { setText(event.target.value); setOpen(true); setActive(-1) }}
        onKeyDown={event => {
          if (event.key === 'Escape') { setOpen(false); setActive(-1) }
          if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
            event.preventDefault(); setOpen(true)
            setActive(previous => items.length ? (previous + (event.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length : -1)
          }
        }} />
      <button className="button" type="submit" disabled={!text.trim() || pending}>{pending ? 'Adding…' : 'Add'}</button>
    </form>
    {show && <ul className="grocery-suggestions" id="grocery-suggestions" role="listbox" aria-label="Known grocery items">
      {items.map((item, index) => <li key={item.id} id={`grocery-suggestion-${index}`} role="option" aria-selected={active === index} onMouseDown={event => event.preventDefault()} onClick={() => void add(item)}><strong>{item.name}</strong><span>{item.section}</span></li>)}
    </ul>}
    {query.error && open && <p className="muted">Suggestions unavailable. You can still add your item.</p>}
  </div>
}

export function SectionPicker({ sections, selected, pending, onPick }: { sections: StoreSection[]; selected: number; pending: boolean; onPick: (id: number) => void }) {
  return <div className="section-chips" role="group" aria-label="Store section">{sections.map(section => <button className={`section-chip${section.id === selected ? ' selected' : ''}`} key={section.id} aria-pressed={section.id === selected} disabled={pending} onClick={() => onPick(section.id)}>{section.name}</button>)}</div>
}

export function SectionOrderEditor({ sections, onClose }: { sections: StoreSection[]; onClose: () => void }) {
  const [order, setOrder] = useState(sections)
  const [dragged, setDragged] = useState<number | null>(null)
  const [announcement, setAnnouncement] = useState('')
  const dialog = useRef<HTMLDialogElement>(null)
  const client = useQueryClient()
  useEffect(() => { const element = dialog.current; element?.showModal(); return () => element?.close() }, [])
  const save = useMutation({ mutationFn: setStoreSections, onSuccess: async data => {
    client.setQueryData(['store-sections'], data)
    await Promise.all([client.invalidateQueries({ queryKey: ['week', 'groceries'] }), client.invalidateQueries({ queryKey: ['weeks', 'history'] })])
    onClose()
  } })
  function move(from: number, to: number) {
    if (save.isPending || to < 0 || to >= order.length || from === to) return
    const next = [...order]; const [section] = next.splice(from, 1); if (!section) return; next.splice(to, 0, section)
    setOrder(next); setAnnouncement(`${section.name} moved to position ${to + 1} of ${next.length}`)
  }
  return <dialog ref={dialog} className="picker-dialog section-order-dialog" aria-labelledby="section-order-title" onCancel={event => { event.preventDefault(); if (!save.isPending) onClose() }}>
    <header><div><div className="eyebrow">Your store walk</div><h2 id="section-order-title">Reorder sections</h2></div><button className="icon-button" disabled={save.isPending} aria-label="Close section order" onClick={onClose}><CloseIcon /></button></header>
    <p className="dialog-note">Start where you walk in. Drag the handles or use the arrows. This order is shared with your household for every week.</p>
    <div className="sr-only" role="status" aria-live="polite">{announcement}</div>
    <ol className="section-order-list">{order.map((section, index) => <li key={section.id} onDragOver={event => { if (dragged !== null) event.preventDefault() }} onDrop={event => { event.preventDefault(); if (dragged !== null) move(dragged, index); setDragged(null) }}>
      <span className="drag-handle" draggable={!save.isPending} onDragStart={event => { setDragged(index); event.dataTransfer.effectAllowed = 'move'; event.dataTransfer.setData('text/plain', String(section.id)) }} onDragEnd={() => setDragged(null)} aria-hidden="true">⠿</span>
      <span className="section-position">{index + 1}</span><strong>{section.name}</strong>
      <button className="icon-button" disabled={index === 0 || save.isPending} aria-label={`Move ${section.name} up`} onClick={() => move(index, index - 1)}>↑</button>
      <button className="icon-button" disabled={index === order.length - 1 || save.isPending} aria-label={`Move ${section.name} down`} onClick={() => move(index, index + 1)}>↓</button>
    </li>)}</ol>
    {save.error && <p className="inline-error" role="alert">{save.error.message}</p>}
    <footer><button className="button" disabled={save.isPending} onClick={() => save.mutate({ reset: true })}>Reset to default</button><button className="button primary" disabled={save.isPending} onClick={() => save.mutate({ sectionIds: order.map(section => section.id) })}>{save.isPending ? 'Saving…' : 'Save order'}</button></footer>
  </dialog>
}
