import { apiGet, apiPost, apiPut, apiPatch, apiDelete } from './client'
import type {
  Provider,
  ModelConfig,
  ProviderType,
  TestResult,
} from '@/types/provider-management'

// --- Backend response types ---

interface BackendModelConfig {
  name: string
  chat_path: string | null
  api_key_env: string
  type: string | null
  base_url: string | null
  enabled: boolean
}

interface BackendProvider {
  name: string
  type: string
  base_url: string
  api_key_env: string
  timeout: string
  rate_limit_seconds: number
  enabled: boolean
  models: string[]
  model_configs: BackendModelConfig[]
  model_count: number
  last_tested_at: string | null
  last_test_result: string | null
  last_test_latency_ms: number | null
}

interface ProvidersResponse {
  providers: BackendProvider[]
}

interface TestResponse {
  result: string
  latency_ms: number
  models_found?: number
  error?: string
}

// --- Transform ---

function transformModelConfig(mc: BackendModelConfig): ModelConfig {
  return {
    id: mc.name,
    name: mc.name,
    chatPath: mc.chat_path,
    apiKeyEnv: mc.api_key_env,
    type: (mc.type as ProviderType) ?? null,
    baseUrl: mc.base_url,
    enabled: mc.enabled,
  }
}

function transformProvider(p: BackendProvider): Provider {
  const modelConfigs = (p.model_configs ?? []).map(transformModelConfig)
  return {
    id: p.name,
    name: p.name,
    type: p.type as ProviderType,
    baseUrl: p.base_url,
    apiKeyEnv: p.api_key_env,
    timeout: p.timeout,
    rateLimitSeconds: p.rate_limit_seconds || null,
    enabled: p.enabled,
    hasModelConfigs: modelConfigs.length > 0,
    modelCount: p.model_count,
    models: p.models ?? [],
    modelConfigs,
    lastTestedAt: p.last_tested_at,
    lastTestResult: (p.last_test_result as TestResult) ?? null,
    lastTestLatencyMs: p.last_test_latency_ms,
  }
}

// --- Public API ---

export async function fetchProviders(): Promise<Provider[]> {
  const data = await apiGet<ProvidersResponse>('/providers')
  return data.providers.map(transformProvider)
}

export async function addProvider(data: {
  name: string
  type: ProviderType
  baseUrl: string
  apiKeyEnv: string
  timeout: string
  rateLimitSeconds: number | null
  enabled: boolean
}): Promise<Provider> {
  const resp = await apiPost<BackendProvider>('/providers', {
    name: data.name,
    type: data.type,
    base_url: data.baseUrl,
    api_key_env: data.apiKeyEnv,
    timeout: data.timeout,
    rate_limit_seconds: data.rateLimitSeconds ?? 0,
    enabled: data.enabled,
  })
  return transformProvider(resp)
}

export async function updateProvider(
  name: string,
  data: {
    type?: string
    baseUrl?: string
    apiKeyEnv?: string
    timeout?: string
    rateLimitSeconds?: number | null
    enabled?: boolean
  }
): Promise<void> {
  await apiPut(`/providers/${encodeURIComponent(name)}`, {
    name,
    type: data.type,
    base_url: data.baseUrl,
    api_key_env: data.apiKeyEnv,
    timeout: data.timeout,
    rate_limit_seconds: data.rateLimitSeconds ?? 0,
    enabled: data.enabled,
  })
}

export async function deleteProvider(name: string): Promise<void> {
  await apiDelete(`/providers/${encodeURIComponent(name)}`)
}

export async function toggleProvider(
  name: string,
  enabled: boolean
): Promise<void> {
  await apiPatch(`/providers/${encodeURIComponent(name)}/enable`, { enabled })
}

export async function testProvider(
  name: string
): Promise<{ result: TestResult; latencyMs: number; modelsFound?: number; error?: string }> {
  const data = await apiPost<TestResponse>(
    `/providers/${encodeURIComponent(name)}/test`
  )
  return {
    result: data.result as TestResult,
    latencyMs: data.latency_ms,
    modelsFound: data.models_found,
    error: data.error,
  }
}

export async function updateModelConfigs(
  providerName: string,
  configs: ModelConfig[]
): Promise<void> {
  await apiPut(
    `/providers/${encodeURIComponent(providerName)}/model-configs`,
    {
      model_configs: configs.map((c) => ({
        name: c.name,
        chat_path: c.chatPath ?? '',
        api_key_env: c.apiKeyEnv,
        type: c.type ?? '',
        base_url: c.baseUrl ?? '',
        enabled: c.enabled,
      })),
    }
  )
}
