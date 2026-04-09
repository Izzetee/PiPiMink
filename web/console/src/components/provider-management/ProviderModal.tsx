import { useState } from 'react'
import type { Provider, ProviderType } from '@/types/provider-management'
import { X } from 'lucide-react'

interface ProviderModalProps {
  provider: Provider | null
  onSave: (data: {
    name: string
    type: ProviderType
    baseUrl: string
    apiKeyEnv: string
    timeout: string
    rateLimitSeconds: number | null
    enabled: boolean
  }) => void
  onClose: () => void
}

export function ProviderModal({ provider, onSave, onClose }: ProviderModalProps) {
  const [form, setForm] = useState({
    name: provider?.name ?? '',
    type: (provider?.type ?? 'openai-compatible') as ProviderType,
    baseUrl: provider?.baseUrl ?? '',
    apiKeyEnv: provider?.apiKeyEnv ?? '',
    timeout: provider?.timeout ?? '2m',
    rateLimitSeconds: provider?.rateLimitSeconds ?? null as number | null,
    enabled: provider?.enabled ?? true,
  })

  const isEdit = provider !== null
  const canSave = form.name.trim() && form.baseUrl.trim()

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (canSave) onSave(form)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 dark:bg-black/60"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative w-full max-w-md bg-white dark:bg-slate-800 rounded-xl shadow-xl border border-slate-200 dark:border-slate-700">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-100 dark:border-slate-700/50">
          <h2 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
            {isEdit ? 'Edit Provider' : 'Add Provider'}
          </h2>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:text-slate-500 dark:hover:text-slate-300 dark:hover:bg-slate-700 transition-colors"
          >
            <X className="w-4 h-4" strokeWidth={2} />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          {/* Name */}
          <div>
            <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1.5">
              Name <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="e.g. openai, az-foundry, local"
              className="w-full px-3 py-2 text-sm font-mono rounded-lg border border-slate-200 bg-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-900 dark:placeholder:text-slate-500 dark:text-slate-200 transition-colors"
              autoFocus
            />
          </div>

          {/* Type */}
          <div>
            <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1.5">
              Type
            </label>
            <select
              value={form.type}
              onChange={(e) =>
                setForm({ ...form, type: e.target.value as ProviderType })
              }
              className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-200 transition-colors"
            >
              <option value="openai-compatible">openai-compatible</option>
              <option value="anthropic">anthropic</option>
            </select>
          </div>

          {/* Base URL */}
          <div>
            <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1.5">
              Base URL <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              value={form.baseUrl}
              onChange={(e) => setForm({ ...form, baseUrl: e.target.value })}
              placeholder="https://api.example.com"
              className="w-full px-3 py-2 text-sm font-mono rounded-lg border border-slate-200 bg-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-900 dark:placeholder:text-slate-500 dark:text-slate-200 transition-colors"
            />
          </div>

          {/* API Key Env */}
          <div>
            <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1.5">
              API Key Environment Variable
            </label>
            <input
              type="text"
              value={form.apiKeyEnv}
              onChange={(e) => setForm({ ...form, apiKeyEnv: e.target.value })}
              placeholder="MY_API_KEY"
              className="w-full px-3 py-2 text-sm font-mono rounded-lg border border-slate-200 bg-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-900 dark:placeholder:text-slate-500 dark:text-slate-200 transition-colors"
            />
            <p className="mt-1 text-[11px] text-slate-400 dark:text-slate-500">
              Name of the environment variable holding the API key. Leave empty if none required.
            </p>
          </div>

          {/* Timeout + Rate Limit (side by side) */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1.5">
                Timeout
              </label>
              <input
                type="text"
                value={form.timeout}
                onChange={(e) => setForm({ ...form, timeout: e.target.value })}
                placeholder="2m"
                className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 bg-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-900 dark:placeholder:text-slate-500 dark:text-slate-200 transition-colors"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1.5">
                Rate Limit (seconds)
              </label>
              <input
                type="number"
                value={form.rateLimitSeconds ?? ''}
                onChange={(e) =>
                  setForm({
                    ...form,
                    rateLimitSeconds: e.target.value
                      ? Number(e.target.value)
                      : null,
                  })
                }
                placeholder="None"
                min={0}
                className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 bg-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-900 dark:placeholder:text-slate-500 dark:text-slate-200 transition-colors"
              />
            </div>
          </div>

          {/* Enabled toggle */}
          <div className="flex items-center justify-between py-1">
            <label className="text-xs font-medium text-slate-600 dark:text-slate-400">
              Enabled
            </label>
            <button
              type="button"
              onClick={() => setForm({ ...form, enabled: !form.enabled })}
              className={`relative w-9 h-5 rounded-full transition-colors ${
                form.enabled
                  ? 'bg-indigo-500 dark:bg-indigo-400'
                  : 'bg-slate-200 dark:bg-slate-600'
              }`}
            >
              <span
                className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow-sm transition-transform duration-200 ${
                  form.enabled ? 'translate-x-4' : 'translate-x-0'
                }`}
              />
            </button>
          </div>

          {/* Actions */}
          <div className="flex items-center justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-3.5 py-2 text-sm font-medium rounded-lg text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!canSave}
              className="px-3.5 py-2 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed dark:bg-indigo-500 dark:hover:bg-indigo-600 transition-colors shadow-sm"
            >
              {isEdit ? 'Save Changes' : 'Add Provider'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
