import { expect, test, type Page } from '@playwright/test'

// The device's zone, which "Use this device's time zone" must pick.
test.use({ timezoneId: 'America/Chicago' })

async function setup(page: Page) {
  let settings = { timezone: 'UTC', defaultRecipeCount: 5, weekStartDay: 0, rolloverTime: '00:00', rolloverMode: 'automatic' }
  const patches: unknown[] = []
  const generated: unknown[] = []
  const duration = { minimumMinutes: null, maximumMinutes: null }
  const recipes = Array.from({ length: 8 }, (_, index) => ({
    id: index + 1, key: `recipe-${index + 1}`, name: `Recipe ${index + 1}`,
    imageUrl: null, yield: null, handsOn: duration, unattended: duration,
  }))
  const noCurrentWeek = { error: { code: 'no_current_week', message: 'The current week has not been generated.' } }
  await page.route('**/api/v2/**', async route => {
    const request = route.request(); const path = new URL(request.url()).pathname
    if (path.endsWith('/session')) {
      return route.fulfill({ json: { user: { displayName: 'Planner' }, households: [{ householdID: 'test', householdName: 'Coward', role: 'owner' }] } })
    }
    if (path.endsWith('/settings')) {
      if (request.method() === 'PATCH') {
        const input = request.postDataJSON()
        patches.push(input)
        settings = { ...settings, ...input }
      }
      return route.fulfill({ json: settings })
    }
    if (path.endsWith('/week/current/generate')) {
      const input = request.postDataJSON()
      generated.push(input)
      const pool = recipes.slice(0, input.recipeCount).map((recipe, index) => ({ id: index + 1, position: index, recipe }))
      return route.fulfill({ json: { id: 1, startsOn: '2026-11-08', endsOn: '2026-11-14', recipes: pool } })
    }
    if (path.endsWith('/week/current') || path.endsWith('/week/current/groceries')) return route.fulfill({ status: 404, json: noCurrentWeek })
    if (path.endsWith('/recipes')) return route.fulfill({ json: { recipes } })
    if (path.endsWith('/weeks/history')) return route.fulfill({ json: { weeks: [] } })
    return route.fulfill({ status: 404, json: { error: { code: 'not_found', message: `${path} is not mocked.` } } })
  })
  return { patches, generated }
}

test('the recipe count saved in Settings starts the Week page', async ({ page }) => {
  const api = await setup(page)
  await page.goto('/settings')
  await expect(page.getByRole('heading', { name: 'Settings', exact: true })).toBeVisible()
  const zone = page.getByLabel('Time zone')
  const count = page.getByLabel('Recipes per week')
  await expect(zone).toHaveValue('UTC')
  await expect(count).toHaveValue('5')

  await page.getByRole('button', { name: /Use this device/ }).click()
  await expect(zone).toHaveValue('America/Chicago')
  await count.fill('3')
  await page.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByRole('status')).toHaveText('Settings saved.')
  expect(api.patches).toEqual([{ timezone: 'America/Chicago', defaultRecipeCount: 3 }])

  await page.getByRole('navigation', { name: 'Primary navigation' }).getByRole('link', { name: 'Week' }).click()
  await expect(page.getByRole('heading', { name: 'Build this week’s recipe pool' })).toBeVisible()
  await expect(page.locator('.count-field select')).toHaveValue('3')
  await page.getByRole('button', { name: 'Generate week' }).click()
  await expect(page.locator('.recipe-row')).toHaveCount(3)
  expect(api.generated).toEqual([{ recipeCount: 3 }])
})

test('an out-of-range recipe count is explained and not saved', async ({ page }) => {
  const api = await setup(page)
  await page.goto('/settings')
  const count = page.getByLabel('Recipes per week')
  await count.fill('0')
  await page.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByRole('alert')).toContainText('from 1 to 21')
  expect(api.patches).toEqual([])
})

test('rollover day, time, and mode are saved together', async ({ page }) => {
  const api = await setup(page)
  await page.goto('/settings')
  const day = page.getByLabel('Week starts on')
  const time = page.getByLabel('Rolls over at')
  await expect(day).toHaveValue('0')
  await expect(time).toHaveValue('00:00')
  await expect(page.getByRole('radio', { name: 'Automatic' })).toBeChecked()
  await expect(page.getByText('The week closes itself at that day and time.')).toBeVisible()
  await expect(page.getByText('The week stays open until someone presses “Start next week” on the Week page.')).toBeVisible()

  await day.selectOption({ label: 'Friday' })
  await time.fill('18:00')
  await page.getByRole('radio', { name: 'Manual' }).check()
  await page.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(page.getByRole('status')).toHaveText('Settings saved.')
  expect(api.patches).toEqual([{ weekStartDay: 5, rolloverTime: '18:00', rolloverMode: 'manual' }])
  await expect(day).toHaveValue('5')
  await expect(page.getByRole('radio', { name: 'Manual' })).toBeChecked()
})
