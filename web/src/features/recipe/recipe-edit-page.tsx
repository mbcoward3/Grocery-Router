import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from '@tanstack/react-router'
import { useEffect, useId, useRef, useState, type FormEvent, type ReactNode } from 'react'
import {
  ApiRequestError,
  editRecipe,
  getRecipe,
  getStoreSections,
  searchGroceryItems,
  type GroceryItemMatch,
  type RecipeDetail,
  type StoreSection,
} from '../../api/client'
import { CloseIcon, PlusIcon } from '../../ui/icons'
import {
  draftFromRecipe,
  draftKey,
  editFromDraft,
  emptyLine,
  groupProblems,
  messagesFor,
  move,
  shoppingModes,
  units,
  type IngredientSectionDraft,
  type InstructionSectionDraft,
  type ItemChoice,
  type LineDraft,
  type RecipeDraft,
} from './recipe-draft'

export function RecipeEditPage() {
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
        <Link className="button" to="/recipes">Back to recipes</Link>
      </div>
    )
  }
  return <RecipeEditor recipe={query.data} />
}

type Problems = Record<string, string[]>

function RecipeEditor({ recipe }: { recipe: RecipeDetail }) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [draft, setDraft] = useState<RecipeDraft>(() => draftFromRecipe(recipe))
  const [problems, setProblems] = useState<Problems>({})
  const sections = useQuery({ queryKey: ['store-sections'], queryFn: getStoreSections })
  const form = useRef<HTMLFormElement>(null)

  const mutation = useMutation({
    mutationFn: () => editRecipe(recipe.id, editFromDraft(draft)),
    onSuccess: async (saved) => {
      queryClient.setQueryData(['recipes', recipe.id], saved)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['recipes'] }),
        queryClient.invalidateQueries({ queryKey: ['week'] }),
        queryClient.invalidateQueries({ queryKey: ['grocery-items'] }),
      ])
      await navigate({ to: '/recipes/$recipeId', params: { recipeId: String(recipe.id) } })
    },
    onError: (error) => {
      if (error instanceof ApiRequestError && error.fields.length > 0) setProblems(groupProblems(error.fields))
    },
  })

  // After a rejected save, focus the first field it names.
  useEffect(() => {
    if (Object.keys(problems).length === 0) return
    form.current?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus()
  }, [problems])

  // Row positions name fields in errors, so a structural change makes them stale.
  function restructure(change: (current: RecipeDraft) => RecipeDraft) {
    setProblems({})
    setDraft(change)
  }
  function update(change: Partial<RecipeDraft>) {
    setDraft((current) => ({ ...current, ...change }))
  }

  function submit(event: FormEvent) {
    event.preventDefault()
    const missingSection: Problems = {}
    draft.ingredientSections.forEach((section, sectionIndex) => section.lines.forEach((line, lineIndex) => {
      if (line.item.kind === 'new' && line.item.storeSectionId === null) {
        missingSection[`ingredientSections[${sectionIndex}].ingredients[${lineIndex}].newGroceryItem.storeSectionId`] = ['Choose a store section for the new item.']
      }
    }))
    if (Object.keys(missingSection).length > 0) {
      setProblems(missingSection)
      return
    }
    setProblems({})
    mutation.mutate()
  }

  const rejected = mutation.error instanceof ApiRequestError && mutation.error.fields.length > 0
  const storeSections = sections.data?.sections ?? []
  return (
    <form className="recipe-edit" ref={form} onSubmit={submit} noValidate aria-labelledby="recipe-edit-heading">
      <Link className="recipe-back" to="/recipes/$recipeId" params={{ recipeId: String(recipe.id) }}>
        <span aria-hidden="true">←</span> {recipe.name}
      </Link>
      <header className="recipe-edit-header">
        <div>
          <div className="eyebrow">Edit recipe</div>
          <h1 id="recipe-edit-heading">{draft.name.trim() || 'Untitled recipe'}</h1>
          <p>Saving applies at once and counts as the household’s approval. Past plans keep what they showed.</p>
        </div>
      </header>

      {(mutation.error || Object.keys(problems).length > 0) && (
        <div className="inline-error" role="alert">
          {rejected || Object.keys(problems).length > 0
            ? 'The recipe was not saved. Fix the marked fields.'
            : `The recipe was not saved. ${mutation.error?.message ?? ''}`}
        </div>
      )}

      <section className="edit-panel" aria-labelledby="edit-details-heading">
        <h2 id="edit-details-heading">Details</h2>
        <TextField label="Name" value={draft.name} onChange={(name) => update({ name })} problems={messagesFor(problems, 'name')} />
        <TextField label="Image URL" value={draft.imageUrl} onChange={(imageUrl) => update({ imageUrl })} problems={messagesFor(problems, 'imageUrl')} type="url" />
        <TextField label="Yield" value={draft.yield} onChange={(value) => update({ yield: value })} problems={messagesFor(problems, 'yield')} placeholder="Serves 4" />
        <MinutesField label="Hands-on minutes" minimum={draft.handsOnMin} maximum={draft.handsOnMax}
          onChange={(handsOnMin, handsOnMax) => update({ handsOnMin, handsOnMax })} problems={messagesFor(problems, 'handsOn')} />
        <MinutesField label="Unattended minutes" minimum={draft.unattendedMin} maximum={draft.unattendedMax}
          onChange={(unattendedMin, unattendedMax) => update({ unattendedMin, unattendedMax })} problems={messagesFor(problems, 'unattended')} />
      </section>

      <section className="edit-panel" aria-labelledby="edit-ingredients-heading">
        <h2 id="edit-ingredients-heading">Ingredients</h2>
        <FieldProblems messages={messagesFor(problems, 'ingredientSections')} />
        {draft.ingredientSections.map((section, sectionIndex) => (
          <IngredientSectionEditor
            key={section.key}
            section={section}
            path={`ingredientSections[${sectionIndex}]`}
            index={sectionIndex}
            count={draft.ingredientSections.length}
            problems={problems}
            storeSections={storeSections}
            onChange={(next) => setDraft((current) => ({
              ...current,
              ingredientSections: current.ingredientSections.map((existing, index) => (index === sectionIndex ? next : existing)),
            }))}
            onRestructure={(next) => restructure((current) => ({
              ...current,
              ingredientSections: current.ingredientSections.map((existing, index) => (index === sectionIndex ? next : existing)),
            }))}
            onMove={(offset) => restructure((current) => ({ ...current, ingredientSections: move(current.ingredientSections, sectionIndex, offset) }))}
            onRemove={() => restructure((current) => ({ ...current, ingredientSections: current.ingredientSections.filter((_, index) => index !== sectionIndex) }))}
          />
        ))}
        <button className="button" type="button" onClick={() => restructure((current) => ({
          ...current,
          ingredientSections: [...current.ingredientSections, { key: draftKey(), name: 'Ingredients', lines: [emptyLine()] }],
        }))}><PlusIcon /> Add ingredient section</button>
      </section>

      <section className="edit-panel" aria-labelledby="edit-steps-heading">
        <h2 id="edit-steps-heading">Instructions</h2>
        <FieldProblems messages={messagesFor(problems, 'instructionSections')} />
        {draft.instructionSections.map((section, sectionIndex) => (
          <InstructionSectionEditor
            key={section.key}
            section={section}
            path={`instructionSections[${sectionIndex}]`}
            index={sectionIndex}
            count={draft.instructionSections.length}
            problems={problems}
            onChange={(next) => setDraft((current) => ({
              ...current,
              instructionSections: current.instructionSections.map((existing, index) => (index === sectionIndex ? next : existing)),
            }))}
            onRestructure={(next) => restructure((current) => ({
              ...current,
              instructionSections: current.instructionSections.map((existing, index) => (index === sectionIndex ? next : existing)),
            }))}
            onMove={(offset) => restructure((current) => ({ ...current, instructionSections: move(current.instructionSections, sectionIndex, offset) }))}
            onRemove={() => restructure((current) => ({ ...current, instructionSections: current.instructionSections.filter((_, index) => index !== sectionIndex) }))}
          />
        ))}
        <button className="button" type="button" onClick={() => restructure((current) => ({
          ...current,
          instructionSections: [...current.instructionSections, { key: draftKey(), name: 'Instructions', steps: [{ key: draftKey(), instruction: '' }] }],
        }))}><PlusIcon /> Add instruction section</button>
      </section>

      <footer className="recipe-edit-footer">
        <Link className="button" to="/recipes/$recipeId" params={{ recipeId: String(recipe.id) }}>Cancel</Link>
        <button className="button primary" type="submit" disabled={mutation.isPending}>{mutation.isPending ? 'Saving…' : 'Save recipe'}</button>
      </footer>
    </form>
  )
}

