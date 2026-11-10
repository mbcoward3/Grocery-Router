import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState, type FormEvent } from 'react'
import { updateSettings, type HouseholdSettings } from '../../api/client'
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
  const [formError, setFormError] = useState<string | null>(null)
  const [justSaved, setJustSaved] = useState(false)
  const deviceZone = Intl.DateTimeFormat().resolvedOptions().timeZone
  const zones = useMemo(() => timeZoneOptions([saved.timezone, timezone, deviceZone]), [saved.timezone, timezone, deviceZone])

  const mutation = useMutation({
    mutationFn: updateSettings,
    onSuccess: (settings) => {
      queryClient.setQueryData(settingsQueryKey, settings)
      setJustSaved(true)
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

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setJustSaved(false)
    const count = change.defaultRecipeCount
    if (count !== undefined && !(Number.isInteger(count) && count >= minRecipeCount && count <= maxRecipeCount)) {
      setFormError(`Recipes per week must be a whole number from ${minRecipeCount} to ${maxRecipeCount}.`)
      return
    }
    setFormError(null)
    mutation.mutate(change)
  }

  const error = formError ?? mutation.error?.message ?? null
  return (
    <form className="settings-panel" onSubmit={submit} noValidate>
      <div className="settings-field">
        <label htmlFor="settings-timezone">Time zone</label>
        <p className="settings-help" id="settings-timezone-help">Decides when your household’s week begins.</p>
        <div className="settings-control">
          <select
            id="settings-timezone"
            aria-describedby="settings-timezone-help"
            value={timezone}
            onChange={(event) => { setTimezone(event.target.value); setJustSaved(false) }}
            disabled={mutation.isPending}
          >
            {zones.map((zone) => <option key={zone} value={zone}>{zone.replaceAll('_', ' ')}</option>)}
          </select>
          <button
            className="button"
            type="button"
            onClick={() => { setTimezone(deviceZone); setJustSaved(false) }}
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
            aria-describedby="settings-recipe-count-help"
            type="number"
            inputMode="numeric"
            min={minRecipeCount}
            max={maxRecipeCount}
            step={1}
            value={recipeCount}
            onChange={(event) => { setRecipeCount(event.target.value); setJustSaved(false) }}
            disabled={mutation.isPending}
          />
        </div>
      </div>
      {error && <p className="inline-error" role="alert">{error}</p>}
      <footer className="settings-footer">
        {justSaved && !changed && <span className="settings-saved" role="status">Settings saved.</span>}
        <button className="button primary" type="submit" disabled={mutation.isPending || !changed}>
          {mutation.isPending ? 'Saving…' : 'Save'}
        </button>
      </footer>
    </form>
  )
}

// timeZoneOptions lists the browser's zones plus any extra (the saved zone, the device zone)
// that the browser leaves out, such as UTC in some engines.
function timeZoneOptions(extra: string[]): string[] {
  const zones = new Set(Intl.supportedValuesOf('timeZone'))
  for (const zone of extra) if (zone) zones.add(zone)
  return [...zones].sort((a, b) => a.localeCompare(b))
}
