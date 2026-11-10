import { queryOptions } from '@tanstack/react-query'
import { listWeekHistory } from '../../api/client'

export const historyQueryKey = ['weeks', 'history'] as const

// Closed weeks, newest first. The first one is the week that most recently ended.
export const historyQueryOptions = queryOptions({
  queryKey: historyQueryKey,
  queryFn: listWeekHistory,
})