function IngredientSectionEditor({ section, path, index, count, problems, storeSections, onChange, onRestructure, onMove, onRemove }: {
  section: IngredientSectionDraft
  path: string
  index: number
  count: number
  problems: Problems
  storeSections: StoreSection[]
  onChange: (section: IngredientSectionDraft) => void
  onRestructure: (section: IngredientSectionDraft) => void
  onMove: (offset: number) => void
  onRemove: () => void
}) {
  const setLine = (lineIndex: number, line: LineDraft) =>
    onChange({ ...section, lines: section.lines.map((existing, position) => (position === lineIndex ? line : existing)) })
  return (
    <fieldset className="edit-section">
      <legend className="sr-only">Ingredient section {index + 1}</legend>
      <div className="edit-section-head">
        <TextField label="Section name" value={section.name} onChange={(name) => onChange({ ...section, name })} problems={messagesFor(problems, path)} />
        <RowControls noun="ingredient section" index={index} count={count} onMove={onMove} onRemove={onRemove} />
      </div>
      <ol className="edit-lines">
        {section.lines.map((line, lineIndex) => (
          <li key={line.key}>
            <LineEditor
              line={line}
              path={`${path}.ingredients[${lineIndex}]`}
              number={lineIndex + 1}
              problems={problems}
              storeSections={storeSections}
              onChange={(next) => setLine(lineIndex, next)}
            />
            <RowControls noun="ingredient" index={lineIndex} count={section.lines.length}
              onMove={(offset) => onRestructure({ ...section, lines: move(section.lines, lineIndex, offset) })}
              onRemove={() => onRestructure({ ...section, lines: section.lines.filter((_, position) => position !== lineIndex) })} />
          </li>
        ))}
      </ol>
      <button className="button quiet-button" type="button" onClick={() => onRestructure({ ...section, lines: [...section.lines, emptyLine()] })}>
        <PlusIcon /> Add ingredient
      </button>
    </fieldset>
  )
}

