import { expect, test, type Page } from '@playwright/test'

async function setup(page: Page, options: { saveDelayMs?: number } = {}) {
  let settings = { timezone: 'UTC', defaultRecipeCount: 5, avoidRecentPlans: 0 }
  let sections = [{ id: 1, name: 'Produce', key: 'produce' }, { id: 2, name: 'Dairy', key: 'dairy' }, { id: 3, name: 'Baking', key: 'baking' }, { id: 4, name: 'Other', key: 'other' }]
  const base = { quantity: null as string | null, generatedQuantity: null as string | null, origin: 'manual', optional: false, removed: false, completed: false, hasContributions: false, recipeNames: [] as string[] }
  const lines = [
    { ...base, id: 1, name: 'Apples', section: 'Produce', sectionId: 1 },
    { ...base, id: 2, name: 'Bananas', section: 'Produce', sectionId: 1 },
    { ...base, id: 3, name: 'Cheese', section: 'Dairy', sectionId: 2 },
    { ...base, id: 4, name: 'Butter', section: 'Dairy', sectionId: 2, origin: 'recipe', quantity: '2 tbsp', generatedQuantity: '2 tbsp', hasContributions: true, recipeNames: ['Pancakes'] },
  ]
  await page.route('**/api/v2/**', async route => {
    const request = route.request(); const path = new URL(request.url()).pathname
    let body: unknown
    if (path.endsWith('/session')) body = { user: { displayName: 'Shopper' }, households: [{ householdID: 'test', householdName: 'Coward', role: 'owner' }] }
    else if (path.endsWith('/week/current')) body = { id: 1, startsOn: '2026-11-08', endsOn: null, recipes: [] }
    else if (path.endsWith('/settings')) {
      if (request.method() === 'PATCH') settings = { ...settings, ...request.postDataJSON() }
      body = settings
    }
    else if (path.endsWith('/grocery-items')) {
      const q = new URL(request.url()).searchParams.get('q') ?? ''
      body = { items: q.startsWith('cot') ? [{ name: 'Cottage cheese', sectionId: 2, section: 'Dairy' }] : [{ id: 10, name: 'Milk', sectionId: 2, section: 'Dairy' }] }
    }
    else if (path.endsWith('/store-sections')) {
      if (request.method() === 'PUT') {
        const input = request.postDataJSON()
        sections = input.reset ? [...sections].sort((a, b) => a.id - b.id) : input.sectionIds.map((id: number) => sections.find(section => section.id === id))
      }
      body = { sections }
    } else if (path.endsWith('/contributions')) {
      body = { contributions: [{ recipeId: 7, recipeName: 'Pancakes', sourceText: '2 tbsp butter, melted', optional: false }] }
    } else if (path.includes('/groceries')) {
      const line = lines.find(line => line.id === Number(path.split('/').at(-1)))
      if (request.method() === 'POST') {
        const input = request.postDataJSON()
        const known = input.groceryItemId === 10 || input.name.toLowerCase().trim() === 'milk'
        const sectionId = known ? 2 : input.storeSectionId ?? 4
        lines.push({ ...base, id: lines.length + 1, name: known ? 'Milk' : input.name, sectionId, section: sections.find(section => section.id === sectionId)!.name })
      }
      if (request.method() === 'DELETE' && line) line.removed = true
      if (request.method() === 'PATCH' && line) {
        const input = request.postDataJSON()
        if ('completed' in input) line.completed = input.completed
        if ('removed' in input) line.removed = input.removed
        if ('overrideText' in input) line.quantity = input.overrideText || line.generatedQuantity
        if (input.storeSectionId) { line.sectionId = input.storeSectionId; line.section = sections.find(section => section.id === input.storeSectionId)!.name }
        if (options.saveDelayMs) await new Promise(resolve => setTimeout(resolve, options.saveDelayMs))
      }
      body = { startsOn: '2026-11-08', lines: [...lines].sort((a, b) => sections.findIndex(s => s.id === a.sectionId) - sections.findIndex(s => s.id === b.sectionId)) }
    } else { body = {} }
    await route.fulfill({ json: body })
  })
  await page.goto('/groceries')
  await expect(page.getByRole('heading', { name: 'Groceries', exact: true })).toBeVisible()
}

const section = (page: Page, name: string) => page.locator('.grocery-section').filter({ has: page.getByRole('heading', { name, exact: true }) })

test('checked items settle into a collapsed Done group and can be unchecked there', async ({ page }) => {
  await setup(page)
  const produce = section(page, 'Produce').locator('.grocery-row')
  // The whole row is the check target, not just the box.
  await page.getByText('Apples', { exact: true }).click()
  await expect(page.getByRole('button', { name: 'Mark incomplete: Apples' })).toBeVisible()
  await expect(produce).toHaveCount(2)
  await expect(produce).toHaveCount(1, { timeout: 5000 })
  const done = page.getByRole('button', { name: /Done/ })
  await expect(done).toHaveAttribute('aria-expanded', 'false')
  await expect(done).toContainText('1')
  await expect(page.getByRole('button', { name: 'Mark incomplete: Apples' })).toHaveCount(0)
  await done.click()
  await page.getByRole('button', { name: 'Mark incomplete: Apples' }).click()
  await expect(produce).toHaveCount(2)
  await expect(produce.first()).toContainText('Apples')
  await expect(done).toHaveCount(0)
  await expect(page.getByRole('button', { name: /^Show done|^Hide done/ })).toHaveCount(0)
})

