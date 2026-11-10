import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { ApiRequestError, updateSettings, type HouseholdSettings } from '../../api/client'
import { settingsQueryKey, settingsQueryOptions } from './queries'

const minRecipeCount = 1
const maxRecipeCount = 21
// Index is the API's weekStartDay: 0 is Sunday, as in JavaScript's Date.getDay.
const weekdays = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
const clockPattern = /^([01][0-9]|2[0-3]):[0-5][0-9]$/

type RolloverMode = HouseholdSettings['rolloverMode']
type Field = 'timezone' | 'count' | 'day' | 'time' | 'mode'

export function SettingsPage() {
  const query = useQuery(settingsQueryOptions)

  return (
    <section aria-labelledby="settings-heading">
      <header className="page-heading">
        <div>
          <div className="eyebrow">Household</div>
          <h1 id="settings-heading">Settings</h1>
          <p>Preferences shared by everyone in your household.</p>
        </div>
      </header>
      {query.isLoading && <div className="settings-panel settings-skeleton" aria-label="Loading settings" aria-busy="true" />}
      {query.error && (
        <div className="error-panel" role="alert">
          <div><strong>Couldn’t load settings.</strong><p>{query.error.message}</p></div>
          <button className="button" type="button" onClick={() => void query.refetch()}>Try again</button>
        </div>
      )}
      {query.data && <SettingsForm saved={query.data} />}
    </section>
  )
}