function LineEditor({ line, path, number, problems, storeSections, onChange }: {
  line: LineDraft
  path: string
  number: number
  problems: Problems
  storeSections: StoreSection[]
  onChange: (line: LineDraft) => void
}) {
  const quantity = line.quantity
  const setQuantity = (change: Partial<LineDraft['quantity']>) => onChange({ ...line, quantity: { ...quantity, ...change } })
  const quantityProblems = messagesFor(problems, `${path}.quantity`, true)
  return (
    <div className="edit-line" role="group" aria-label={`Ingredient ${number}`}>
      <TextField label="As written" value={line.sourceText} onChange={(sourceText) => onChange({ ...line, sourceText })}
        problems={messagesFor(problems, `${path}.sourceText`)} placeholder="2 cups chicken broth" />
      <ItemPicker choice={line.item} storeSections={storeSections} path={path} problems={problems}
        onChange={(item) => onChange({ ...line, item })} />
      <div className="edit-quantity" role="group" aria-label="Quantity">
        <Compact label="Kind">{(id) => (
          <select id={id} value={quantity.kind} aria-invalid={quantity.kind === 'unspecified' && quantityProblems.length > 0}
            onChange={(event) => setQuantity({ kind: event.target.value as LineDraft['quantity']['kind'] })}>
            <option value="exact">Exact</option>
            <option value="range">Range</option>
            <option value="unspecified">No amount</option>
          </select>
        )}</Compact>
        {quantity.kind !== 'unspecified' && (
          <>
            <Compact label={quantity.kind === 'range' ? 'From' : 'Amount'} narrow>{(id) => (
              <input id={id} value={quantity.amount} inputMode="decimal" placeholder="1 1/2" aria-invalid={quantityProblems.length > 0}
                onChange={(event) => setQuantity({ amount: event.target.value })} />
            )}</Compact>
            {quantity.kind === 'range' && (
              <Compact label="To" narrow>{(id) => (
                <input id={id} value={quantity.maximum} inputMode="decimal" onChange={(event) => setQuantity({ maximum: event.target.value })} />
              )}</Compact>
            )}
            <Compact label="Unit">{(id) => (
              <select id={id} value={quantity.measure} onChange={(event) => setQuantity({ measure: event.target.value })}>
                {units.map((unit) => <option key={unit.key} value={unit.key}>{unit.label}</option>)}
                {!units.some((unit) => unit.key === quantity.measure) && quantity.measure !== 'package' && <option value={quantity.measure}>{quantity.measure}</option>}
                <option value="package">package…</option>
              </select>
            )}</Compact>
            {quantity.measure === 'package' && (
              <>
                <Compact label="Package">{(id) => (
                  <input id={id} value={quantity.packageType} placeholder="can" onChange={(event) => setQuantity({ packageType: event.target.value })} />
                )}</Compact>
                <Compact label="Size" narrow>{(id) => (
                  <input id={id} value={quantity.packageSize} inputMode="decimal" placeholder="15" onChange={(event) => setQuantity({ packageSize: event.target.value })} />
                )}</Compact>
                <Compact label="Size unit">{(id) => (
                  <select id={id} value={quantity.packageUnit} onChange={(event) => setQuantity({ packageUnit: event.target.value })}>
                    <option value="">none</option>
                    {units.map((unit) => <option key={unit.key} value={unit.key}>{unit.label}</option>)}
                  </select>
                )}</Compact>
              </>
            )}
          </>
        )}
      </div>
      <FieldProblems messages={quantityProblems} />
      <div className="edit-line-extras">
        <TextField label="Preparation" value={line.preparation} onChange={(preparation) => onChange({ ...line, preparation })}
          problems={messagesFor(problems, `${path}.preparation`)} placeholder="diced" />
        <TextField label="Note" value={line.displayNote} onChange={(displayNote) => onChange({ ...line, displayNote })}
          problems={messagesFor(problems, `${path}.displayNote`)} />
        <label className="edit-check"><input type="checkbox" checked={line.optional} onChange={(event) => onChange({ ...line, optional: event.target.checked })} /> Optional</label>
        <label className="edit-check"><input type="checkbox" checked={line.onGroceryList} onChange={(event) => onChange({ ...line, onGroceryList: event.target.checked })} /> On the grocery list</label>
      </div>
    </div>
  )
}

