import { expect, test, type Page } from '@playwright/test'

type Recipe = Record<string, unknown> & { ingredientSections: { name: string; ingredients: Record<string, unknown>[] }[] }

// The mock API keeps one recipe and derives the current grocery list from it, as the server's
// rebuild does, so a saved edit shows on Groceries.
async function setup(page: Page) {
  const quantity = (amount: string, unit: string) => ({ kind: 'exact', amount, maximum: null, unit, package: null })
  let recipe: Recipe = {
    id: 7, key: 'pancakes', name: 'Pancakes', imageUrl: null, yield: 'Serves 4',
    handsOn: { minimumMinutes: 15, maximumMinutes: 20 }, unattended: { minimumMinutes: null, maximumMinutes: null },
    verifiedAt: '2026-08-17T00:00:00Z', verifiedBy: null,
    sources: [{ relationship: 'source', attribution: 'Family', url: null, primary: true }],
    ingredientSections: [{
      name: 'Batter',
      ingredients: [
        { id: 1, sourceText: '2 cups flour', preparation: null, optional: false, displayNote: null, shoppingItem: 'Flour', groceryItemId: 11, quantity: quantity('2', 'cup'), onGroceryList: true },
        { id: 2, sourceText: '1 cup milk', preparation: null, optional: false, displayNote: null, shoppingItem: 'Milk', groceryItemId: 12, quantity: quantity('1', 'cup'), onGroceryList: true },
      ],
    }],
    instructionSections: [{ name: 'Instructions', steps: [{ id: 1, instruction: 'Whisk and cook.' }] }],
  }
  const edits: unknown[] = []
  const groceries = () => ({
    startsOn: '2026-11-08',
    lines: recipe.ingredientSections[0].ingredients.map((ingredient, index) => {
      const amount = ingredient.quantity as { amount: string; unit: string }
      const text = `${amount.amount} ${amount.amount === '1' ? amount.unit : `${amount.unit}s`}`
      return {
        id: index + 1, sectionId: 1, section: 'Pantry', name: ingredient.shoppingItem, quantity: text, generatedQuantity: text,
        origin: 'generated', optional: false, removed: false, completed: false, hasContributions: true, recipeNames: ['Pancakes'],
      }
    }),
  })
  await page.route('**/api/v2/**', async (route) => {
    const request = route.request()
    const path = new URL(request.url()).pathname
    if (path.endsWith('/session')) {
      return route.fulfill({ json: { user: { displayName: 'Cook' }, households: [{ householdID: 'test', householdName: 'Coward', role: 'owner' }] } })
    }
    if (path.endsWith('/recipes/7') && request.method() === 'PUT') {
      const edit = request.postDataJSON()
      edits.push(edit)
      if (!edit.name.trim()) {
        return route.fulfill({ status: 422, json: { error: { code: 'invalid_recipe', message: 'The recipe was not saved.' }, fields: [{ path: 'name', message: 'name is required' }] } })
      }
      recipe = {
        ...recipe, name: edit.name, verifiedBy: 'Cook', verifiedAt: '2026-11-14T12:00:00.000Z',
        ingredientSections: edit.ingredientSections.map((section: { name: string; ingredients: Record<string, unknown>[] }, sectionIndex: number) => ({
          name: section.name,
          ingredients: section.ingredients.map((line, index) => ({
            ...recipe.ingredientSections[sectionIndex].ingredients[index], ...line, id: 10 + index,
          })),
        })),
      }
      return route.fulfill({ json: recipe })
    }
    if (path.endsWith('/recipes/7')) return route.fulfill({ json: recipe })
    if (path.endsWith('/recipes')) return route.fulfill({ json: { recipes: [recipe] } })
    if (path.endsWith('/store-sections')) return route.fulfill({ json: { sections: [{ id: 1, name: 'Pantry', key: 'pantry' }] } })
    if (path.endsWith('/grocery-items')) return route.fulfill({ json: { items: [] } })
    if (path.endsWith('/week/current')) return route.fulfill({ json: { id: 1, startsOn: '2026-11-08', endsOn: null, recipes: [] } })
    if (path.endsWith('/week/current/groceries')) return route.fulfill({ json: groceries() })
    return route.fulfill({ json: {} })
  })
  return { edits }
}

test('editing a quantity saves it and shows it on the recipe and on Groceries', async ({ page }) => {
  const { edits } = await setup(page)
  await page.goto('/recipes/7')
  await page.getByRole('link', { name: 'Edit' }).click()
  await expect(page.getByRole('heading', { name: 'Pancakes' })).toBeVisible()

  const flour = page.getByRole('group', { name: 'Ingredient 1' })
  await flour.getByLabel('As written').fill('3 cups flour')
  await flour.getByRole('textbox', { name: 'Amount', exact: true }).fill('3')
  await page.getByRole('button', { name: 'Save recipe' }).click()

  await expect(page).toHaveURL(/\/recipes\/7$/)
  await expect(page.getByText('3 cups flour')).toBeVisible()
  await expect(page.getByText(/Last saved by Cook/)).toBeVisible()
  expect(edits).toHaveLength(1)
  const saved = edits[0] as { ingredientSections: { ingredients: { quantity: { amount: string }; groceryItemId: number }[] }[] }
  expect(saved.ingredientSections[0].ingredients[0]).toMatchObject({ groceryItemId: 11, quantity: { amount: '3' } })

  await page.goto('/groceries')
  await expect(page.locator('.grocery-row').filter({ hasText: 'Flour' })).toContainText('3 cups')
})

test('a rejected save keeps the form and marks the field', async ({ page }) => {
  await setup(page)
  await page.goto('/recipes/7/edit')
  const name = page.getByLabel('Name', { exact: true })
  await name.fill('')
  await page.getByRole('button', { name: 'Save recipe' }).click()
  await expect(page.getByRole('alert')).toContainText('Fix the marked fields')
  await expect(page.getByText('name is required')).toBeVisible()
  await expect(name).toHaveAttribute('aria-invalid', 'true')
  await expect(name).toBeFocused()
  await expect(page).toHaveURL(/\/recipes\/7\/edit$/)
})