function SettingsForm({ saved }: { saved: HouseholdSettings }) {
  const queryClient = useQueryClient()
  const [timezone, setTimezone] = useState(saved.timezone)
  const [recipeCount, setRecipeCount] = useState(String(saved.defaultRecipeCount))
  const [weekStartDay, setWeekStartDay] = useState(saved.weekStartDay)
  const [rolloverTime, setRolloverTime] = useState(saved.rolloverTime)
  const [rolloverMode, setRolloverMode] = useState<RolloverMode>(saved.rolloverMode)
  // A field the form itself rejected before saving, with the message to show.
  const [localError, setLocalError] = useState<{ field: Field; message: string } | null>(null)
  const [status, setStatus] = useState('')
  const timezoneRef = useRef<HTMLSelectElement>(null)
  const countRef = useRef<HTMLInputElement>(null)
  const dayRef = useRef<HTMLSelectElement>(null)
  const timeRef = useRef<HTMLInputElement>(null)
  const modeRef = useRef<HTMLInputElement>(null)
  const focusField = (field: Field | null) => {
    const refs = { timezone: timezoneRef, count: countRef, day: dayRef, time: timeRef, mode: modeRef }
    if (field) refs[field].current?.focus()
  }
  const deviceZone = Intl.DateTimeFormat().resolvedOptions().timeZone
  const zones = useMemo(() => timeZoneOptions([saved.timezone, timezone, deviceZone]), [saved.timezone, timezone, deviceZone])

  const mutation = useMutation({
    mutationFn: updateSettings,
    onSuccess: (settings) => {
      queryClient.setQueryData(settingsQueryKey, settings)
      setStatus('Settings saved.')
      // A new time zone or rollover moves the current week's end, so the week, its groceries, and
      // history reload.
      void queryClient.invalidateQueries({ queryKey: ['week'] })
      void queryClient.invalidateQueries({ queryKey: ['weeks'] })
    },
  })

  const change: Partial<HouseholdSettings> = {}
  if (timezone !== saved.timezone) change.timezone = timezone
  const parsedCount = recipeCount.trim() === '' ? Number.NaN : Number(recipeCount)
  if (parsedCount !== saved.defaultRecipeCount) change.defaultRecipeCount = parsedCount
  if (weekStartDay !== saved.weekStartDay) change.weekStartDay = weekStartDay
  if (rolloverTime !== saved.rolloverTime) change.rolloverTime = rolloverTime
  if (rolloverMode !== saved.rolloverMode) change.rolloverMode = rolloverMode
  const changed = Object.keys(change).length > 0

  // When a stored setting changes (another member saved, or this form did), its field follows it,
  // so Save never offers to write back a value nobody here chose. Only the changed field resets.
  const previous = useRef(saved)
  const { reset } = mutation
  useEffect(() => {
    const before = previous.current
    if (before.timezone === saved.timezone && before.defaultRecipeCount === saved.defaultRecipeCount &&
      before.weekStartDay === saved.weekStartDay && before.rolloverTime === saved.rolloverTime &&
      before.rolloverMode === saved.rolloverMode) return
    previous.current = saved
    if (before.timezone !== saved.timezone) setTimezone(saved.timezone)
    if (before.defaultRecipeCount !== saved.defaultRecipeCount) setRecipeCount(String(saved.defaultRecipeCount))
    if (before.weekStartDay !== saved.weekStartDay) setWeekStartDay(saved.weekStartDay)
    if (before.rolloverTime !== saved.rolloverTime) setRolloverTime(saved.rolloverTime)
    if (before.rolloverMode !== saved.rolloverMode) setRolloverMode(saved.rolloverMode)
    setLocalError(null)
    reset()
  }, [saved, reset])

  // A rejected save moves focus to the field it names. This runs after render, once the fields
  // reflect the failed save.
  useEffect(() => {
    // focusField reads only refs, so the error alone decides when this runs.
    if (mutation.error) focusField(invalidField(mutation.error))
  }, [mutation.error])
  const saveUnavailable = mutation.isPending || !changed

  const edited = () => {
    setStatus('')
    setLocalError(null)
    mutation.reset()
  }

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    // Save stays focusable while unavailable (aria-disabled), so a second press lands here.
    if (saveUnavailable) return
    setStatus('')
    const count = change.defaultRecipeCount
    let rejected: { field: Field; message: string } | null = null
    if (count !== undefined && !(Number.isInteger(count) && count >= minRecipeCount && count <= maxRecipeCount)) {
      rejected = { field: 'count', message: `Recipes per week must be a whole number from ${minRecipeCount} to ${maxRecipeCount}.` }
    } else if (change.rolloverTime !== undefined && !clockPattern.test(change.rolloverTime)) {
      rejected = { field: 'time', message: 'Choose the time of day the week rolls over.' }
    }
    setLocalError(rejected)
    if (rejected) {
      focusField(rejected.field)
      return
    }
    mutation.mutate(change)
  }

  const invalid = localError?.field ?? (mutation.error ? invalidField(mutation.error) : null)
  const error = localError?.message ?? mutation.error?.message ?? null
  const describedBy = (field: Field, help: string) => (invalid === field ? `${help} settings-error` : help)
  return (
    // While a save is pending the fields ignore edits but stay enabled, so focus (for example after
    // Enter in the count field) never falls back to the page body.
    <form className="settings-panel" onSubmit={submit} noValidate>
      <div className="settings-field">
        <label htmlFor="settings-timezone">Time zone</label>
        <p className="settings-help" id="settings-timezone-help">Decides when your household’s week begins.</p>
        <div className="settings-control">
          <select
            id="settings-timezone"
            ref={timezoneRef}
            aria-describedby={describedBy('timezone', 'settings-timezone-help')}
            aria-invalid={invalid === 'timezone' || undefined}
            value={timezone}
            onChange={(event) => { if (!mutation.isPending) { setTimezone(event.target.value); edited() } }}
            aria-readonly={mutation.isPending || undefined}
          >
            {zones.map((zone) => <option key={zone} value={zone}>{zone.replaceAll('_', ' ')}</option>)}
          </select>
          <button
            className="button"
            type="button"
            onClick={() => { setTimezone(deviceZone); edited() }}
            disabled={mutation.isPending || timezone === deviceZone}
          >
            Use this device’s time zone
          </button>
        </div>
      </div>
      <div className="settings-field">
        <label htmlFor="settings-recipe-count">Recipes per week</label>
        <p className="settings-help" id="settings-recipe-count-help">How many recipes a new week starts with, from {minRecipeCount} to {maxRecipeCount}.</p>
        <div className="settings-control">
          <input
            id="settings-recipe-count"
            ref={countRef}
            aria-describedby={describedBy('count', 'settings-recipe-count-help')}
            aria-invalid={invalid === 'count' || undefined}
            type="number"
            inputMode="numeric"
            min={minRecipeCount}
            max={maxRecipeCount}
            step={1}
            value={recipeCount}
            onChange={(event) => { if (!mutation.isPending) { setRecipeCount(event.target.value); edited() } }}
            readOnly={mutation.isPending}
          />
        </div>
      </div>
      <div className="settings-field">
        <label htmlFor="settings-week-start">Week starts on</label>
        <p className="settings-help" id="settings-week-start-help">The day one week ends and the next begins.</p>
        <div className="settings-control">
          <select
            id="settings-week-start"
            ref={dayRef}
            aria-describedby={describedBy('day', 'settings-week-start-help')}
            aria-invalid={invalid === 'day' || undefined}
            value={weekStartDay}
            onChange={(event) => { if (!mutation.isPending) { setWeekStartDay(Number(event.target.value)); edited() } }}
            aria-readonly={mutation.isPending || undefined}
          >
            {weekdays.map((day, index) => <option key={day} value={index}>{day}</option>)}
          </select>
        </div>
      </div>
      <div className="settings-field">
        <label htmlFor="settings-rollover-time">Rolls over at</label>
        <p className="settings-help" id="settings-rollover-time-help">The time on that day, in your household’s time zone.</p>
        <div className="settings-control">
          <input
            id="settings-rollover-time"
            ref={timeRef}
            className="settings-time"
            aria-describedby={describedBy('time', 'settings-rollover-time-help')}
            aria-invalid={invalid === 'time' || undefined}
            type="time"
            step={60}
            value={rolloverTime}
            onChange={(event) => { if (!mutation.isPending) { setRolloverTime(event.target.value); edited() } }}
            readOnly={mutation.isPending}
          />
        </div>
      </div>
      <div
        className="settings-field"
        role="radiogroup"
        aria-labelledby="settings-mode-label"
        aria-describedby={describedBy('mode', 'settings-mode-help')}
        aria-invalid={invalid === 'mode' || undefined}
      >
        <span className="settings-label" id="settings-mode-label">Rollover</span>
        <p className="settings-help" id="settings-mode-help">How the week ends.</p>
        <div className="settings-control settings-choices">
          {([
            ['automatic', 'Automatic', 'The week closes itself at that day and time.'],
            ['manual', 'Manual', 'The week stays open until someone starts the next one.'],
          ] as const).map(([mode, name, help]) => (
            <div className="settings-choice" key={mode}>
              <label>
                <input
                  ref={mode === rolloverMode ? modeRef : undefined}
                  type="radio"
                  name="settings-rollover-mode"
                  value={mode}
                  checked={rolloverMode === mode}
                  aria-describedby={`settings-mode-${mode}-help`}
                  onChange={() => { if (!mutation.isPending) { setRolloverMode(mode); edited() } }}
                />
                {name}
              </label>
              <p className="settings-help" id={`settings-mode-${mode}-help`}>{help}</p>
            </div>
          ))}
        </div>
      </div>
      {error && <p className="inline-error" id="settings-error" role="alert">{error}</p>}
      <footer className="settings-footer">
        <span className="settings-saved" role="status">{status}</span>
        <button className="button primary" type="submit" aria-disabled={saveUnavailable}>
          {mutation.isPending ? 'Saving…' : 'Save'}
        </button>
      </footer>
    </form>
  )
}

// invalidField names the field a rejected save points at, so focus can move there. The messages
// are pinned by TestValidationMessages in internal/settings.
function invalidField(error: Error): Field | null {
  if (!(error instanceof ApiRequestError) || error.code !== 'invalid_settings') return null
  if (/time zone/i.test(error.message)) return 'timezone'
  if (/week start day/i.test(error.message)) return 'day'
  if (/rollover time/i.test(error.message)) return 'time'
  if (/rollover mode/i.test(error.message)) return 'mode'
  return 'count'
}

// timeZoneOptions lists the browser's zones plus any extra (the saved zone, the device zone)
// that the browser leaves out, such as UTC in some engines.
function timeZoneOptions(extra: string[]): string[] {
  const zones = new Set(typeof Intl.supportedValuesOf === 'function' ? Intl.supportedValuesOf('timeZone') : [])
  for (const zone of extra) if (zone) zones.add(zone)
  return [...zones].sort((a, b) => a.localeCompare(b))
}