// ItemPicker chooses a line's grocery item: one the household already has, found with the same
// search the grocery list uses, or a new household item with a store section and shopping mode.
function ItemPicker({ choice, storeSections, path, problems, onChange }: {
  choice: ItemChoice
  storeSections: StoreSection[]
  path: string
  problems: Problems
  onChange: (choice: ItemChoice) => void
}) {
  const inputId = useId()
  const listId = useId()
  const [text, setText] = useState('')
  const [term, setTerm] = useState('')
  const [searching, setSearching] = useState(choice.kind === 'none')
  const [active, setActive] = useState(-1)
  useEffect(() => { const timer = setTimeout(() => setTerm(text.trim()), 150); return () => clearTimeout(timer) }, [text])
  const query = useQuery({ queryKey: ['grocery-items', term], queryFn: () => searchGroceryItems(term), enabled: term.length > 0 })
  const matches = term === text.trim() ? query.data?.items ?? [] : []
  const options: { label: string; detail: string; pick: () => void }[] = [
    ...matches.map((match: GroceryItemMatch) => ({
      label: match.name,
      detail: match.id ? match.section : `New item · ${match.section}`,
      pick: () => pick(match.id
        ? { kind: 'existing', id: match.id, name: match.name }
        : { kind: 'new', name: match.name, storeSectionId: match.sectionId, shoppingMode: 'measured' }),
    })),
    ...(text.trim() && !matches.some((match) => match.name.toLowerCase() === text.trim().toLowerCase())
      ? [{ label: `New item “${text.trim()}”`, detail: 'Choose its section', pick: () => pick({ kind: 'new', name: text.trim(), storeSectionId: null, shoppingMode: 'measured' }) }]
      : []),
  ]
  function pick(next: ItemChoice) {
    onChange(next)
    setSearching(false)
    setText('')
    setActive(-1)
  }
  const itemProblems = [
    ...messagesFor(problems, `${path}.groceryItemId`),
    ...messagesFor(problems, `${path}.newGroceryItem.name`),
  ]
  const invalid = itemProblems.length > 0

  if (searching || choice.kind === 'none') {
    const show = text.trim().length > 0 && options.length > 0
    return (
      <div className="edit-field item-search">
        <label htmlFor={inputId}>Grocery item</label>
        <input id={inputId} role="combobox" aria-autocomplete="list" aria-expanded={show} aria-controls={listId}
          aria-activedescendant={show && active >= 0 ? `${listId}-${active}` : undefined} aria-invalid={invalid}
          value={text} placeholder="Search your items…" autoComplete="off"
          onChange={(event) => { setText(event.target.value); setActive(-1) }}
          onKeyDown={(event) => {
            if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
              event.preventDefault()
              setActive((previous) => (options.length ? (previous + (event.key === 'ArrowDown' ? 1 : -1) + options.length) % options.length : -1))
            }
            if (event.key === 'Enter' && show) {
              event.preventDefault()
              options[active >= 0 ? active : 0]?.pick()
            }
            if (event.key === 'Escape' && choice.kind !== 'none') setSearching(false)
          }} />
        {show && (
          <ul className="grocery-suggestions" id={listId} role="listbox" aria-label="Grocery items">
            {options.map((option, index) => (
              <li key={`${option.label}-${index}`} id={`${listId}-${index}`} role="option" aria-selected={active === index}
                onMouseDown={(event) => event.preventDefault()} onClick={option.pick}>
                <strong>{option.label}</strong><span>{option.detail}</span>
              </li>
            ))}
          </ul>
        )}
        <FieldProblems messages={itemProblems} />
      </div>
    )
  }

  return (
    <div className="edit-field item-choice">
      <span className="edit-label">Grocery item</span>
      <div className="item-choice-row">
        <strong>{choice.name}</strong>
        {choice.kind === 'new' && <span className="optional-label">New item</span>}
        <button className="button quiet-button" type="button" aria-label={`Change grocery item ${choice.name}`} onClick={() => setSearching(true)}>Change</button>
      </div>
      {choice.kind === 'new' && (
        <div className="new-item-fields">
          <Compact label="Store section">{(id) => (
            <select id={id} value={choice.storeSectionId ?? ''} aria-invalid={messagesFor(problems, `${path}.newGroceryItem.storeSectionId`).length > 0}
              onChange={(event) => onChange({ ...choice, storeSectionId: event.target.value ? Number(event.target.value) : null })}>
              <option value="">Choose…</option>
              {storeSections.map((section) => <option key={section.id} value={section.id}>{section.name}</option>)}
            </select>
          )}</Compact>
          <Compact label="Shopping mode">{(id) => (
            <select id={id} value={choice.shoppingMode} onChange={(event) => onChange({ ...choice, shoppingMode: event.target.value as typeof choice.shoppingMode })}>
              {shoppingModes.map((mode) => <option key={mode.value} value={mode.value}>{mode.label}</option>)}
            </select>
          )}</Compact>
          <FieldProblems messages={messagesFor(problems, `${path}.newGroceryItem`, true)} />
        </div>
      )}
      {choice.kind === 'existing' && <FieldProblems messages={itemProblems} />}
    </div>
  )
}

