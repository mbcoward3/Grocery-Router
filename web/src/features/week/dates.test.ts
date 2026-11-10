import { describe, expect, it } from 'vitest'
import { formatWeekday, formatWeekRange } from './dates'

describe('week dates', () => {
  it('shows the range the server reports instead of assuming Sunday to Saturday', () => {
    expect(formatWeekRange('2026-11-08', '2026-11-14', 'en-US')).toBe('Nov 8 – Nov 14, 2026')
    expect(formatWeekRange('2026-08-21', '2026-08-28', 'en-US')).toBe('Aug 21 – Aug 28, 2026')
    expect(formatWeekRange('2026-12-27', '2027-01-02', 'en-US')).toBe('Dec 27 – Jan 2, 2027')
  })

  it('shows where an open-ended week began', () => {
    expect(formatWeekRange('2026-11-11', null, 'en-US')).toBe('Since Nov 11, 2026')
  })

  it('names the weekday of a calendar date in any viewer zone', () => {
    expect(formatWeekday('2026-11-07', 'en-US')).toBe('Saturday')
    expect(formatWeekday('2026-08-28', 'en-US')).toBe('Friday')
  })
})
