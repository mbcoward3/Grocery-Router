import { expect, test, type Page } from '@playwright/test'

// Dates on the page are formatted for the viewer's locale.
test.use({ locale: 'en-US' })

type CurrentWeek = { id: number; startsOn: string; endsOn: string | null; recipes: unknown[] }

// By default, a household whose week of Nov 1 – Nov 7 has rolled over: no week is open and
// History holds it.
async function setup(page: Page, options: { pastWeeks?: boolean; current?: CurrentWeek } = {}) {
  const settings = { timezone: 'America/Chicago', defaultRecipeCount: 4, weekStartDay: 0, rolloverTime: '00:00', rolloverMode: 'automatic' }
  const generated: unknown[] = []
  const duration = { minimumMinutes: null, maximumMinutes: null }
  const recipes = Array.from({ length: 8 }, (_, index) => ({
    id: index + 1, key: `recipe-${index + 1}`, name: `Recipe ${index + 1}`,
    imageUrl: null, yield: null, handsOn: duration, unattended: duration,
  }))
  const closedWeek = { id: 7, startsOn: '2026-11-01', endsOn: '2026-11-07', recipeCount: 3, groceryCount: 12, completedCount: 10 }
  let current: CurrentWeek | null = options.current ?? null
  const pastWeeks = options.pastWeeks ?? true
  const noCurrentWeek = { error: { code: 'no_current_week', message: 'The current week has not been generated.' } }
  await page.route('**/api/v2/**', async route => {
    const request = route.request(); const path = new URL(request.url()).pathname
    if (path.endsWith('/session')) {
      return route.fulfill({ json: { user: { displayName: 'Planner' }, households: [{ householdID: 'test', householdName: 'Coward', role: 'owner' }] } })
    }
    if (path.endsWith('/settings')) return route.fulfill({ json: settings })
    if (path.endsWith('/week/current/generate')) {
      const input = request.postDataJSON()
      generated.push(input)
      const pool = recipes.slice(0, input.recipeCount).map((recipe, index) => ({ id: index + 1, position: index, recipe }))
      current = { id: 8, startsOn: '2026-11-08', endsOn: '2026-11-14', recipes: pool }
      return route.fulfill({ json: current })
    }
    if (path.endsWith('/week/current/groceries')) {
      return current ? route.fulfill({ json: { startsOn: current.startsOn, lines: [] } }) : route.fulfill({ status: 404, json: noCurrentWeek })
    }
    if (path.endsWith('/week/current')) return current ? route.fulfill({ json: current }) : route.fulfill({ status: 404, json: noCurrentWeek })
    if (path.endsWith('/recipes')) return route.fulfill({ json: { recipes } })
    if (path.endsWith('/weeks/history')) return route.fulfill({ json: { weeks: pastWeeks ? [closedWeek] : [] } })
    if (path.endsWith(`/weeks/history/${closedWeek.id}`)) {
      return route.fulfill({ json: {
        week: { id: closedWeek.id, startsOn: closedWeek.startsOn, endsOn: closedWeek.endsOn, recipes: [] },
        groceries: { startsOn: closedWeek.startsOn, lines: [] },
      } })
    }
    if (path.endsWith('/store-sections')) return route.fulfill({ json: { sections: [] } })
    return route.fulfill({ status: 404, json: { error: { code: 'not_found', message: `${path} is not mocked.` } } })
  })
  return { generated }
}

test('after a rollover the Week page leads straight into planning the next week', async ({ page }) => {
  const api = await setup(page)
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Plan next week' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Build this week’s recipe pool' })).toHaveCount(0)
  await expect(page.locator('.plan-next-week')).toContainText('Your week of Nov 1 – Nov 7, 2026 ended Saturday.')
  await expect(page.getByRole('link', { name: 'See it in History' })).toHaveAttribute('href', '/history/7')
  // The household's default count is ready; nothing is generated until Generate is pressed.
  await expect(page.locator('.count-field select')).toHaveValue('4')
  expect(api.generated).toEqual([])

  await page.getByRole('button', { name: 'Generate week' }).click()
  await expect(page.locator('.recipe-row')).toHaveCount(4)
  await expect(page.getByText('Planned · 4 meals')).toBeVisible()
  await expect(page.getByText('Nov 8 – Nov 14, 2026')).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Plan next week' })).toHaveCount(0)
  expect(api.generated).toEqual([{ recipeCount: 4 }])
})

test('the History link opens the week that ended', async ({ page }) => {
  await setup(page)
  await page.goto('/')
  await page.getByRole('link', { name: 'See it in History' }).click()
  await expect(page.getByRole('heading', { name: 'Nov 1 – Nov 7, 2026' })).toBeVisible()
})

test('after a rollover the Groceries page points to History and to planning', async ({ page }) => {
  await setup(page)
  await page.goto('/groceries')
  await expect(page.getByRole('heading', { name: 'Last week’s list is in History' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Generate a week first' })).toHaveCount(0)
  await expect(page.getByRole('link', { name: 'See last week’s list' })).toHaveAttribute('href', '/history/7')
  await page.getByRole('link', { name: 'Plan next week' }).click()
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByRole('heading', { name: 'Plan next week' })).toBeVisible()
  await expect(page.locator('.count-field select')).toHaveValue('4')
})

test('History shows each week from its start to its reported end', async ({ page }) => {
  await setup(page)
  await page.goto('/history')
  await expect(page.locator('.history-row').first()).toContainText('Nov 1 – Nov 7, 2026')
})

test('a household with no weeks at all keeps the first-time start', async ({ page }) => {
  await setup(page, { pastWeeks: false })
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Build this week’s recipe pool' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Plan next week' })).toHaveCount(0)
  await expect(page.locator('.count-field select')).toHaveValue('4')

  await page.goto('/groceries')
  await expect(page.getByRole('heading', { name: 'Generate a week first' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Plan next week' })).toHaveCount(0)
})

test('an open manual week shows where it began instead of an end date', async ({ page }) => {
  await setup(page, { current: { id: 9, startsOn: '2026-11-11', endsOn: null, recipes: [] } })
  await page.goto('/')
  await expect(page.getByText('Planned · 0 meals')).toBeVisible()
  await expect(page.locator('.page-heading p')).toHaveText('Since Nov 11, 2026')
})