function InstructionSectionEditor({ section, path, index, count, problems, onChange, onRestructure, onMove, onRemove }: {
  section: InstructionSectionDraft
  path: string
  index: number
  count: number
  problems: Problems
  onChange: (section: InstructionSectionDraft) => void
  onRestructure: (section: InstructionSectionDraft) => void
  onMove: (offset: number) => void
  onRemove: () => void
}) {
  return (
    <fieldset className="edit-section">
      <legend className="sr-only">Instruction section {index + 1}</legend>
      <div className="edit-section-head">
        <TextField label="Section name" value={section.name} onChange={(name) => onChange({ ...section, name })} problems={messagesFor(problems, path)} />
        <RowControls noun="instruction section" index={index} count={count} onMove={onMove} onRemove={onRemove} />
      </div>
      <ol className="edit-lines">
        {section.steps.map((step, stepIndex) => (
          <li key={step.key}>
            <TextArea label={`Step ${stepIndex + 1}`} value={step.instruction}
              problems={messagesFor(problems, `${path}.steps[${stepIndex}]`, true)}
              onChange={(instruction) => onChange({ ...section, steps: section.steps.map((existing, position) => (position === stepIndex ? { ...existing, instruction } : existing)) })} />
            <RowControls noun="step" index={stepIndex} count={section.steps.length}
              onMove={(offset) => onRestructure({ ...section, steps: move(section.steps, stepIndex, offset) })}
              onRemove={() => onRestructure({ ...section, steps: section.steps.filter((_, position) => position !== stepIndex) })} />
          </li>
        ))}
      </ol>
      <button className="button quiet-button" type="button" onClick={() => onRestructure({ ...section, steps: [...section.steps, { key: draftKey(), instruction: '' }] })}>
        <PlusIcon /> Add step
      </button>
    </fieldset>
  )
}

