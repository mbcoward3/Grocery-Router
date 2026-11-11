import { expect, test, type Page } from '@playwright/test'

// Dates on the page are formatted for the viewer's locale.
test.use({ locale: 'en-US' })

type CurrentWeek = { id: number; startsOn: string; endsOn: string | null; recipes: unknown[] }
type NextWeek = Omit<CurrentWeek, 'recipes'>
type Summary = { id: number; startsOn: string; endsOn: string; recipeCount: number; groceryCount: number; completedCount: number }

// By default, a household whose plan of Nov 1 – Nov 10 is closed: no plan is current and History
// holds it. Generate opens the plan `next` (starting Nov 12 by default); closing the current plan
// moves it to the top of History, ending on `closedEndsOn`. Reopening a past plan swaps it with
// the current one.
async function setup(page: Page, options: { pastWeeks?: boolean; current?: CurrentWeek; next?: NextWeek; closedEndsOn?: string } = {}) {
  const settings = { timezone: 'America/Chicago', defaultRecipeCount: 4, avoidRecentPlans: 0 }
  const generated: unknown[] = []
  const duration = { minimumMinutes: null, maximumMinutes: null }
  const recipes = Array.from({ length: 8 }, (_, index) => ({
    id: index + 1, key: `recipe-${index + 1}`, name: `Recipe ${index + 1}`,
    imageUrl: null, yield: null, handsOn: duration, unattended: duration,
  }))
  const closedWeek = { id: 7, startsOn: '2026-11-01', endsOn: '2026-11-10', recipeCount: 3, groceryCount: 12, completedCount: 10 }
  let current: CurrentWeek | null = options.current ?? null
  let history: Summary[] = (options.pastWeeks ?? true) ? [closedWeek] : []
  const next = options.next ?? { id: 8, startsOn: '2026-11-12', endsOn: null }
  let closes = 0
  const reopened: number[] = []
  const noCurrentWeek = { error: { code: 'no_current_week', message: 'There is no current plan.' } }
  const closeCurrent = () => {
    if (!current) return
    history.unshift({
      id: current.id, startsOn: current.startsOn, endsOn: options.closedEndsOn ?? '2026-11-20',
      recipeCount: current.recipes.length, groceryCount: 0, completedCount: 0,
    })
    current = null
  }
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
      current = { ...next, recipes: pool }
      return route.fulfill({ json: current })
    }
    if (path.endsWith('/week/current/close') && request.method() === 'POST') {
      closes++
      if (!current) return route.fulfill({ status: 404, json: noCurrentWeek })
      closeCurrent()
      return route.fulfill({ status: 204 })
    }
    const reopen = path.match(/\/weeks\/history\/(\d+)\/reopen$/)
    if (reopen && request.method() === 'POST') {
      const id = Number(reopen[1])
      const past = history.find((week) => week.id === id)
      if (!past) return route.fulfill({ status: 404, json: { error: { code: 'past_week_not_found', message: 'The past plan was not found.' } } })
      reopened.push(id)
      closeCurrent()
      history = history.filter((week) => week.id !== id)
      current = { id, startsOn: past.startsOn, endsOn: null, recipes: [] }
      return route.fulfill({ json: current })
    }
    if (path.endsWith('/week/current/groceries')) {
      return current ? route.fulfill({ json: { startsOn: current.startsOn, lines: [] } }) : route.fulfill({ status: 404, json: noCurrentWeek })
    }
    if (path.endsWith('/week/current')) return current ? route.fulfill({ json: current }) : route.fulfill({ status: 404, json: noCurrentWeek })
    if (path.endsWith('/recipes')) return route.fulfill({ json: { recipes } })
    if (path.endsWith('/weeks/history')) return route.fulfill({ json: { weeks: history } })
    if (path.endsWith(`/weeks/history/${closedWeek.id}`)) {
      return route.fulfill({ json: {
        week: { id: closedWeek.id, startsOn: closedWeek.startsOn, endsOn: closedWeek.endsOn, recipes: [] },
        groceries: { startsOn: closedWeek.startsOn, lines: [] },
      } })
    }
    if (path.endsWith('/store-sections')) return route.fulfill({ json: { sections: [] } })
    return route.fulfill({ status: 404, json: { error: { code: 'not_found', message: `${path} is not mocked.` } } })
  })
  return { generated, closes: () => closes, reopened }
}

