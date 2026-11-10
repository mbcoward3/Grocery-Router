import { describe, expect, it } from 'vitest'
import { deviceToday, formatWeekRange } from './dates'

describe('plan dates', () => {
  it('shows the range the server reports, of any length', () => {
    expect(formatWeekRange('2026-11-08', '2026-11-14', 'en-US')).toBe('Nov 8 – Nov 14, 2026')
    expect(formatWeekRange('2026-08-21', '2026-08-31', 'en-US')).toBe('Aug 21 – Aug 31, 2026')
    expect(formatWeekRange('2026-12-27', '2027-01-02', 'en-US')).toBe('Dec 27 – Jan 2, 2027')
  })

  it('shows where the current plan began', () => {
    expect(formatWeekRange('2026-11-11', null, 'en-US')).toBe('Since Nov 11, 2026')
  })

  it('reads the calendar date on the viewer device', () => {
    expect(deviceToday(new Date(2026, 10, 7, 23, 30))).toBe('2026-11-07')
    expect(deviceToday(new Date(2027, 0, 2, 0, 5))).toBe('2027-01-02')
  })
})
