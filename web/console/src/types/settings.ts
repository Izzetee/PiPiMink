/** Setting input type determines the rendered control */
export type SettingType =
  | 'text'
  | 'number'
  | 'toggle'
  | 'duration'
  | 'url'
  | 'secret'
  | 'provider-select'
  | 'model-select'

/** Setting category tab */
export type SettingCategory =
  | 'routing'
  | 'cache'
  | 'database'
  | 'server'
  | 'benchmarking'
  | 'observability'

/** Validation constraints for a setting */
export interface SettingValidation {
  min?: number
  max?: number
  step?: number
  pattern?: string
}

/** A single system configuration value */
export interface Setting {
  key: string
  value: string | number | boolean
  type: SettingType
  label: string
  description: string
  required: boolean
  /** For model-select fields: the key of the provider-select this depends on */
  dependsOn?: string
  validation?: SettingValidation
}

/** All settings grouped by category */
export interface SettingsMap {
  routing: Setting[]
  cache: Setting[]
  database: Setting[]
  server: Setting[]
  benchmarking: Setting[]
  observability: Setting[]
}

/** A stored API key shown as a vault card */
export interface ApiKey {
  id: string
  envVarName: string
  providerName: string
  providerId: string
  maskedValue: string
  lastUpdatedAt: string
}

/** A model option within a provider dropdown */
export interface ModelOption {
  id: string
  name: string
}

/** A provider available for selection in dropdowns and API key creation */
export interface ProviderOption {
  id: string
  name: string
  apiKeyEnvVars: string[]
  models: ModelOption[]
}

/** A pending change tracked by the global save bar */
export interface PendingChange {
  key: string
  category: SettingCategory | 'apiKeys'
  previousValue: string | number | boolean
  newValue: string | number | boolean
}

export interface SettingsProps {
  settings: SettingsMap
  apiKeys: ApiKey[]
  providerOptions: ProviderOption[]
  pendingChanges: PendingChange[]

  /** Called when a setting value is changed inline */
  onSettingChange?: (key: string, value: string | number | boolean) => void
  /** Called when the user clicks Save in the global save bar */
  onSaveAll?: (changes: PendingChange[]) => void
  /** Called when the user clicks Discard in the global save bar */
  onDiscardAll?: () => void

  /** Called when a new API key is added via the vault modal */
  onAddApiKey?: (envVarName: string, value: string) => void
  /** Called when an existing API key is updated */
  onEditApiKey?: (id: string, newValue: string) => void
  /** Called when an API key is deleted */
  onDeleteApiKey?: (id: string) => void
}
