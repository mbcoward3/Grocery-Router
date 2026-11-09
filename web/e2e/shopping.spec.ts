import { expect, test, type Page } from '@playwright/test'

async function setup(page: Page) {
  let sections = [{ id: 1, name: 'Produce', key: 'produce' }, { id: 2, name: 'Dairy', key: 'dairy' }, { id: 3, name: 'Baking', key: 'baking' }, { id: 4, name: 'Other', key: 'other' }]
  const base = { quantity: null, generatedQuantity: null, origin: 'manual', optional: false, removed: false, completed: false, hasContributions: false, recipeNames: [] }
  const lines = [{ ...base, id: 1, name: 'Apples', section: 'Produce', sectionId: 1 }, { ...base, id: 2, name: 'Bananas', section: 'Produce', sectionId: 1 }, { ...base, id: 3, name: 'Cheese', section: 'Dairy', sectionId: 2 }]
  await page.route('**/api/v2/**', async route => {
    const request = route.request(); const path = new URL(request.url()).pathname
    let body: unknown
    if (path.endsWith('/session')) body = { user: { displayName: 'Shopper' }, households: [{ householdID: 'test', householdName: 'Coward', role: 'owner' }] }
    else if (path.endsWith('/week/current')) body = { id: 1, startsOn: '2026-11-08', recipes: [] }
    else if (path.endsWith('/grocery-items')) body = { items: [{ id: 10, name: 'Milk', sectionId: 2, section: 'Dairy' }] }
    else if (path.endsWith('/store-sections')) {
      if (request.method() === 'PUT') {
        const input = request.postDataJSON()
        sections = input.reset ? [...sections].sort((a, b) => a.id - b.id) : input.sectionIds.map((id: number) => sections.find(section => section.id === id))
      }
      body = { sections }
    } else if (path.includes('/groceries')) {
      if (request.method() === 'POST') {
        const input = request.postDataJSON()
        const known = input.groceryItemId === 10 || input.name.toLowerCase().trim() === 'milk'
        lines.push({ ...base, id: lines.length + 1, name: known ? 'Milk' : input.name, sectionId: known ? 2 : 4, section: known ? 'Dairy' : 'Other' })
      }
      if (request.method() === 'PATCH') {
        const line = lines.find(line => line.id === Number(path.split('/').at(-1)))!
        const input = request.postDataJSON()
        Object.assign(line, input)
        if (input.storeSectionId) { line.sectionId = input.storeSectionId; line.section = sections.find(section => section.id === input.storeSectionId)!.name }
      }
      body = { startsOn: '2026-11-08', lines: [...lines].sort((a, b) => sections.findIndex(s => s.id === a.sectionId) - sections.findIndex(s => s.id === b.sectionId)) }
    } else { body = {} }
    await route.fulfill({ json: body })
  })
  await page.goto('/groceries')
  await expect(page.getByRole('heading', { name: 'Groceries', exact: true })).toBeVisible()
}

test('check, delayed sink, undo, collapse, hide and restore preference', async ({ page }) => {
  await setup(page)
  const rows = page.locator('.grocery-section').filter({ has: page.getByRole('heading', { name: 'Produce', exact: true }) }).locator('.grocery-row')
  await page.getByRole('button', { name: 'Mark complete: Apples', exact: true }).click()
  await expect(rows.first()).toContainText('Apples')
  await expect(rows.last()).toContainText('Apples', { timeout: 5000 })
  await page.getByRole('button', { name: 'Undo', exact: true }).click()
  await expect(rows.first()).toContainText('Apples')
  await page.getByRole('button', { name: 'Mark complete: Cheese', exact: true }).click()
  const collapse = page.getByRole('button', { name: '1 done' })
  await expect(collapse).toHaveAttribute('aria-expanded', 'false')
  await collapse.click()
  await expect(page.getByRole('button', { name: 'Mark incomplete: Cheese' })).toBeVisible()
  await page.getByRole('button', { name: 'Hide done', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Mark incomplete: Cheese' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: '1 done · show' })).toBeVisible()
  await page.reload()
  await expect(page.getByRole('button', { name: '1 done · show' })).toBeVisible()
  await page.getByRole('button', { name: '1 done · show' }).click()
  await page.getByRole('button', { name: '1 done', exact: true }).click()
  await page.getByRole('button', { name: 'Mark incomplete: Cheese' }).click()
  await expect(page.getByRole('button', { name: 'Mark complete: Cheese' })).toBeVisible()
})

test('manual suggestions, exact matching, and section corrections', async ({ page }) => {
  await setup(page)
  const input = page.getByRole('combobox', { name: 'Add grocery item' })
  await input.fill('mil')
  await expect(page.getByRole('option', { name: 'Milk Dairy' })).toBeVisible()
  await input.press('ArrowDown'); await input.press('Enter')
  await expect(page.locator('.grocery-section').filter({ has: page.getByRole('heading', { name: 'Dairy', exact: true }) })).toContainText('Milk')
  await input.fill('milk'); await input.press('Enter')
  await expect(page.locator('.grocery-name').filter({ hasText: /^Milk$/ })).toHaveCount(2)
  await input.fill('Birthday candles'); await input.press('Enter')
  await expect(page.getByText('Where will you find Birthday candles?')).toBeVisible()
  await page.getByRole('button', { name: 'Baking', exact: true }).click()
  await expect(page.locator('.grocery-section').filter({ has: page.getByRole('heading', { name: 'Baking', exact: true }) })).toContainText('Birthday candles')
  await page.getByRole('button', { name: 'Change section for Birthday candles' }).click()
  await page.getByRole('button', { name: 'Other', exact: true }).click()
  await expect(page.locator('.grocery-section').filter({ has: page.getByRole('heading', { name: 'Other', exact: true }) })).toContainText('Birthday candles')
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
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
  await page.screenshot({ path: `test-results/shopping-${test.info().project.name}.png`, fullPage: true })
})


test('reduced motion keeps completion movement non-animated', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await setup(page)
  await page.getByRole('button', { name: 'Mark complete: Apples', exact: true }).click()
  const rows = page.locator('.grocery-section').first().locator('.grocery-row')
  await expect(rows.last()).toContainText('Apples')
  expect(await rows.last().evaluate(row => row.getAnimations().length)).toBe(0)
})

test('desktop drag handle reorders a section', async ({ page }, info) => {
  test.skip(info.project.name !== 'desktop', 'Touch devices use the accessible arrow controls.')
  await setup(page)
  await page.getByRole('button', { name: 'Reorder sections', exact: true }).click()
  const rows = page.locator('.section-order-list li')
  await rows.nth(1).locator('.drag-handle').dragTo(rows.first())
  await expect(rows.first()).toContainText('Dairy')
  await page.screenshot({ path: 'test-results/section-order-desktop.png', fullPage: true })
})