test('after a plan closes the Plan page leads straight into starting a new one', async ({ page }) => {
  const api = await setup(page)
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Start a new plan' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Build your first plan' })).toHaveCount(0)
  await expect(page.locator('.plan-next-week')).toContainText('Your plan for Nov 1 – Nov 10, 2026 is in History.')
  await expect(page.getByRole('link', { name: 'View or reopen it' })).toHaveAttribute('href', '/history/7')
  // The household's default count is ready; nothing is generated until Generate is pressed.
  await expect(page.locator('.count-field select')).toHaveValue('4')
  expect(api.generated).toEqual([])
  // There is no plan to close.
  await expect(page.getByRole('button', { name: 'Start new plan' })).toHaveCount(0)

  await page.getByRole('button', { name: 'Generate plan' }).click()
  await expect(page.locator('.recipe-row')).toHaveCount(4)
  await expect(page.getByText('Planned · 4 meals')).toBeVisible()
  // A plan has no set length: it shows where it began.
  await expect(page.locator('.page-heading p')).toHaveText('Since Nov 12, 2026')
  await expect(page.getByRole('heading', { name: 'Start a new plan' })).toHaveCount(0)
  expect(api.generated).toEqual([{ recipeCount: 4 }])
})

test('the History link opens the plan that ended', async ({ page }) => {
  await setup(page)
  await page.goto('/')
  await page.getByRole('link', { name: 'View or reopen it' }).click()
  await expect(page.getByRole('heading', { name: 'Nov 1 – Nov 10, 2026' })).toBeVisible()
})

