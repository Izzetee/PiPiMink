import { useEffect, useState, useCallback } from 'react'
import { Settings } from '@/components/settings'
import {
  fetchSettings,
  patchSettings,
  fetchApiKeys,
  addApiKey,
  editApiKey,
  deleteApiKey,
  getApiKey,
  setApiKey,
  fetchAdminStatus,
} from '@/api'
import type {
  SettingsMap,
  ProviderOption,
  ApiKey as ApiKeyType,
  PendingChange,
  SettingCategory,
  Setting,
} from '@/types/settings'
import { Loader2, AlertCircle, Key } from 'lucide-react'

const SETTING_CATEGORIES: SettingCategory[] = [
  'routing',
  'cache',
  'database',
  'server',
  'benchmarking',
  'observability',
]

export function SettingsPage() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [settings, setSettings] = useState<SettingsMap>({
    routing: [],
    cache: [],
    database: [],
    server: [],
    benchmarking: [],
    observability: [],
  })
  const [serverSettings, setServerSettings] = useState<SettingsMap>({
    routing: [],
    cache: [],
    database: [],
    server: [],
    benchmarking: [],
    observability: [],
  })
  const [providerOptions, setProviderOptions] = useState<ProviderOption[]>([])
  const [apiKeys, setApiKeys] = useState<ApiKeyType[]>([])
  const [pendingChanges, setPendingChanges] = useState<PendingChange[]>([])
  const [apiKeyInput, setApiKeyInput] = useState('')
  const [showApiKeyPrompt, setShowApiKeyPrompt] = useState(false)

  const loadAll = useCallback(async () => {
    try {
      setError(null)
      const [settingsData, keysData] = await Promise.all([
        fetchSettings(),
        fetchApiKeys(),
      ])
      setSettings(settingsData.settings)
      setServerSettings(settingsData.settings)
      setProviderOptions(settingsData.providerOptions)
      setApiKeys(keysData)
      setPendingChanges([])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load settings')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadAll()
  }, [loadAll])

  useEffect(() => {
    // Only prompt for an API key if the server has one configured but localStorage
    // doesn't have it. When no key is configured server-side (first-run / passthrough
    // mode), skip the prompt — the server allows unauthenticated access.
    if (!getApiKey()) {
      fetchAdminStatus()
        .then((status) => {
          if (status.adminKeyConfigured) setShowApiKeyPrompt(true)
        })
        .catch(() => {
          // Fallback: show prompt if status endpoint is unreachable
          setShowApiKeyPrompt(true)
        })
    }
  }, [])

  // Find which category a setting key belongs to
  const findCategory = useCallback(
    (key: string): SettingCategory | undefined => {
      for (const cat of SETTING_CATEGORIES) {
        if (settings[cat].some((s) => s.key === key)) return cat
      }
      return undefined
    },
    [settings]
  )

  const handleSettingChange = useCallback(
    (key: string, value: string | number | boolean) => {
      const category = findCategory(key)
      if (!category) return

      // Update local settings state
      setSettings((prev) => ({
        ...prev,
        [category]: prev[category].map((s: Setting) =>
          s.key === key ? { ...s, value } : s
        ),
      }))

      // Find previous value from server state
      const serverSetting = serverSettings[category].find(
        (s: Setting) => s.key === key
      )
      const previousValue = serverSetting?.value ?? ''

      // Update pending changes
      setPendingChanges((prev) => {
        const existing = prev.filter((c) => c.key !== key)
        // Only add if different from server value
        if (value !== previousValue) {
          existing.push({ key, category, previousValue, newValue: value })
        }
        return existing
      })
    },
    [findCategory, serverSettings]
  )

  const handleSaveAll = useCallback(
    async (changes: PendingChange[]) => {
      try {
        setError(null)
        const result = await patchSettings(changes)
        setSettings(result.settings)
        setServerSettings(result.settings)
        setProviderOptions(result.providerOptions)
        setPendingChanges([])
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to save settings')
      }
    },
    []
  )

  const handleDiscardAll = useCallback(() => {
    setSettings(serverSettings)
    setPendingChanges([])
  }, [serverSettings])

  const handleAddApiKey = useCallback(
    async (envVarName: string, value: string) => {
      try {
        setError(null)
        await addApiKey(envVarName, value)
        const keys = await fetchApiKeys()
        setApiKeys(keys)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to add API key')
      }
    },
    []
  )

  const handleEditApiKey = useCallback(
    async (id: string, newValue: string) => {
      try {
        setError(null)
        await editApiKey(id, newValue)
        const keys = await fetchApiKeys()
        setApiKeys(keys)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update API key')
      }
    },
    []
  )

  const handleDeleteApiKey = useCallback(
    async (id: string) => {
      try {
        setError(null)
        await deleteApiKey(id)
        const keys = await fetchApiKeys()
        setApiKeys(keys)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to delete API key')
      }
    },
    []
  )

  const handleSaveApiKey = () => {
    setApiKey(apiKeyInput.trim())
    setShowApiKeyPrompt(false)
    setApiKeyInput('')
    loadAll()
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2
          className="w-6 h-6 text-indigo-500 animate-spin"
          strokeWidth={2}
        />
      </div>
    )
  }

  return (
    <>
      {/* API Key prompt */}
      {showApiKeyPrompt && (
        <div className="mx-5 mt-5 lg:mx-6 lg:mt-6 rounded-xl border border-amber-200 bg-amber-50/60 dark:border-amber-500/20 dark:bg-amber-950/30 px-4 py-3">
          <div className="flex items-start gap-3">
            <Key
              className="w-4 h-4 text-amber-600 dark:text-amber-400 mt-0.5 shrink-0"
              strokeWidth={1.75}
            />
            <div className="flex-1">
              <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
                Admin API key required
              </p>
              <p className="text-xs text-amber-600/80 dark:text-amber-400/60 mt-0.5 mb-2">
                Set your ADMIN_API_KEY to manage settings and API keys.
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
            <AlertCircle
              className="w-4 h-4 text-red-500 dark:text-red-400 shrink-0"
              strokeWidth={1.75}
            />
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

      <Settings
        settings={settings}
        apiKeys={apiKeys}
        providerOptions={providerOptions}
        pendingChanges={pendingChanges}
        onSettingChange={handleSettingChange}
        onSaveAll={handleSaveAll}
        onDiscardAll={handleDiscardAll}
        onAddApiKey={handleAddApiKey}
        onEditApiKey={handleEditApiKey}
        onDeleteApiKey={handleDeleteApiKey}
      />
    </>
  )
}
