/** Provider type identifier */
export type ProviderType = 'openai-compatible' | 'anthropic'

/** Result of a connectivity test */
export type TestResult = 'success' | 'error'

/** Per-model configuration override for multi-endpoint providers */
export interface ModelConfig {
  id: string
  name: string
  chatPath: string | null
  apiKeyEnv: string
  type: ProviderType | null
  baseUrl: string | null
  enabled: boolean
}

/** An LLM provider with connection details and optional per-model overrides */
export interface Provider {
  id: string
  name: string
  type: ProviderType
  baseUrl: string
  apiKeyEnv: string
  timeout: string
  rateLimitSeconds: number | null
  enabled: boolean
  hasModelConfigs: boolean
  modelCount: number
  models: string[]
  modelConfigs: ModelConfig[]
  lastTestedAt: string | null
  lastTestResult: TestResult | null
  lastTestLatencyMs: number | null
}

export interface ProviderManagementProps {
  providers: Provider[]

  /** Called when a new provider is saved */
  onAddProvider?: (provider: Omit<Provider, 'id' | 'modelCount' | 'models' | 'modelConfigs' | 'lastTestedAt' | 'lastTestResult' | 'lastTestLatencyMs'>) => void
  /** Called when an existing provider is updated */
  onEditProvider?: (id: string, updates: Partial<Provider>) => void
  /** Called when a provider is deleted */
  onDeleteProvider?: (id: string) => void
  /** Called when a provider is duplicated */
  onDuplicateProvider?: (id: string) => void
  /** Called when a provider is enabled or disabled */
  onToggleProvider?: (id: string, enabled: boolean) => void
  /** Called when model configs mode is toggled on a provider */
  onToggleModelConfigs?: (id: string, hasModelConfigs: boolean) => void
  /** Called when the user requests a connectivity test */
  onTestConnection?: (id: string) => void

  /** Called when a model config is added to a provider */
  onAddModelConfig?: (providerId: string, config: Omit<ModelConfig, 'id'>) => void
  /** Called when a model config is updated */
  onEditModelConfig?: (providerId: string, configId: string, updates: Partial<ModelConfig>) => void
  /** Called when a model config is enabled or disabled */
  onToggleModelConfig?: (providerId: string, configId: string, enabled: boolean) => void
  /** Called when a model config is removed */
  onDeleteModelConfig?: (providerId: string, configId: string) => void
}
