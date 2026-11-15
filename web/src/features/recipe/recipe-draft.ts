import type { RecipeDetail, RecipeEdit, RecipeIngredientEdit, RecipeQuantity, ShoppingMode } from '../../api/client'

// The unit vocabulary the server accepts (internal/ingest/units.go). Units have no runtime write
// path, so the editor lists them here instead of fetching them.
export const units: { key: string; label: string }[] = [
  { key: 'each', label: 'each' }, { key: 'clove', label: 'clove' }, { key: 'slice', label: 'slice' },
  { key: 'leaf', label: 'leaf' }, { key: 'bunch', label: 'bunch' }, { key: 'sprig', label: 'sprig' },
  { key: 'tsp', label: 'tsp' }, { key: 'tbsp', label: 'tbsp' }, { key: 'floz', label: 'fl oz' },
  { key: 'cup', label: 'cup' }, { key: 'pint', label: 'pint' }, { key: 'quart', label: 'quart' },
  { key: 'gallon', label: 'gallon' }, { key: 'ml', label: 'mL' }, { key: 'l', label: 'L' },
  { key: 'oz', label: 'oz' }, { key: 'lb', label: 'lb' }, { key: 'g', label: 'g' }, { key: 'kg', label: 'kg' },
]

export const shoppingModes: { value: ShoppingMode; label: string }[] = [
  { value: 'measured', label: 'By amount' },
  { value: 'counted', label: 'By count' },
  { value: 'presence-only', label: 'Just have some' },
]

export type ItemChoice =
  | { kind: 'existing'; id: number; name: string }
  | { kind: 'new'; name: string; storeSectionId: number | null; shoppingMode: ShoppingMode }
  | { kind: 'none' }

export interface QuantityDraft {
  kind: RecipeQuantity['kind']
  amount: string
  maximum: string
  /** A unit key, or 'package' when the amount counts packages. */
  measure: string
  packageType: string
  packageSize: string
  packageUnit: string
}

export interface LineDraft {
  key: string
  sourceText: string
  quantity: QuantityDraft
  preparation: string
  optional: boolean
  displayNote: string
  onGroceryList: boolean
  item: ItemChoice
}

export interface IngredientSectionDraft { key: string; name: string; lines: LineDraft[] }
export interface StepDraft { key: string; instruction: string }
export interface InstructionSectionDraft { key: string; name: string; steps: StepDraft[] }

export interface RecipeDraft {
  name: string
  imageUrl: string
  yield: string
  handsOnMin: string
  handsOnMax: string
  unattendedMin: string
  unattendedMax: string
  ingredientSections: IngredientSectionDraft[]
  instructionSections: InstructionSectionDraft[]
}

let nextKey = 0
/** A stable React key for a row the editor creates or loads. */
export function draftKey(): string {
  nextKey += 1
  return `draft-${nextKey}`
}

export function emptyQuantity(): QuantityDraft {
  return { kind: 'exact', amount: '', maximum: '', measure: 'each', packageType: '', packageSize: '', packageUnit: '' }
}

export function emptyLine(): LineDraft {
  return {
    key: draftKey(), sourceText: '', quantity: emptyQuantity(), preparation: '', optional: false,
    displayNote: '', onGroceryList: true, item: { kind: 'none' },
  }
}

export function draftFromRecipe(recipe: RecipeDetail): RecipeDraft {
  const minutes = (value: number | null) => (value === null ? '' : String(value))
  return {
    name: recipe.name,
    imageUrl: recipe.imageUrl ?? '',
    yield: recipe.yield ?? '',
    handsOnMin: minutes(recipe.handsOn.minimumMinutes),
    handsOnMax: minutes(recipe.handsOn.maximumMinutes),
    unattendedMin: minutes(recipe.unattended.minimumMinutes),
    unattendedMax: minutes(recipe.unattended.maximumMinutes),
    ingredientSections: recipe.ingredientSections.map((section) => ({
      key: draftKey(),
      name: section.name,
      lines: section.ingredients.map((ingredient) => ({
        key: draftKey(),
        sourceText: ingredient.sourceText,
        quantity: {
          kind: ingredient.quantity.kind,
          amount: ingredient.quantity.amount ?? '',
          maximum: ingredient.quantity.maximum ?? '',
          measure: ingredient.quantity.package ? 'package' : ingredient.quantity.unit ?? 'each',
          packageType: ingredient.quantity.package?.type ?? '',
          packageSize: ingredient.quantity.package?.size ?? '',
          packageUnit: ingredient.quantity.package?.unit ?? '',
        },
        preparation: ingredient.preparation ?? '',
        optional: ingredient.optional,
        displayNote: ingredient.displayNote ?? '',
        onGroceryList: ingredient.onGroceryList,
        item: ingredient.groceryItemId === null
          ? { kind: 'none' }
          : { kind: 'existing', id: ingredient.groceryItemId, name: ingredient.shoppingItem ?? '' },
      })),
    })),
    instructionSections: recipe.instructionSections.map((section) => ({
      key: draftKey(),
      name: section.name,
      steps: section.steps.map((step) => ({ key: draftKey(), instruction: step.instruction })),
    })),
  }
}

