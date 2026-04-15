import type { Setting, ProviderOption } from '@/types/settings'
import { SettingField } from './SettingField'

interface SettingsPanelProps {
  settings: Setting[]
  providerOptions: ProviderOption[]
  modifiedKeys?: Set<string>
  onChange?: (key: string, value: string | number | boolean) => void
}

export function SettingsPanel({
  settings,
  providerOptions,
  modifiedKeys = new Set(),
  onChange,
}: SettingsPanelProps) {
  // Build a map of current setting values for dependsOn resolution
  const valueMap = new Map(settings.map((s) => [s.key, s.value]))

  // Group settings: toggles separated from other fields, and detect provider/model pairs
  const groups = groupSettings(settings)

  return (
    <div className="divide-y divide-slate-100 dark:divide-slate-700/50">
      {groups.map((group, idx) => (
        <div key={idx} className={group.label ? 'py-2' : ''}>
          {group.label && (
            <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 mb-1 pt-3 px-1">
              {group.label}
            </h4>
          )}
          {group.settings.map((setting) => (
            <SettingField
              key={setting.key}
              setting={setting}
              providerOptions={providerOptions}
              dependsOnValue={
                setting.dependsOn
                  ? String(valueMap.get(setting.dependsOn) ?? '')
                  : undefined
              }
              isModified={modifiedKeys.has(setting.key)}
              onChange={onChange}
            />
          ))}
        </div>
      ))}
    </div>
  )
}

interface SettingGroup {
  label?: string
  settings: Setting[]
}

function groupSettings(settings: Setting[]): SettingGroup[] {
  // Simple heuristic: group provider-select + model-select pairs together,
  // and separate toggles at the top
  const toggles = settings.filter((s) => s.type === 'toggle')
  const rest = settings.filter((s) => s.type !== 'toggle')

  const groups: SettingGroup[] = []

  if (toggles.length > 0) {
    groups.push({ settings: toggles })
  }

  if (rest.length > 0) {
    groups.push({ settings: rest })
  }

  return groups.length > 0 ? groups : [{ settings }]
}
