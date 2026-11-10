// Week dates are household-local calendar dates (YYYY-MM-DD). They are formatted in UTC so the
// viewer's own zone never moves them a day.

function asDate(value: string): Date {
  return new Date(`${value}T00:00:00Z`)
}

export function formatShortDate(value: string, locale?: string): string {
  return new Intl.DateTimeFormat(locale, { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(asDate(value))
}

function formatLongDate(value: string, locale?: string): string {
  return new Intl.DateTimeFormat(locale, { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' }).format(asDate(value))
}

// formatWeekRange shows a week's first and last dates, or only where it began when it has no end
// yet (an open week in manual mode).
export function formatWeekRange(startsOn: string, endsOn: string | null, locale?: string): string {
  if (endsOn === null) return `Since ${formatLongDate(startsOn, locale)}`
  return `${formatShortDate(startsOn, locale)} – ${formatLongDate(endsOn, locale)}`
}

export function formatWeekday(value: string, locale?: string): string {
  return new Intl.DateTimeFormat(locale, { weekday: 'long', timeZone: 'UTC' }).format(asDate(value))
}
