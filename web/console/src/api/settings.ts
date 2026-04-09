import { apiGet, apiPatch, apiPut } from './client'
import type {
  SettingsMap,
  ProviderOption,
  ApiKey,
  PendingChange,
} from '@/types/settings'

// ── Backend response types (not exported) ──

interface SettingsResponse {
  settings: SettingsMap
  providerOptions: ProviderOption[]
}

// ── Settings ──

export async function fetchSettings(): Promise<{
  settings: SettingsMap
  providerOptions: ProviderOption[]
}> {
  return apiGet<SettingsResponse>('/admin/settings')
}

export async function patchSettings(
  changes: PendingChange[]
): Promise<{
  settings: SettingsMap
  providerOptions: ProviderOption[]
}> {
  return apiPatch<SettingsResponse>('/admin/settings', {
    changes: changes.map((c) => ({ key: c.key, value: c.newValue })),
  })
}

// ── API Keys ──

export async function fetchApiKeys(): Promise<ApiKey[]> {
  return apiGet<ApiKey[]>('/admin/api-keys')
}

export async function addApiKey(
  envVarName: string,
  value: string
): Promise<ApiKey> {
  return apiPut<ApiKey>(`/admin/api-keys/${encodeURIComponent(envVarName)}`, {
    value,
  })
}

export async function editApiKey(
  envVarName: string,
  newValue: string
): Promise<ApiKey> {
  return apiPut<ApiKey>(`/admin/api-keys/${encodeURIComponent(envVarName)}`, {
    value: newValue,
  })
}

export async function deleteApiKey(envVarName: string): Promise<void> {
  const key =
    localStorage.getItem('pipimink-api-key') ?? ''
  const res = await fetch(
    `/admin/api-keys/${encodeURIComponent(envVarName)}`,
    {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        ...(key ? { 'X-API-Key': key } : {}),
      },
    }
  )
  if (!res.ok) {
    throw new Error(
      `DELETE /admin/api-keys/${envVarName}: ${res.status} ${res.statusText}`
    )
  }
}
