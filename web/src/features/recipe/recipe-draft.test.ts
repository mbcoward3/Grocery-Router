import { describe, expect, it } from 'vitest'
import type { RecipeDetail } from '../../api/client'
import { draftFromRecipe, editFromDraft, groupProblems, messagesFor, move } from './recipe-draft'

const recipe: RecipeDetail = {
  id: 7, key: 'chili', name: 'Chili', imageUrl: null, yield: 'Serves 6',
  handsOn: { minimumMinutes: 20, maximumMinutes: 25 }, unattended: { minimumMinutes: null, maximumMinutes: null },
  verifiedAt: '2026-08-17T00:00:00Z', verifiedBy: null,
  sources: [{ relationship: 'source', attribution: 'Family', url: null, primary: true }],
  ingredientSections: [{
    name: 'Chili',
    ingredients: [
      {
        id: 1, sourceText: '1 1/2 lb ground beef', preparation: null, optional: false, displayNote: null, shoppingItem: 'Ground beef',
        groceryItemId: 4, quantity: { kind: 'exact', amount: '1 1/2', maximum: null, unit: 'lb', package: null }, onGroceryList: true,
      },
      {
        id: 2, sourceText: '2 (15 oz) cans beans', preparation: 'drained', optional: true, displayNote: 'Any beans', shoppingItem: 'Beans',
        groceryItemId: 5, quantity: { kind: 'exact', amount: '2', maximum: null, unit: null, package: { type: 'can', size: '15', unit: 'oz' } }, onGroceryList: true,
      },
      {
        id: 3, sourceText: 'Water', preparation: null, optional: false, displayNote: null, shoppingItem: 'Water',
        groceryItemId: 6, quantity: { kind: 'unspecified', amount: null, maximum: null, unit: null, package: null }, onGroceryList: false,
      },
    ],
  }],
  instructionSections: [{ name: 'Instructions', steps: [{ id: 1, instruction: 'Brown the beef.' }, { id: 2, instruction: 'Simmer.' }] }],
}

describe('recipe draft', () => {
  it('saves an untouched recipe as the content it loaded', () => {
    expect(editFromDraft(draftFromRecipe(recipe))).toEqual({
      name: 'Chili', imageUrl: null, yield: 'Serves 6',
      handsOn: { minimumMinutes: 20, maximumMinutes: 25 }, unattended: { minimumMinutes: null, maximumMinutes: null },
      ingredientSections: [{
        name: 'Chili',
        ingredients: [
          { sourceText: '1 1/2 lb ground beef', quantity: { kind: 'exact', amount: '1 1/2', maximum: null, unit: 'lb', package: null }, preparation: null, optional: false, displayNote: null, onGroceryList: true, groceryItemId: 4 },
          { sourceText: '2 (15 oz) cans beans', quantity: { kind: 'exact', amount: '2', maximum: null, unit: null, package: { type: 'can', size: '15', unit: 'oz' } }, preparation: 'drained', optional: true, displayNote: 'Any beans', onGroceryList: true, groceryItemId: 5 },
          { sourceText: 'Water', quantity: { kind: 'unspecified', amount: null, maximum: null, unit: null, package: null }, preparation: null, optional: false, displayNote: null, onGroceryList: false, groceryItemId: 6 },
        ],
      }],
      instructionSections: [{ name: 'Instructions', steps: [{ instruction: 'Brown the beef.' }, { instruction: 'Simmer.' }] }],
    })
  })

  it('sends a new grocery item instead of an ID', () => {
    const draft = draftFromRecipe(recipe)
    draft.ingredientSections[0]!.lines[0]!.item = { kind: 'new', name: 'Bison', storeSectionId: 3, shoppingMode: 'measured' }
    const line = editFromDraft(draft).ingredientSections[0]?.ingredients[0]
    expect(line?.groceryItemId).toBeUndefined()
    expect(line?.newGroceryItem).toEqual({ name: 'Bison', storeSectionId: 3, shoppingMode: 'measured' })
  })

  it('moves rows within bounds only', () => {
    expect(move(['a', 'b', 'c'], 0, 1)).toEqual(['b', 'a', 'c'])
    expect(move(['a', 'b', 'c'], 2, -2)).toEqual(['c', 'a', 'b'])
    expect(move(['a', 'b'], 0, -1)).toEqual(['a', 'b'])
  })

  it('finds field problems by path', () => {
    const problems = groupProblems([
      { path: 'ingredientSections[0].ingredients[1].quantity', message: 'unknown unit "pinch"' },
      { path: 'ingredientSections[0].ingredients[10].sourceText', message: 'source_text is required' },
      { path: 'name', message: 'name is required' },
    ])
    expect(messagesFor(problems, 'name')).toEqual(['name is required'])
    expect(messagesFor(problems, 'ingredientSections[0].ingredients[1]', true)).toEqual(['unknown unit "pinch"'])
    expect(messagesFor(problems, 'ingredientSections[0].ingredients[1].quantity')).toEqual(['unknown unit "pinch"'])
    expect(messagesFor(problems, 'ingredientSections')).toEqual([])
  })
})
