import { useState } from 'react'
import type { ApiKey, ProviderOption } from '@/types/settings'
import {
  KeyRound,
  Plus,
  Pencil,
  Trash2,
  X,
  Shield,
  Clock,
  Eye,
  EyeOff,
  ChevronDown,
} from 'lucide-react'

interface ApiKeyVaultProps {
  apiKeys: ApiKey[]
  providerOptions: ProviderOption[]
  onAdd?: (envVarName: string, value: string) => void
  onEdit?: (id: string, newValue: string) => void
  onDelete?: (id: string) => void
}

export function ApiKeyVault({
  apiKeys,
  providerOptions,
  onAdd,
  onEdit,
  onDelete,
}: ApiKeyVaultProps) {
  const [addModalOpen, setAddModalOpen] = useState(false)
  const [addModalPreselect, setAddModalPreselect] = useState('')
  const [editingKey, setEditingKey] = useState<ApiKey | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<ApiKey | null>(null)

  // Build a lookup of stored keys by env var name
  const storedEnvNames = new Set(apiKeys.map((k) => k.envVarName))

  // Collect all available (unstored) env var names for the add modal
  const availableEnvVars = providerOptions.flatMap((p) =>
    p.apiKeyEnvVars
      .filter((env) => !storedEnvNames.has(env))
      .map((env) => ({ envVarName: env, providerName: p.name, providerId: p.id }))
  )

  // Build a per-provider view: stored keys + missing keys
  const keysByEnvName = new Map(apiKeys.map((k) => [k.envVarName, k]))

  const providerSections = providerOptions.map((provider) => {
    const stored = provider.apiKeyEnvVars
      .filter((env) => keysByEnvName.has(env))
      .map((env) => keysByEnvName.get(env)!)
    const missing = provider.apiKeyEnvVars.filter((env) => !keysByEnvName.has(env))
    return { provider, stored, missing }
  })

  // Only show providers that have at least one env var defined
  const visibleProviders = providerSections.filter(
    (s) => s.provider.apiKeyEnvVars.length > 0
  )

  const totalMissing = visibleProviders.reduce((n, s) => n + s.missing.length, 0)

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div>
          <p className="text-sm text-slate-500 dark:text-slate-400">
            Securely store API keys referenced by your provider configurations.
          </p>
        </div>
        {totalMissing > 0 && (
          <span className="text-xs text-amber-600 dark:text-amber-400 font-medium">
            {totalMissing} key{totalMissing !== 1 ? 's' : ''} missing
          </span>
        )}
      </div>

      {visibleProviders.length === 0 ? (
        <div className="py-16 text-center rounded-xl border-2 border-dashed border-slate-200 dark:border-slate-700">
          <Shield
            className="w-10 h-10 text-slate-300 dark:text-slate-600 mx-auto mb-3"
            strokeWidth={1}
          />
          <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
            No providers configured
          </p>
          <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">
            Add providers first, then store their API keys here
          </p>
        </div>
      ) : (
        <div className="space-y-6">
          {visibleProviders.map(({ provider, stored, missing }) => (
            <div key={provider.id}>
              <div className="flex items-center gap-2 mb-2">
                <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">
                  {provider.name}
                </h4>
                {missing.length > 0 && stored.length > 0 && (
                  <span className="inline-flex items-center px-1.5 py-0.5 text-[10px] font-semibold rounded bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400">
                    {missing.length} missing
                  </span>
                )}
                {stored.length === 0 && (
                  <span className="inline-flex items-center px-1.5 py-0.5 text-[10px] font-semibold rounded bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400">
                    No keys
                  </span>
                )}
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
                {/* Stored keys */}
                {stored.map((apiKey) => (
                  <div
                    key={apiKey.id}
                    className="group relative rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 hover:border-slate-300 dark:hover:border-slate-600 transition-colors cursor-pointer"
                    onClick={() => setEditingKey(apiKey)}
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex items-center gap-2">
                        <div className="w-8 h-8 rounded-lg bg-amber-50 dark:bg-amber-900/20 flex items-center justify-center">
                          <KeyRound
                            className="w-4 h-4 text-amber-600 dark:text-amber-400"
                            strokeWidth={1.5}
                          />
                        </div>
                        <span className="text-xs font-mono font-medium text-slate-700 dark:text-slate-300 break-all">
                          {apiKey.envVarName}
                        </span>
                      </div>
                      <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            setEditingKey(apiKey)
                          }}
                          className="p-1 rounded-md text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                        >
                          <Pencil className="w-3.5 h-3.5" strokeWidth={1.5} />
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            setDeleteTarget(apiKey)
                          }}
                          className="p-1 rounded-md text-slate-400 hover:text-red-600 dark:hover:text-red-400 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                        >
                          <Trash2 className="w-3.5 h-3.5" strokeWidth={1.5} />
                        </button>
                      </div>
                    </div>

                    <div className="font-mono text-xs text-slate-400 dark:text-slate-500 tracking-wider mb-3">
                      {apiKey.maskedValue}
                    </div>

                    <div className="flex items-center gap-1 text-[10px] text-slate-400 dark:text-slate-500">
                      <Clock className="w-3 h-3" strokeWidth={1.5} />
                      Updated{' '}
                      {new Date(apiKey.lastUpdatedAt).toLocaleDateString('en-US', {
                        month: 'short',
                        day: 'numeric',
                        year: 'numeric',
                      })}
                    </div>
                  </div>
                ))}

                {/* Missing keys — placeholder cards */}
                {missing.map((envVarName) => (
                  <button
                    key={envVarName}
                    onClick={() => {
                      setAddModalPreselect(envVarName)
                      setAddModalOpen(true)
                    }}
                    className="rounded-xl border-2 border-dashed border-slate-200 dark:border-slate-700 p-4 hover:border-indigo-300 dark:hover:border-indigo-600 hover:bg-indigo-50/50 dark:hover:bg-indigo-900/10 transition-colors text-left group"
                  >
                    <div className="flex items-center gap-2 mb-3">
                      <div className="w-8 h-8 rounded-lg bg-slate-100 dark:bg-slate-700 flex items-center justify-center group-hover:bg-indigo-100 dark:group-hover:bg-indigo-900/30 transition-colors">
                        <KeyRound
                          className="w-4 h-4 text-slate-300 dark:text-slate-500 group-hover:text-indigo-500 dark:group-hover:text-indigo-400 transition-colors"
                          strokeWidth={1.5}
                        />
                      </div>
                      <span className="text-xs font-mono font-medium text-slate-400 dark:text-slate-500 break-all">
                        {envVarName}
                      </span>
                    </div>

                    <div className="flex items-center gap-1.5 text-xs text-slate-400 dark:text-slate-500 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                      <Plus className="w-3.5 h-3.5" strokeWidth={2} />
                      Add key
                    </div>
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Add modal */}
      {addModalOpen && (
        <AddKeyModal
          availableEnvVars={availableEnvVars}
          preselectEnvVar={addModalPreselect}
          onSave={(envVarName, value) => {
            onAdd?.(envVarName, value)
            setAddModalOpen(false)
            setAddModalPreselect('')
          }}
          onClose={() => {
            setAddModalOpen(false)
            setAddModalPreselect('')
          }}
        />
      )}

      {/* Edit modal */}
      {editingKey && (
        <EditKeyModal
          apiKey={editingKey}
          onSave={(newValue) => {
            onEdit?.(editingKey.id, newValue)
            setEditingKey(null)
          }}
          onClose={() => setEditingKey(null)}
        />
      )}

      {/* Delete confirmation */}
      {deleteTarget && (
        <DeleteKeyDialog
          apiKey={deleteTarget}
          onConfirm={() => {
            onDelete?.(deleteTarget.id)
            setDeleteTarget(null)
          }}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}

/* ── Add Key Modal ───────────────────────────────────────────────── */

function AddKeyModal({
  availableEnvVars,
  preselectEnvVar = '',
  onSave,
  onClose,
}: {
  availableEnvVars: { envVarName: string; providerName: string; providerId: string }[]
  preselectEnvVar?: string
  onSave: (envVarName: string, value: string) => void
  onClose: () => void
}) {
  const [selectedEnv, setSelectedEnv] = useState(preselectEnvVar)
  const [value, setValue] = useState('')
  const [dropdownOpen, setDropdownOpen] = useState(false)

  const selected = availableEnvVars.find((e) => e.envVarName === selectedEnv)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="w-full max-w-md mx-4 rounded-xl bg-white dark:bg-slate-800 shadow-2xl border border-slate-200 dark:border-slate-700">
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-100 dark:border-slate-700">
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
            Add API Key
          </h3>
          <button
            onClick={onClose}
            className="p-1 rounded-md text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            <X className="w-4 h-4" strokeWidth={1.5} />
          </button>
        </div>

        <div className="p-5 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              Environment Variable
            </label>
            <div className="relative">
              <button
                onClick={() => setDropdownOpen(!dropdownOpen)}
                className="w-full flex items-center justify-between px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 hover:border-slate-300 dark:hover:border-slate-500 transition-colors"
              >
                <span className={selectedEnv ? 'font-mono text-xs' : 'text-slate-400 dark:text-slate-500'}>
                  {selectedEnv || 'Select env var...'}
                </span>
                <ChevronDown className="w-4 h-4 text-slate-400" strokeWidth={1.5} />
              </button>
              {dropdownOpen && (
                <div className="absolute z-10 mt-1 w-full rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 shadow-lg overflow-hidden">
                  <div className="max-h-56 overflow-y-auto py-1">
                    {availableEnvVars.map((env) => (
                      <button
                        key={env.envVarName}
                        onClick={() => {
                          setSelectedEnv(env.envVarName)
                          setDropdownOpen(false)
                        }}
                        className="w-full px-3 py-2 text-left hover:bg-slate-50 dark:hover:bg-slate-600 transition-colors"
                      >
                        <span className="block text-xs font-mono font-medium text-slate-700 dark:text-slate-200">
                          {env.envVarName}
                        </span>
                        <span className="block text-[10px] text-slate-400 dark:text-slate-500">
                          {env.providerName}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
            {selected && (
              <p className="mt-1 text-xs text-slate-400 dark:text-slate-500">
                Provider: {selected.providerName}
              </p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              API Key Value
            </label>
            <input
              type="password"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder="Paste your API key..."
              className="w-full px-3 py-2 text-sm font-mono rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 transition-colors"
            />
          </div>
        </div>

        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-slate-100 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50 rounded-b-xl">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-sm font-medium rounded-lg text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={() => onSave(selectedEnv, value)}
            disabled={!selectedEnv || !value}
            className="px-4 py-1.5 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 transition-colors shadow-sm disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Store Key
          </button>
        </div>
      </div>
    </div>
  )
}

/* ── Edit Key Modal ──────────────────────────────────────────────── */

function EditKeyModal({
  apiKey,
  onSave,
  onClose,
}: {
  apiKey: ApiKey
  onSave: (newValue: string) => void
  onClose: () => void
}) {
  const [value, setValue] = useState('')
  const [showValue, setShowValue] = useState(false)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="w-full max-w-md mx-4 rounded-xl bg-white dark:bg-slate-800 shadow-2xl border border-slate-200 dark:border-slate-700">
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-100 dark:border-slate-700">
          <div>
            <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
              Update API Key
            </h3>
            <p className="text-xs font-mono text-slate-400 dark:text-slate-500 mt-0.5">
              {apiKey.envVarName}
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-md text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            <X className="w-4 h-4" strokeWidth={1.5} />
          </button>
        </div>

        <div className="p-5 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Current Value
            </label>
            <div className="px-3 py-2 text-sm font-mono text-slate-400 dark:text-slate-500 bg-slate-50 dark:bg-slate-700/50 rounded-lg border border-slate-100 dark:border-slate-700">
              {apiKey.maskedValue}
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              New Value
            </label>
            <div className="relative">
              <input
                type={showValue ? 'text' : 'password'}
                value={value}
                onChange={(e) => setValue(e.target.value)}
                placeholder="Paste new API key..."
                className="w-full px-3 py-2 pr-10 text-sm font-mono rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 transition-colors"
              />
              <button
                onClick={() => setShowValue(!showValue)}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
              >
                {showValue ? (
                  <EyeOff className="w-4 h-4" strokeWidth={1.5} />
                ) : (
                  <Eye className="w-4 h-4" strokeWidth={1.5} />
                )}
              </button>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-slate-100 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50 rounded-b-xl">
          <button
            onClick={onClose}
            className="px-3 py-1.5 text-sm font-medium rounded-lg text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={() => onSave(value)}
            disabled={!value}
            className="px-4 py-1.5 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 transition-colors shadow-sm disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Update Key
          </button>
        </div>
      </div>
    </div>
  )
}

/* ── Delete Confirmation ─────────────────────────────────────────── */

function DeleteKeyDialog({
  apiKey,
  onConfirm,
  onCancel,
}: {
  apiKey: ApiKey
  onConfirm: () => void
  onCancel: () => void
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="w-full max-w-sm mx-4 rounded-xl bg-white dark:bg-slate-800 shadow-2xl border border-slate-200 dark:border-slate-700 p-5">
        <div className="flex items-center gap-3 mb-3">
          <div className="w-10 h-10 rounded-full bg-red-50 dark:bg-red-900/20 flex items-center justify-center">
            <Trash2
              className="w-5 h-5 text-red-600 dark:text-red-400"
              strokeWidth={1.5}
            />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
              Delete API Key
            </h3>
            <p className="text-xs text-slate-400 dark:text-slate-500">
              This cannot be undone
            </p>
          </div>
        </div>
        <p className="text-sm text-slate-600 dark:text-slate-400 mb-4">
          Are you sure you want to delete{' '}
          <span className="font-mono text-xs font-medium text-slate-800 dark:text-slate-200">
            {apiKey.envVarName}
          </span>
          ? Any providers referencing this key will lose access.
        </p>
        <div className="flex items-center justify-end gap-2">
          <button
            onClick={onCancel}
            className="px-3 py-1.5 text-sm font-medium rounded-lg text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className="px-4 py-1.5 text-sm font-medium rounded-lg bg-red-600 text-white hover:bg-red-700 dark:bg-red-500 dark:hover:bg-red-600 transition-colors shadow-sm"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  )
}
