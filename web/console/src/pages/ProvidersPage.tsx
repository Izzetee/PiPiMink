import { useEffect, useState, useCallback } from 'react'
import { ProviderManagement } from '@/components/provider-management'
import {
  fetchProviders,
  addProvider,
  updateProvider,
  deleteProvider,
  toggleProvider,
  testProvider,
  updateModelConfigs,
} from '@/api'
import type { Provider, ModelConfig } from '@/types/provider-management'
import { Loader2, AlertCircle, Key } from 'lucide-react'
import { getApiKey, setApiKey } from '@/api'

export function ProvidersPage() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [apiKeyInput, setApiKeyInput] = useState('')
  const [showApiKeyPrompt, setShowApiKeyPrompt] = useState(false)

  const loadProviders = useCallback(async () => {
    try {
      setError(null)
      const data = await fetchProviders()
      setProviders(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load providers')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadProviders()
  }, [loadProviders])

  useEffect(() => {
    if (!getApiKey()) setShowApiKeyPrompt(true)
  }, [])

  const handleAddProvider = useCallback(
    async (data: Omit<Provider, 'id' | 'modelCount' | 'models' | 'modelConfigs' | 'lastTestedAt' | 'lastTestResult' | 'lastTestLatencyMs'>) => {
      try {
        await addProvider({
          name: data.name,
          type: data.type,
          baseUrl: data.baseUrl,
          apiKeyEnv: data.apiKeyEnv,
          timeout: data.timeout,
          rateLimitSeconds: data.rateLimitSeconds,
          enabled: data.enabled,
        })
        await loadProviders()
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to add provider')
      }
    },
    [loadProviders]
  )

  const handleEditProvider = useCallback(
    async (id: string, updates: Partial<Provider>) => {
      try {
        await updateProvider(id, {
          type: updates.type,
          baseUrl: updates.baseUrl,
          apiKeyEnv: updates.apiKeyEnv,
          timeout: updates.timeout,
          rateLimitSeconds: updates.rateLimitSeconds,
          enabled: updates.enabled,
        })
        await loadProviders()
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update provider')
      }
    },
    [loadProviders]
  )

  const handleDeleteProvider = useCallback(
    async (id: string) => {
      try {
        await deleteProvider(id)
        setProviders((prev) => prev.filter((p) => p.id !== id))
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to delete provider')
      }
    },
    []
  )

  const handleDuplicateProvider = useCallback(
    async (id: string) => {
      const provider = providers.find((p) => p.id === id)
      if (!provider) return
      try {
        await addProvider({
          name: `${provider.name}-copy`,
          type: provider.type,
          baseUrl: provider.baseUrl,
          apiKeyEnv: provider.apiKeyEnv,
          timeout: provider.timeout,
          rateLimitSeconds: provider.rateLimitSeconds,
          enabled: provider.enabled,
        })
        await loadProviders()
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to duplicate provider')
      }
    },
    [providers, loadProviders]
  )

  const handleToggleProvider = useCallback(
    async (id: string, enabled: boolean) => {
      // Optimistic update
      setProviders((prev) =>
        prev.map((p) => (p.id === id ? { ...p, enabled } : p))
      )
      try {
        await toggleProvider(id, enabled)
      } catch {
        loadProviders()
      }
    },
    [loadProviders]
  )

  const handleToggleModelConfigs = useCallback(
    (id: string, hasModelConfigs: boolean) => {
      setProviders((prev) =>
        prev.map((p) =>
          p.id === id
            ? {
                ...p,
                hasModelConfigs,
                modelConfigs: hasModelConfigs ? p.modelConfigs : [],
              }
            : p
        )
      )
      if (!hasModelConfigs) {
        // Clear model configs on the backend
        updateModelConfigs(id, []).catch(() => loadProviders())
      }
    },
    [loadProviders]
  )

  const handleTestConnection = useCallback(
    async (id: string) => {
      try {
        const result = await testProvider(id)
        setProviders((prev) =>
          prev.map((p) =>
            p.id === id
              ? {
                  ...p,
                  lastTestResult: result.result,
                  lastTestLatencyMs: result.latencyMs,
                  lastTestedAt: new Date().toISOString(),
                }
              : p
          )
        )
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Connection test failed')
      }
    },
    []
  )

  const handleAddModelConfig = useCallback(
    async (providerId: string, config: Omit<ModelConfig, 'id'>) => {
      const provider = providers.find((p) => p.id === providerId)
      if (!provider) return
      const newConfig: ModelConfig = { ...config, id: config.name }
      const updatedConfigs = [...provider.modelConfigs, newConfig]
      setProviders((prev) =>
        prev.map((p) =>
          p.id === providerId ? { ...p, modelConfigs: updatedConfigs } : p
        )
      )
      try {
        await updateModelConfigs(providerId, updatedConfigs)
      } catch {
        loadProviders()
      }
    },
    [providers, loadProviders]
  )

  const handleEditModelConfig = useCallback(
    async (providerId: string, configId: string, updates: Partial<ModelConfig>) => {
      const provider = providers.find((p) => p.id === providerId)
      if (!provider) return
      const updatedConfigs = provider.modelConfigs.map((c) =>
        c.id === configId ? { ...c, ...updates } : c
      )
      setProviders((prev) =>
        prev.map((p) =>
          p.id === providerId ? { ...p, modelConfigs: updatedConfigs } : p
        )
      )
      try {
        await updateModelConfigs(providerId, updatedConfigs)
      } catch {
        loadProviders()
      }
    },
    [providers, loadProviders]
  )

  const handleToggleModelConfig = useCallback(
    async (providerId: string, configId: string, enabled: boolean) => {
      const provider = providers.find((p) => p.id === providerId)
      if (!provider) return
      const updatedConfigs = provider.modelConfigs.map((c) =>
        c.id === configId ? { ...c, enabled } : c
      )
      setProviders((prev) =>
        prev.map((p) =>
          p.id === providerId ? { ...p, modelConfigs: updatedConfigs } : p
        )
      )
      try {
        await updateModelConfigs(providerId, updatedConfigs)
      } catch {
        loadProviders()
      }
    },
    [providers, loadProviders]
  )

  const handleDeleteModelConfig = useCallback(
    async (providerId: string, configId: string) => {
      const provider = providers.find((p) => p.id === providerId)
      if (!provider) return
      const updatedConfigs = provider.modelConfigs.filter((c) => c.id !== configId)
      setProviders((prev) =>
        prev.map((p) =>
          p.id === providerId ? { ...p, modelConfigs: updatedConfigs } : p
        )
      )
      try {
        await updateModelConfigs(providerId, updatedConfigs)
      } catch {
        loadProviders()
      }
    },
    [providers, loadProviders]
  )

  const handleSaveApiKey = () => {
    setApiKey(apiKeyInput.trim())
    setShowApiKeyPrompt(false)
    setApiKeyInput('')
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-6 h-6 text-indigo-500 animate-spin" strokeWidth={2} />
      </div>
    )
  }

  return (
    <>
      {/* API Key prompt */}
      {showApiKeyPrompt && (
        <div className="mx-5 mt-5 lg:mx-6 lg:mt-6 rounded-xl border border-amber-200 bg-amber-50/60 dark:border-amber-500/20 dark:bg-amber-950/30 px-4 py-3">
          <div className="flex items-start gap-3">
            <Key className="w-4 h-4 text-amber-600 dark:text-amber-400 mt-0.5 shrink-0" strokeWidth={1.75} />
            <div className="flex-1">
              <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
                Admin API key required
              </p>
              <p className="text-xs text-amber-600/80 dark:text-amber-400/60 mt-0.5 mb-2">
                Set your ADMIN_API_KEY to manage providers.
              </p>
              <div className="flex gap-2">
                <input
                  type="password"
                  placeholder="Enter API key..."
                  value={apiKeyInput}
                  onChange={(e) => setApiKeyInput(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleSaveApiKey()}
                  className="flex-1 max-w-xs px-3 py-1.5 text-sm rounded-lg border border-amber-200 bg-white dark:border-amber-600/30 dark:bg-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-amber-500/20"
                />
                <button
                  onClick={handleSaveApiKey}
                  className="px-3 py-1.5 text-sm font-medium rounded-lg bg-amber-600 text-white hover:bg-amber-700 dark:bg-amber-500 dark:hover:bg-amber-600 transition-colors"
                >
                  Save
                </button>
                <button
                  onClick={() => setShowApiKeyPrompt(false)}
                  className="px-3 py-1.5 text-sm font-medium rounded-lg text-amber-700 hover:bg-amber-100 dark:text-amber-400 dark:hover:bg-amber-900/30 transition-colors"
                >
                  Skip
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Error banner */}
      {error && (
        <div className="mx-5 mt-5 lg:mx-6 lg:mt-6 rounded-xl border border-red-200 bg-red-50/60 dark:border-red-500/20 dark:bg-red-950/30 px-4 py-3">
          <div className="flex items-center gap-2">
            <AlertCircle className="w-4 h-4 text-red-500 dark:text-red-400 shrink-0" strokeWidth={1.75} />
            <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
            <button
              onClick={() => setError(null)}
              className="ml-auto text-xs text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
            >
              Dismiss
            </button>
          </div>
        </div>
      )}

      <ProviderManagement
        providers={providers}
        onAddProvider={handleAddProvider}
        onEditProvider={handleEditProvider}
        onDeleteProvider={handleDeleteProvider}
        onDuplicateProvider={handleDuplicateProvider}
        onToggleProvider={handleToggleProvider}
        onToggleModelConfigs={handleToggleModelConfigs}
        onTestConnection={handleTestConnection}
        onAddModelConfig={handleAddModelConfig}
        onEditModelConfig={handleEditModelConfig}
        onToggleModelConfig={handleToggleModelConfig}
        onDeleteModelConfig={handleDeleteModelConfig}
      />
    </>
  )
}
