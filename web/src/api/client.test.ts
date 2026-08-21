import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiRequestError, generateWeek, getCurrentWeek, listRecipes } from './client'

const emptyWeek = { id: 1, startsOn: '2026-08-16', recipes: [] }

afterEach(() => vi.unstubAllGlobals())

describe('API client', () => {
  it('represents a missing current week as intentional empty state', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ error: { code: 'no_current_week', message: 'Not generated.' } }),
      { status: 404, headers: { 'Content-Type': 'application/json' } },
    )))

    await expect(getCurrentWeek()).resolves.toBeNull()
  })

  it('sends the selected recipe count when generating', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(emptyWeek), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }))
    vi.stubGlobal('fetch', fetchMock)

    await expect(generateWeek(5)).resolves.toEqual(emptyWeek)
    expect(fetchMock).toHaveBeenCalledWith('/api/week/current/generate', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ recipeCount: 5 }),
    }))
  })

  it('retains structured API failures', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ error: { code: 'internal_error', message: 'Could not load recipes.' } }),
      { status: 500, headers: { 'Content-Type': 'application/json' } },
    )))

    const error = await listRecipes().catch((reason: unknown) => reason)
    expect(error).toBeInstanceOf(ApiRequestError)
    expect(error).toMatchObject({ status: 500, code: 'internal_error', message: 'Could not load recipes.' })
  })
})
