import type { components } from './schema'

export type ApiError = components['schemas']['APIError']
export type RecipeSummary = components['schemas']['RecipeSummary']
export type Week = components['schemas']['Week']
export type RecipeDetail = components['schemas']['RecipeDetail']
export type Groceries = components['schemas']['Groceries']
export type GroceryContribution = components['schemas']['GroceryContribution']
export type GroceryLine = components['schemas']['GroceryLine']

export class ApiRequestError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, error: ApiError['error']) {
    super(error.message)
    this.name = 'ApiRequestError'
    this.status = status
    this.code = error.code
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })

  const body: unknown = await response.json()
  if (!response.ok) {
    const fallback: ApiError['error'] = {
      code: 'request_failed',
      message: 'The request could not be completed.',
    }
    const error = isApiError(body) ? body.error : fallback
    throw new ApiRequestError(response.status, error)
  }
  return body as T
}

function isApiError(value: unknown): value is ApiError {
  if (!value || typeof value !== 'object' || !('error' in value)) return false
  const error = value.error
  return Boolean(
    error &&
      typeof error === 'object' &&
      'code' in error &&
      typeof error.code === 'string' &&
      'message' in error &&
      typeof error.message === 'string',
  )
}

export function listRecipes(): Promise<{ recipes: RecipeSummary[] }> {
  return request('/recipes')
}

export function getRecipe(recipeId: number): Promise<RecipeDetail> {
  return request(`/recipes/${recipeId}`)
}

export async function getCurrentWeek(): Promise<Week | null> {
  try {
    return await request('/week/current')
  } catch (error) {
    if (error instanceof ApiRequestError && error.code === 'no_current_week') return null
    throw error
  }
}

export function generateWeek(recipeCount: number): Promise<Week> {
  return request('/week/current/generate', {
    method: 'POST',
    body: JSON.stringify({ recipeCount }),
  })
}

export function addRecipe(recipeId: number): Promise<Week> {
  return request('/week/current/recipes', {
    method: 'POST',
    body: JSON.stringify({ recipeId }),
  })
}

export function removeRecipe(occurrenceId: number): Promise<Week> {
  return request(`/week/current/recipes/${occurrenceId}`, { method: 'DELETE' })
}

export function swapRecipe(occurrenceId: number, recipeId: number): Promise<Week> {
  return request(`/week/current/recipes/${occurrenceId}`, {
    method: 'PUT',
    body: JSON.stringify({ recipeId }),
  })
}

export function randomSwapRecipe(occurrenceId: number): Promise<Week> {
  return request(`/week/current/recipes/${occurrenceId}/random-swap`, { method: 'POST' })
}

export async function getGroceries(): Promise<Groceries | null> {
  try {
    return await request('/week/current/groceries')
  } catch (error) {
    if (error instanceof ApiRequestError && error.code === 'no_current_week') return null
    throw error
  }
}

export function addGroceryLine(name: string): Promise<Groceries> {
  return request('/week/current/groceries', { method: 'POST', body: JSON.stringify({ name }) })
}

export function updateGroceryLine(lineId: number, update: { completed: boolean } | { overrideText: string }): Promise<Groceries> {
  return request(`/week/current/groceries/${lineId}`, { method: 'PATCH', body: JSON.stringify(update) })
}

export function removeGroceryLine(lineId: number): Promise<Groceries> {
  return request(`/week/current/groceries/${lineId}`, { method: 'DELETE' })
}

export function getGroceryContributions(lineId: number): Promise<{ contributions: GroceryContribution[] }> {
  return request(`/week/current/groceries/${lineId}/contributions`)
}
