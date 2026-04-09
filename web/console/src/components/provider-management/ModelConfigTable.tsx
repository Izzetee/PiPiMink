import { useState } from 'react'
import type { ModelConfig, ProviderType } from '@/types/provider-management'
import {
  Plus,
  Check,
  X,
  Settings2,
  Globe,
  Key,
  Route,
  Server,
} from 'lucide-react'

interface ModelConfigTableProps {
  configs: ModelConfig[]
  onAdd: (config: Omit<ModelConfig, 'id'>) => void
  onEdit: (configId: string, updates: Partial<ModelConfig>) => void
  onToggle: (configId: string, enabled: boolean) => void
  onDelete: (configId: string) => void
}

const emptyConfig: Omit<ModelConfig, 'id'> = {
  name: '',
  chatPath: null,
  apiKeyEnv: '',
  type: null,
  baseUrl: null,
  enabled: true,
}

// --- Edit / Add form ---

function ConfigForm({
  initial,
  isNew,
  onSave,
  onCancel,
}: {
  initial: Omit<ModelConfig, 'id'>
  isNew: boolean
  onSave: (data: Omit<ModelConfig, 'id'>) => void
  onCancel: () => void
}) {
  const [form, setForm] = useState(initial)

  return (
    <div className="flex-1 p-4 space-y-3.5 overflow-y-auto">
      <div className="flex items-center justify-between mb-1">
        <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">
          {isNew ? 'New Model Config' : 'Edit Model Config'}
        </h4>
      </div>

      <div>
        <label className="block text-[11px] font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400 mb-1">
          Model Name <span className="text-red-400">*</span>
        </label>
        <input
          type="text"
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
          placeholder="e.g. Mistral-large-3"
          className="w-full px-2.5 py-1.5 text-sm font-mono rounded-md border border-slate-200 bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
          autoFocus
        />
      </div>

      <div>
        <label className="block text-[11px] font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400 mb-1">
          Type Override
        </label>
        <select
          value={form.type ?? ''}
          onChange={(e) =>
            setForm({
              ...form,
              type: (e.target.value || null) as ProviderType | null,
            })
          }
          className="w-full px-2.5 py-1.5 text-sm rounded-md border border-slate-200 bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        >
          <option value="">Inherit from provider</option>
          <option value="openai-compatible">openai-compatible</option>
          <option value="anthropic">anthropic</option>
        </select>
      </div>

      <div>
        <label className="block text-[11px] font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400 mb-1">
          Base URL
        </label>
        <input
          type="text"
          value={form.baseUrl ?? ''}
          onChange={(e) =>
            setForm({ ...form, baseUrl: e.target.value || null })
          }
          placeholder="Inherit from provider"
          className="w-full px-2.5 py-1.5 text-sm font-mono rounded-md border border-slate-200 bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        />
      </div>

      <div>
        <label className="block text-[11px] font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400 mb-1">
          Chat Path
        </label>
        <input
          type="text"
          value={form.chatPath ?? ''}
          onChange={(e) =>
            setForm({ ...form, chatPath: e.target.value || null })
          }
          placeholder="/models/chat/completions?api-version=..."
          className="w-full px-2.5 py-1.5 text-sm font-mono rounded-md border border-slate-200 bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        />
      </div>

      <div>
        <label className="block text-[11px] font-medium uppercase tracking-wider text-slate-500 dark:text-slate-400 mb-1">
          API Key Env
        </label>
        <input
          type="text"
          value={form.apiKeyEnv}
          onChange={(e) => setForm({ ...form, apiKeyEnv: e.target.value })}
          placeholder="ENV_VAR_NAME"
          className="w-full px-2.5 py-1.5 text-sm font-mono rounded-md border border-slate-200 bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200"
        />
      </div>

      <div className="flex items-center justify-end gap-2 pt-2">
        <button
          onClick={onCancel}
          className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
        >
          <X className="w-3 h-3" strokeWidth={2} />
          Cancel
        </button>
        <button
          onClick={() => form.name.trim() && onSave(form)}
          disabled={!form.name.trim()}
          className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-40 dark:bg-indigo-500 dark:hover:bg-indigo-600 transition-colors shadow-sm"
        >
          <Check className="w-3 h-3" strokeWidth={2} />
          Save
        </button>
      </div>
    </div>
  )
}

