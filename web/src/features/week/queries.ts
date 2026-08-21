import { queryOptions } from '@tanstack/react-query'
import { getCurrentWeek, listRecipes } from '../../api/client'

export const weekQueryKey = ['week', 'current'] as const

export const currentWeekQueryOptions = queryOptions({
  queryKey: weekQueryKey,
  queryFn: getCurrentWeek,
})

export const recipesQueryOptions = queryOptions({
  queryKey: ['recipes', 'verified'] as const,
  queryFn: listRecipes,
  staleTime: Number.POSITIVE_INFINITY,
})
