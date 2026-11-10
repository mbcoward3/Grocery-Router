// Plan dates are household-local calendar dates (YYYY-MM-DD). They are formatted in UTC so the
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

// formatWeekRange shows a plan's first and last dates, or only where it began while it is the
// current plan and has no end yet.
export function formatWeekRange(startsOn: string, endsOn: string | null, locale?: string): string {
  if (endsOn === null) return `Since ${formatLongDate(startsOn, locale)}`
  return `${formatShortDate(startsOn, locale)} – ${formatLongDate(endsOn, locale)}`
}

// deviceToday is the viewer's own calendar date (YYYY-MM-DD), for when the household's is not known.
export function deviceToday(now = new Date()): string {
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
}