// --- Read-only detail view ---

function ConfigDetail({
  config,
  onEdit,
  onDelete,
}: {
  config: ModelConfig
  onEdit: () => void
  onDelete: () => void
}) {
  return (
    <div className="flex-1 p-4 overflow-y-auto">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2 min-w-0">
          <h4 className="text-sm font-mono font-semibold text-slate-800 dark:text-slate-200 truncate">
            {config.name}
          </h4>
          {config.type && (
            <span
              className={`shrink-0 px-1.5 py-0.5 text-[10px] font-medium rounded ${
                config.type === 'anthropic'
                  ? 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
                  : 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
              }`}
            >
              {config.type}
            </span>
          )}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          <button
            onClick={onEdit}
            className="px-2.5 py-1 text-[11px] font-medium rounded-md text-slate-500 hover:text-slate-700 hover:bg-slate-100 dark:text-slate-400 dark:hover:text-slate-200 dark:hover:bg-slate-700 transition-colors"
          >
            Edit
          </button>
          <button
            onClick={onDelete}
            className="px-2.5 py-1 text-[11px] font-medium rounded-md text-slate-400 hover:text-red-500 hover:bg-red-50 dark:text-slate-500 dark:hover:text-red-400 dark:hover:bg-red-950/30 transition-colors"
          >
            Delete
          </button>
        </div>
      </div>

      <div className="space-y-4">
        <DetailField
          icon={Globe}
          label="Base URL"
          value={config.baseUrl}
          fallback="Inherit from provider"
          mono
        />
        <DetailField
          icon={Route}
          label="Chat Path"
          value={config.chatPath}
          fallback="Default"
          mono
        />
        <DetailField
          icon={Key}
          label="API Key Env"
          value={config.apiKeyEnv || null}
          fallback="—"
          mono
        />
        <DetailField
          icon={Server}
          label="Type Override"
          value={config.type}
          fallback="Inherit from provider"
        />
      </div>
    </div>
  )
}

function DetailField({
  icon: Icon,
  label,
  value,
  fallback,
  mono = false,
}: {
  icon: typeof Globe
  label: string
  value: string | null
  fallback: string
  mono?: boolean
}) {
  return (
    <div className="flex items-start gap-2.5">
      <Icon
        className="w-3.5 h-3.5 text-slate-400 dark:text-slate-500 mt-0.5 shrink-0"
        strokeWidth={1.75}
      />
      <div className="min-w-0">
        <p className="text-[10px] font-medium uppercase tracking-wider text-slate-400 dark:text-slate-500">
          {label}
        </p>
        <p
          className={`text-sm mt-0.5 break-all ${
            mono ? 'font-mono' : ''
          } ${
            value
              ? 'text-slate-800 dark:text-slate-200'
              : 'text-slate-400 dark:text-slate-500 italic'
          }`}
        >
          {value || fallback}
        </p>
      </div>
    </div>
  )
}

// --- Main component ---