test('after a plan closes the Groceries page points to History and to a new plan', async ({ page }) => {
  await setup(page)
  await page.goto('/groceries')
  await expect(page.getByRole('heading', { name: 'Your last list is in History' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Generate a plan first' })).toHaveCount(0)
  await expect(page.getByRole('link', { name: 'See the last list' })).toHaveAttribute('href', '/history/7')
  await page.getByRole('link', { name: 'Start a new plan' }).click()
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByRole('heading', { name: 'Start a new plan' })).toBeVisible()
  await expect(page.locator('.count-field select')).toHaveValue('4')
})

test('History shows each plan from its start to its end, of any length', async ({ page }) => {
  await setup(page)
  await page.goto('/history')
  await expect(page.locator('.history-row').first()).toContainText('Nov 1 – Nov 10, 2026')
})

test('a household with no plans at all keeps the first-time start', async ({ page }) => {
  await setup(page, { pastWeeks: false })
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Build your first plan' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Start a new plan' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Start new plan' })).toHaveCount(0)
  await expect(page.locator('.count-field select')).toHaveValue('4')

  await page.goto('/groceries')
  await expect(page.getByRole('heading', { name: 'Generate a plan first' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Start a new plan' })).toHaveCount(0)
})

test('reopening a past plan makes it current and moves the current plan to History', async ({ page }) => {
  const api = await setup(page, { current: plannedWeek(3) })
  await page.goto('/history/7')
  await page.getByRole('button', { name: 'Reopen this plan' }).click()
  await expect(page).toHaveURL(/\/$/)
  await expect(page.locator('.page-heading p')).toHaveText('Since Nov 1, 2026')
  expect(api.reopened).toEqual([7])

  await page.getByRole('navigation', { name: 'Primary navigation' }).getByRole('link', { name: 'History' }).click()
  await expect(page.locator('.history-row')).toHaveCount(1)
  await expect(page.locator('.history-row').first()).toHaveAttribute('href', '/history/8')
})

const plannedWeek = (count: number): CurrentWeek => ({
  id: 8, startsOn: '2026-11-12', endsOn: null,
  recipes: Array.from({ length: count }, (_, index) => ({
    id: index + 1, position: index,
    recipe: { id: index + 1, key: `recipe-${index + 1}`, name: `Recipe ${index + 1}`, imageUrl: null, yield: null,
      handsOn: { minimumMinutes: null, maximumMinutes: null }, unattended: { minimumMinutes: null, maximumMinutes: null } },
  })),
})

test('Start new plan closes the plan and leads into the next in three presses', async ({ page }) => {
  // Ten days after the plan began, the household shops again.
  const api = await setup(page, { current: plannedWeek(3), next: { id: 9, startsOn: '2026-11-22', endsOn: null }, closedEndsOn: '2026-11-22' })
  await page.goto('/')
  await expect(page.getByText('Planned · 3 meals')).toBeVisible()

  // Press 1: the button only asks; nothing is closed yet.
  await page.getByRole('button', { name: 'Start new plan' }).click()
  const dialog = page.getByRole('dialog', { name: 'Start a new plan' })
  await expect(dialog).toBeVisible()
  await expect(dialog).toContainText('This plan’s meals and grocery list move to History, where you can reopen it later.')
  await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeVisible()
  expect(api.closes()).toBe(0)

  // Press 2: the plan closes; nothing is generated.
  await dialog.getByRole('button', { name: 'Close plan and start a new one' }).click()
  await expect(page.getByRole('heading', { name: 'Start a new plan' })).toBeVisible()
  await expect(dialog).toHaveCount(0)
  await expect(page.locator('.plan-next-week')).toContainText('Your plan for Nov 12 – Nov 22, 2026 is in History.')
  await expect(page.getByRole('link', { name: 'View or reopen it' })).toHaveAttribute('href', '/history/8')
  const count = page.locator('.count-field select')
  await expect(count).toBeFocused()
  await expect(count).toBeInViewport()
  await expect(count).toHaveValue('4')
  await expect(page.getByRole('button', { name: 'Start new plan' })).toHaveCount(0)
  expect(api.closes()).toBe(1)
  expect(api.generated).toEqual([])

  // Press 3: the next plan is generated.
  await page.getByRole('button', { name: 'Generate plan' }).click()
  await expect(page.locator('.recipe-row')).toHaveCount(4)
  await expect(page.getByText('Since Nov 22, 2026')).toBeVisible()
  expect(api.generated).toEqual([{ recipeCount: 4 }])

  // The plan just closed is first in History.
  await page.getByRole('navigation', { name: 'Primary navigation' }).getByRole('link', { name: 'History' }).click()
  await expect(page.locator('.history-row').first()).toContainText('Nov 12 – Nov 22, 2026')
  await expect(page.locator('.history-row').first()).toHaveAttribute('href', '/history/8')
})

test('cancelling Start new plan keeps the plan', async ({ page }) => {
  const api = await setup(page, { current: plannedWeek(3) })
  await page.goto('/')
  const start = page.getByRole('button', { name: 'Start new plan' })
  await start.click()
  const dialog = page.getByRole('dialog', { name: 'Start a new plan' })
  await dialog.getByRole('button', { name: 'Cancel' }).click()
  await expect(dialog).toHaveCount(0)
  await expect(start).toBeFocused()
  await page.keyboard.press('Enter')
  await expect(dialog).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(dialog).toHaveCount(0)
  await expect(page.getByText('Planned · 3 meals')).toBeVisible()
  expect(api.closes()).toBe(0)
})

test('a close that succeeds still leads into a new plan when the plan cannot be reloaded', async ({ page }) => {
  await setup(page, { current: plannedWeek(3) })
  await page.goto('/')
  await expect(page.getByText('Planned · 3 meals')).toBeVisible()
  // From here on, reading the current plan and History fails. Later routes take precedence.
  await page.route(/\/api\/v2\/households\/test\/(week\/current|weeks\/history)$/, route => route.fulfill({
    status: 500, json: { error: { code: 'internal_error', message: 'The request could not be completed.' } },
  }))
  await page.getByRole('button', { name: 'Start new plan' }).click()
  await page.getByRole('dialog', { name: 'Start a new plan' }).getByRole('button', { name: 'Close plan and start a new one' }).click()
  await expect(page.getByRole('heading', { name: 'Start a new plan' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'View or reopen it' })).toHaveAttribute('href', '/history/8')
  await expect(page.locator('.count-field select')).toBeFocused()
  await expect(page.getByText('Planned · 3 meals')).toHaveCount(0)
  await expect(page.getByText('Couldn’t load your plan.')).toHaveCount(0)
})