function RowControls({ noun, index, count, onMove, onRemove }: {
  noun: string
  index: number
  count: number
  onMove: (offset: number) => void
  onRemove: () => void
}) {
  const position = `${noun} ${index + 1}`
  return (
    <div className="row-controls">
      <button className="icon-button quiet" type="button" aria-label={`Move ${position} up`} disabled={index === 0} onClick={() => onMove(-1)}>↑</button>
      <button className="icon-button quiet" type="button" aria-label={`Move ${position} down`} disabled={index === count - 1} onClick={() => onMove(1)}>↓</button>
      <button className="icon-button quiet danger" type="button" aria-label={`Remove ${position}`} onClick={onRemove}><CloseIcon /></button>
    </div>
  )
}

function TextField({ label, value, onChange, problems, placeholder, type = 'text' }: {
  label: string
  value: string
  onChange: (value: string) => void
  problems: string[]
  placeholder?: string
  type?: string
}) {
  const id = useId()
  return (
    <div className="edit-field">
      <label htmlFor={id}>{label}</label>
      <input id={id} type={type} value={value} placeholder={placeholder} aria-invalid={problems.length > 0}
        aria-describedby={problems.length > 0 ? `${id}-problems` : undefined} onChange={(event) => onChange(event.target.value)} />
      <FieldProblems id={`${id}-problems`} messages={problems} />
    </div>
  )
}

function TextArea({ label, value, onChange, problems }: { label: string; value: string; onChange: (value: string) => void; problems: string[] }) {
  const id = useId()
  return (
    <div className="edit-field">
      <label htmlFor={id}>{label}</label>
      <textarea id={id} value={value} rows={3} aria-invalid={problems.length > 0}
        aria-describedby={problems.length > 0 ? `${id}-problems` : undefined} onChange={(event) => onChange(event.target.value)} />
      <FieldProblems id={`${id}-problems`} messages={problems} />
    </div>
  )
}

function MinutesField({ label, minimum, maximum, onChange, problems }: {
  label: string
  minimum: string
  maximum: string
  onChange: (minimum: string, maximum: string) => void
  problems: string[]
}) {
  const id = useId()
  return (
    <div className="edit-field" role="group" aria-labelledby={id}>
      <span className="edit-label" id={id}>{label}</span>
      <div className="minutes-range">
        <input type="number" min={0} inputMode="numeric" aria-label={`${label}, from`} value={minimum} aria-invalid={problems.length > 0}
          onChange={(event) => onChange(event.target.value, maximum)} />
        <span aria-hidden="true">–</span>
        <input type="number" min={0} inputMode="numeric" aria-label={`${label}, to`} value={maximum} aria-invalid={problems.length > 0}
          onChange={(event) => onChange(minimum, event.target.value)} />
      </div>
      <FieldProblems messages={problems} />
    </div>
  )
}

// Compact is a short labelled control in a row of controls. The label points at the control by
// id rather than wrapping it, so a select's options never become part of its name.
function Compact({ label, narrow = false, children }: { label: string; narrow?: boolean; children: (id: string) => ReactNode }) {
  const id = useId()
  return (
    <div className={`edit-field compact${narrow ? ' narrow' : ''}`}>
      <label htmlFor={id}>{label}</label>
      {children(id)}
    </div>
  )
}

function FieldProblems({ id, messages }: { id?: string; messages: string[] }): ReactNode {
  if (messages.length === 0) return null
  return <p className="field-problem" id={id}>{messages.join(' ')}</p>
}
