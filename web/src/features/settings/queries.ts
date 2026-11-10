import { queryOptions } from '@tanstack/react-query'
import { getSettings } from '../../api/client'

export const settingsQueryKey = ['settings'] as const

export const settingsQueryOptions = queryOptions({
  queryKey: settingsQueryKey,
  queryFn: getSettings,
})