function optionalText(value: string): string | null {
  return value === '' ? null : value
}

function optionalMinutes(value: string): number | null {
  if (value.trim() === '') return null
  const parsed = Number(value)
  return Number.isInteger(parsed) ? parsed : null
}

function quantityEdit(quantity: QuantityDraft): RecipeQuantity {
  if (quantity.kind === 'unspecified') return { kind: 'unspecified', amount: null, maximum: null, unit: null, package: null }
  const packaged = quantity.measure === 'package'
  return {
    kind: quantity.kind,
    amount: quantity.amount.trim(),
    maximum: quantity.kind === 'range' ? quantity.maximum.trim() : null,
    unit: packaged ? null : quantity.measure,
    package: packaged
      ? { type: quantity.packageType, size: optionalText(quantity.packageSize.trim()), unit: optionalText(quantity.packageUnit) }
      : null,
  }
}

/** The PUT body that saves a draft. Validation is the server's: it names each rejected field. */
export function editFromDraft(draft: RecipeDraft): RecipeEdit {
  return {
    name: draft.name,
    imageUrl: optionalText(draft.imageUrl.trim()),
    yield: optionalText(draft.yield),
    handsOn: { minimumMinutes: optionalMinutes(draft.handsOnMin), maximumMinutes: optionalMinutes(draft.handsOnMax) },
    unattended: { minimumMinutes: optionalMinutes(draft.unattendedMin), maximumMinutes: optionalMinutes(draft.unattendedMax) },
    ingredientSections: draft.ingredientSections.map((section) => ({
      name: section.name,
      ingredients: section.lines.map((line) => {
        const edit: RecipeIngredientEdit = {
          sourceText: line.sourceText,
          quantity: quantityEdit(line.quantity),
          preparation: optionalText(line.preparation),
          optional: line.optional,
          displayNote: optionalText(line.displayNote),
          onGroceryList: line.onGroceryList,
        }
        if (line.item.kind === 'existing') edit.groceryItemId = line.item.id
        if (line.item.kind === 'new' && line.item.storeSectionId !== null) {
          edit.newGroceryItem = { name: line.item.name, storeSectionId: line.item.storeSectionId, shoppingMode: line.item.shoppingMode }
        }
        return edit
      }),
    })),
    instructionSections: draft.instructionSections.map((section) => ({
      name: section.name,
      steps: section.steps.map((step) => ({ instruction: step.instruction })),
    })),
  }
}

/** Moves the item at index by offset, returning a new array; out-of-range moves change nothing. */
export function move<T>(items: T[], index: number, offset: number): T[] {
  const target = index + offset
  if (target < 0 || target >= items.length) return items
  const result = items.slice()
  result.splice(target, 0, ...result.splice(index, 1))
  return result
}

/**
 * Messages for one field. A path names the field itself; with nested, problems inside it
 * (its sub-fields) count too.
 */
export function messagesFor(problems: Record<string, string[]>, path: string, nested = false): string[] {
  return Object.entries(problems).flatMap(([problemPath, messages]) =>
    problemPath === path || (nested && (problemPath.startsWith(`${path}.`) || problemPath.startsWith(`${path}[`))) ? messages : [])
}

export function groupProblems(fields: { path: string; message: string }[]): Record<string, string[]> {
  const grouped: Record<string, string[]> = {}
  for (const field of fields) (grouped[field.path] ??= []).push(field.message)
  return grouped
}