test('quick taps are never blocked by saves in flight', async ({ page }) => {
  await setup(page, { saveDelayMs: 700 })
  for (const name of ['Apples', 'Bananas', 'Cheese']) await page.getByRole('button', { name: `Mark complete: ${name}`, exact: true }).click()
  for (const name of ['Apples', 'Bananas', 'Cheese']) await expect(page.getByRole('button', { name: `Mark incomplete: ${name}`, exact: true })).toBeVisible()
  await expect(section(page, 'Produce')).toHaveCount(0, { timeout: 6000 })
  await expect(page.getByRole('button', { name: /Done/ })).toContainText('3')
  await expect(page.locator('.grocery-progress')).toContainText('1 left')
})

test('details sheet adjusts quantity, shows recipes, and remove can be undone', async ({ page }) => {
  await setup(page)
  await page.getByRole('button', { name: 'Details for Butter' }).click()
  const sheet = page.getByRole('dialog', { name: 'Butter' })
  await expect(sheet).toContainText('Pancakes')
  await expect(sheet).toContainText('2 tbsp butter, melted')
  await sheet.getByLabel('Quantity for this plan').fill('1 stick')
  await sheet.getByRole('button', { name: 'Save' }).click()
  await expect(sheet).toHaveCount(0)
  await expect(section(page, 'Dairy').locator('.grocery-quantity.adjusted')).toHaveText('1 stick')

  await page.getByRole('button', { name: 'Details for Cheese' }).click()
  await page.getByRole('button', { name: 'Remove from list' }).click()
  await expect(page.getByRole('button', { name: 'Mark complete: Cheese' })).toHaveCount(0)
  await expect(page.getByRole('status')).toContainText('Removed Cheese')
  await page.screenshot({ path: `test-results/groceries-${test.info().project.name}.png`, fullPage: true })
  await page.getByRole('button', { name: 'Undo', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Mark complete: Cheese' })).toBeVisible()

  await page.getByRole('button', { name: 'Details for Bananas' }).click()
  await page.getByRole('button', { name: 'Remove from list' }).click()
  await expect(page.getByRole('status')).toHaveCount(0, { timeout: 7000 })
  await expect(page.getByRole('button', { name: 'Mark complete: Bananas' })).toHaveCount(0)
})

test('manual suggestions, exact matching, and section corrections', async ({ page }) => {
  await setup(page)
  const input = page.getByRole('combobox', { name: 'Add grocery item' })
  await expect(page.getByRole('button', { name: 'Add', exact: true })).toHaveCount(0)
  await input.fill('mil')
  await expect(page.getByRole('option', { name: 'Milk Dairy' })).toBeVisible()
  await input.press('ArrowDown'); await input.press('Enter')
  await expect(section(page, 'Dairy')).toContainText('Milk')
  await input.fill('milk'); await input.press('Enter')
  await expect(page.locator('.grocery-name').filter({ hasText: /^Milk$/ })).toHaveCount(2)
  await input.fill('Birthday candles'); await input.press('Enter')
  const placement = page.getByRole('group', { name: 'Choose a section for Birthday candles' })
  await expect(placement).toContainText('Where will you find Birthday candles?')
  await placement.getByRole('button', { name: 'Baking', exact: true }).click()
  await expect(placement).toHaveCount(0)
  await expect(section(page, 'Baking')).toContainText('Birthday candles')
  await page.getByRole('button', { name: 'Details for Birthday candles' }).click()
  await page.getByRole('dialog', { name: 'Birthday candles' }).getByRole('button', { name: 'Other', exact: true }).click()
  await expect(section(page, 'Other')).toContainText('Birthday candles')
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})

test('a name placed before is suggested and added to its remembered section', async ({ page }) => {
  await setup(page)
  const input = page.getByRole('combobox', { name: 'Add grocery item' })
  await input.fill('cot')
  await expect(page.getByRole('option', { name: 'Cottage cheese Dairy' })).toBeVisible()
  await page.getByRole('option', { name: 'Cottage cheese Dairy' }).click()
  await expect(section(page, 'Dairy')).toContainText('Cottage cheese')
  await expect(page.getByRole('group', { name: /Choose a section/ })).toHaveCount(0)
})

test('section order saves, survives reload and resets', async ({ page }) => {
  await setup(page)
  await page.getByRole('button', { name: 'Reorder sections', exact: true }).click()
  await page.getByRole('button', { name: 'Move Dairy up' }).focus()
  await page.keyboard.press('Enter')
  await expect(page.locator('.section-order-list li').first()).toContainText('Dairy')
  await page.getByRole('button', { name: 'Save order', exact: true }).click()
  await expect(page.locator('.grocery-section h2').first()).toHaveText('Dairy')
  await page.reload()
  await expect(page.locator('.grocery-section h2').first()).toHaveText('Dairy')
  await page.getByRole('button', { name: 'Reorder sections', exact: true }).click()
  await page.getByRole('button', { name: 'Reset to default', exact: true }).click()
  await expect(page.locator('.grocery-section h2').first()).toHaveText('Produce')
})

test('reduced motion keeps settling movement non-animated', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await setup(page)
  await page.getByRole('button', { name: 'Mark complete: Apples', exact: true }).click()
  const rows = section(page, 'Produce').locator('.grocery-row')
  await expect(rows).toHaveCount(1, { timeout: 5000 })
  expect(await rows.first().evaluate(row => row.getAnimations().length)).toBe(0)
})

test('desktop drag handle reorders a section', async ({ page }, info) => {
  test.skip(info.project.name !== 'desktop', 'Touch devices use the accessible arrow controls.')
  await setup(page)
  await page.getByRole('button', { name: 'Reorder sections', exact: true }).click()
  const rows = page.locator('.section-order-list li')
  await rows.nth(1).locator('.drag-handle').dragTo(rows.first())
  await expect(rows.first()).toContainText('Dairy')
})
