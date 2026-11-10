import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useRef, useState, type FormEvent } from 'react'
import { ApiRequestError, updateSettings, type HouseholdSettings } from '../../api/client'
import { settingsQueryKey, settingsQueryOptions } from './queries'

const minRecipeCount = 1
const maxRecipeCount = 21

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
  const [baseline, setBaseline] = useState(saved)
  const [countError, setCountError] = useState<string | null>(null)
  const [status, setStatus] = useState('')
  const timezoneRef = useRef<HTMLSelectElement>(null)
  const countRef = useRef<HTMLInputElement>(null)
  const deviceZone = Intl.DateTimeFormat().resolvedOptions().timeZone
  const zones = useMemo(() => timeZoneOptions([saved.timezone, timezone, deviceZone]), [saved.timezone, timezone, deviceZone])

  // When the stored settings change (another member saved, or this form did), the fields follow
  // them, so Save never offers to write back values nobody here chose.
  if (saved.timezone !== baseline.timezone || saved.defaultRecipeCount !== baseline.defaultRecipeCount) {
    setBaseline(saved)
    setTimezone(saved.timezone)
    setRecipeCount(String(saved.defaultRecipeCount))
  }

  const mutation = useMutation({
    mutationFn: updateSettings,
    onSuccess: (settings) => {
      queryClient.setQueryData(settingsQueryKey, settings)
      setStatus('Settings saved.')
      // A new time zone can move the current week, so the week, its groceries, and history reload.
      void queryClient.invalidateQueries({ queryKey: ['week'] })
      void queryClient.invalidateQueries({ queryKey: ['weeks'] })
    },
  })

  const change: Partial<HouseholdSettings> = {}
  if (timezone !== saved.timezone) change.timezone = timezone
  const parsedCount = recipeCount.trim() === '' ? Number.NaN : Number(recipeCount)
  if (parsedCount !== saved.defaultRecipeCount) change.defaultRecipeCount = parsedCount
  const changed = Object.keys(change).length > 0
  const saveUnavailable = mutation.isPending || !changed

  const edited = () => {
    setStatus('')
    setCountError(null)
    mutation.reset()
  }

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    // Save stays focusable while unavailable (aria-disabled), so a second press lands here.
    if (saveUnavailable) return
    setStatus('')
    const count = change.defaultRecipeCount
    if (count !== undefined && !(Number.isInteger(count) && count >= minRecipeCount && count <= maxRecipeCount)) {
      setCountError(`Recipes per week must be a whole number from ${minRecipeCount} to ${maxRecipeCount}.`)
      countRef.current?.focus()
      return
    }
    setCountError(null)
    mutation.mutate(change, {
      onError: (error) => {
        const field = invalidField(error)
        if (field === 'timezone') timezoneRef.current?.focus()
        if (field === 'count') countRef.current?.focus()
      },
    })
  }

  const serverField = mutation.error ? invalidField(mutation.error) : null
  const error = countError ?? mutation.error?.message ?? null
  const timezoneInvalid = !countError && serverField === 'timezone'
  const countInvalid = Boolean(countError) || serverField === 'count'
  return (
    <form className="settings-panel" onSubmit={submit} noValidate>
      <div className="settings-field">
        <label htmlFor="settings-timezone">Time zone</label>
        <p className="settings-help" id="settings-timezone-help">Decides when your household’s week begins.</p>
        <div className="settings-control">
          <select
            id="settings-timezone"
            ref={timezoneRef}
            aria-describedby={timezoneInvalid ? 'settings-timezone-help settings-error' : 'settings-timezone-help'}
            aria-invalid={timezoneInvalid || undefined}
            value={timezone}
            onChange={(event) => { setTimezone(event.target.value); edited() }}
            disabled={mutation.isPending}
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
            aria-describedby={countInvalid ? 'settings-recipe-count-help settings-error' : 'settings-recipe-count-help'}
            aria-invalid={countInvalid || undefined}
            type="number"
            inputMode="numeric"
            min={minRecipeCount}
            max={maxRecipeCount}
            step={1}
            value={recipeCount}
            onChange={(event) => { setRecipeCount(event.target.value); edited() }}
            disabled={mutation.isPending}
          />
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

// invalidField names the field a rejected save points at, so focus can move there.
function invalidField(error: Error): 'timezone' | 'count' | null {
  if (!(error instanceof ApiRequestError) || error.code !== 'invalid_settings') return null
  return /time zone/i.test(error.message) ? 'timezone' : 'count'
}

// timeZoneOptions lists the browser's zones plus any extra (the saved zone, the device zone)
// that the browser leaves out, such as UTC in some engines.
function timeZoneOptions(extra: string[]): string[] {
  const zones = new Set(typeof Intl.supportedValuesOf === 'function' ? Intl.supportedValuesOf('timeZone') : [])
  for (const zone of extra) if (zone) zones.add(zone)
  return [...zones].sort((a, b) => a.localeCompare(b))
}