export function ModelConfigTable({
  configs,
  onAdd,
  onEdit,
  onToggle,
  onDelete,
}: ModelConfigTableProps) {
  const [selectedId, setSelectedId] = useState<string | null>(
    configs[0]?.id ?? null
  )
  const [adding, setAdding] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)

  const selectedConfig = configs.find((c) => c.id === selectedId) ?? null

  return (
    <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
      {/* Header */}
      <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-700/50 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Settings2
            className="w-3.5 h-3.5 text-amber-500 dark:text-amber-400"
            strokeWidth={2}
          />
          <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">
            Model Configurations
          </h3>
          <span className="text-[10px] tabular-nums text-slate-400 dark:text-slate-500">
            ({configs.length})
          </span>
        </div>
        <button
          onClick={() => {
            setAdding(true)
            setEditingId(null)
          }}
          className="inline-flex items-center gap-1 px-2 py-1 text-[11px] font-medium rounded-md text-indigo-600 hover:bg-indigo-50 dark:text-indigo-400 dark:hover:bg-indigo-950/30 transition-colors"
        >
          <Plus className="w-3 h-3" strokeWidth={2} />
          Add
        </button>
      </div>

      {/* Master-detail split */}
      <div className="flex min-h-[200px]">
        {/* Left: config list */}
        <div className="w-48 shrink-0 border-r border-slate-100 dark:border-slate-700/50 overflow-y-auto">
          {configs.map((config) => {
            const isSelected = config.id === selectedId && !adding
            return (
              <button
                key={config.id}
                onClick={() => {
                  setSelectedId(config.id)
                  setAdding(false)
                  setEditingId(null)
                }}
                className={`w-full text-left px-3 py-2.5 flex items-center gap-2 transition-colors border-l-2 ${
                  isSelected
                    ? 'bg-indigo-50/60 dark:bg-indigo-950/30 border-l-indigo-500 dark:border-l-indigo-400'
                    : 'border-l-transparent hover:bg-slate-50 dark:hover:bg-slate-700/30'
                }`}
              >
                {/* Mini toggle */}
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    onToggle(config.id, !config.enabled)
                  }}
                  className={`relative w-6 h-3.5 rounded-full transition-colors shrink-0 ${
                    config.enabled
                      ? 'bg-indigo-500 dark:bg-indigo-400'
                      : 'bg-slate-200 dark:bg-slate-600'
                  }`}
                >
                  <span
                    className={`absolute top-0.5 left-0.5 w-2.5 h-2.5 rounded-full bg-white shadow-sm transition-transform duration-200 ${
                      config.enabled ? 'translate-x-2.5' : 'translate-x-0'
                    }`}
                  />
                </button>

                <span
                  className={`text-xs font-mono truncate ${
                    config.enabled
                      ? isSelected
                        ? 'text-indigo-700 dark:text-indigo-300 font-medium'
                        : 'text-slate-700 dark:text-slate-300'
                      : 'text-slate-400 dark:text-slate-500'
                  }`}
                >
                  {config.name}
                </span>
              </button>
            )
          })}

          {configs.length === 0 && !adding && (
            <div className="px-3 py-6 text-center">
              <p className="text-[11px] text-slate-400 dark:text-slate-500">
                No configs yet
              </p>
            </div>
          )}
        </div>

        {/* Right: detail / form */}
        {adding ? (
          <ConfigForm
            initial={emptyConfig}
            isNew
            onSave={(data) => {
              onAdd(data)
              setAdding(false)
            }}
            onCancel={() => setAdding(false)}
          />
        ) : editingId && selectedConfig ? (
          <ConfigForm
            initial={{
              name: selectedConfig.name,
              chatPath: selectedConfig.chatPath,
              apiKeyEnv: selectedConfig.apiKeyEnv,
              type: selectedConfig.type,
              baseUrl: selectedConfig.baseUrl,
              enabled: selectedConfig.enabled,
            }}
            isNew={false}
            onSave={(data) => {
              onEdit(selectedConfig.id, data)
              setEditingId(null)
            }}
            onCancel={() => setEditingId(null)}
          />
        ) : selectedConfig ? (
          <ConfigDetail
            config={selectedConfig}
            onEdit={() => setEditingId(selectedConfig.id)}
            onDelete={() => {
              onDelete(selectedConfig.id)
              const remaining = configs.filter((c) => c.id !== selectedConfig.id)
              setSelectedId(remaining[0]?.id ?? null)
            }}
          />
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <p className="text-xs text-slate-400 dark:text-slate-500">
              {configs.length > 0
                ? 'Select a model config'
                : 'Add a model config to get started'}
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
