import { useState } from 'react'
import type { Provider, ModelConfig } from '@/types/provider-management'
import { ModelConfigTable } from './ModelConfigTable'
import {
  Pencil,
  Trash2,
  Copy,
  Zap,
  CheckCircle2,
  XCircle,
  Loader2,
  Globe,
  Key,
  Timer,
  Gauge,
  Server,
  Hash,
} from 'lucide-react'

interface ProviderDetailProps {
  provider: Provider
  onEdit: () => void
  onDelete: () => void
  onDuplicate: () => void
  onToggle: (enabled: boolean) => void
  onToggleModelConfigs: (hasModelConfigs: boolean) => void
  onTestConnection: () => void
  onAddModelConfig: (config: Omit<ModelConfig, 'id'>) => void
  onEditModelConfig: (configId: string, updates: Partial<ModelConfig>) => void
  onToggleModelConfig: (configId: string, enabled: boolean) => void
  onDeleteModelConfig: (configId: string) => void
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleDateString('en-GB', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function Field({
  icon: Icon,
  label,
  value,
  mono = false,
  masked = false,
}: {
  icon: typeof Globe
  label: string
  value: string | number | null
  mono?: boolean
  masked?: boolean
}) {
  const [revealed, setRevealed] = useState(false)
  const display = value === null || value === '' ? '—' : String(value)
  const isEmpty = value === null || value === ''

  return (
    <div className="flex items-start gap-3 py-2.5">
      <Icon
        className="w-4 h-4 text-slate-400 dark:text-slate-500 mt-0.5 shrink-0"
        strokeWidth={1.75}
      />
      <div className="min-w-0 flex-1">
        <p className="text-[11px] font-medium uppercase tracking-wider text-slate-400 dark:text-slate-500 mb-0.5">
          {label}
        </p>
        {masked && !isEmpty ? (
          <button
            onClick={() => setRevealed(!revealed)}
            className="group flex items-center gap-1.5"
          >
            <span
              className={`text-sm break-all ${
                mono ? 'font-mono' : ''
              } text-slate-800 dark:text-slate-200`}
            >
              {revealed ? display : '••••••••••••'}
            </span>
            <span className="text-[10px] text-indigo-500 dark:text-indigo-400 opacity-0 group-hover:opacity-100 transition-opacity">
              {revealed ? 'hide' : 'show'}
            </span>
          </button>
        ) : (
          <p
            className={`text-sm break-all ${
              mono ? 'font-mono' : ''
            } ${
              isEmpty
                ? 'text-slate-400 dark:text-slate-500 italic'
                : 'text-slate-800 dark:text-slate-200'
            }`}
          >
            {display}
          </p>
        )}
      </div>
    </div>
  )
}

export function ProviderDetail({
  provider,
  onEdit,
  onDelete,
  onDuplicate,
  onToggle,
  onTestConnection,
  onToggleModelConfigs,
  onAddModelConfig,
  onEditModelConfig,
  onToggleModelConfig,
  onDeleteModelConfig,
}: ProviderDetailProps) {
  const [testing, setTesting] = useState(false)

  const handleTest = () => {
    setTesting(true)
    onTestConnection()
    setTimeout(() => setTesting(false), 2000)
  }

  return (
    <div className="p-5 lg:p-6 space-y-5 max-w-3xl">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3">
            <h1 className="text-lg font-semibold text-slate-800 dark:text-slate-200 font-mono truncate">
              {provider.name}
            </h1>
            <span
              className={`shrink-0 px-2 py-0.5 text-[11px] font-medium rounded-full ${
                provider.type === 'anthropic'
                  ? 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
                  : 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
              }`}
            >
              {provider.type}
            </span>
          </div>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
            {provider.modelCount} model{provider.modelCount !== 1 ? 's' : ''} discovered
            {provider.modelConfigs.length > 0 && (
              <span>
                {' '}· {provider.modelConfigs.length} model config
                {provider.modelConfigs.length !== 1 ? 's' : ''}
              </span>
            )}
          </p>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          {/* Enable/disable toggle */}
          <button
            onClick={() => onToggle(!provider.enabled)}
            className={`relative w-9 h-5 rounded-full transition-colors ${
              provider.enabled
                ? 'bg-indigo-500 dark:bg-indigo-400'
                : 'bg-slate-200 dark:bg-slate-600'
            }`}
            title={provider.enabled ? 'Disable provider' : 'Enable provider'}
          >
            <span
              className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow-sm transition-transform ${
                provider.enabled ? 'translate-x-4' : 'translate-x-0'
              }`}
            />
          </button>

          <div className="h-4 w-px bg-slate-200 dark:bg-slate-700 mx-1" />

          <button
            onClick={onEdit}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:text-slate-500 dark:hover:text-slate-300 dark:hover:bg-slate-700 transition-colors"
            title="Edit provider"
          >
            <Pencil className="w-4 h-4" strokeWidth={1.75} />
          </button>
          <button
            onClick={onDuplicate}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:text-slate-500 dark:hover:text-slate-300 dark:hover:bg-slate-700 transition-colors"
            title="Duplicate provider"
          >
            <Copy className="w-4 h-4" strokeWidth={1.75} />
          </button>
          <button
            onClick={onDelete}
            className="p-1.5 rounded-lg text-slate-400 hover:text-red-500 hover:bg-red-50 dark:text-slate-500 dark:hover:text-red-400 dark:hover:bg-red-950/30 transition-colors"
            title="Delete provider"
          >
            <Trash2 className="w-4 h-4" strokeWidth={1.75} />
          </button>
        </div>
      </div>

      {/* Connection test */}
      <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {provider.lastTestResult === 'success' ? (
              <div className="w-8 h-8 rounded-full bg-emerald-50 dark:bg-emerald-900/30 flex items-center justify-center">
                <CheckCircle2
                  className="w-4 h-4 text-emerald-600 dark:text-emerald-400"
                  strokeWidth={2}
                />
              </div>
            ) : provider.lastTestResult === 'error' ? (
              <div className="w-8 h-8 rounded-full bg-red-50 dark:bg-red-900/30 flex items-center justify-center">
                <XCircle
                  className="w-4 h-4 text-red-500 dark:text-red-400"
                  strokeWidth={2}
                />
              </div>
            ) : (
              <div className="w-8 h-8 rounded-full bg-slate-100 dark:bg-slate-700 flex items-center justify-center">
                <Zap
                  className="w-4 h-4 text-slate-400 dark:text-slate-500"
                  strokeWidth={2}
                />
              </div>
            )}
            <div>
              <p className="text-sm font-medium text-slate-800 dark:text-slate-200">
                {provider.lastTestResult === 'success'
                  ? 'Connected'
                  : provider.lastTestResult === 'error'
                    ? 'Connection failed'
                    : 'Not tested'}
              </p>
              {provider.lastTestedAt ? (
                <p className="text-xs text-slate-400 dark:text-slate-500">
                  Last tested {formatDate(provider.lastTestedAt)}
                  {provider.lastTestLatencyMs !== null && (
                    <span className="ml-1.5 font-mono tabular-nums text-emerald-600 dark:text-emerald-400">
                      {provider.lastTestLatencyMs}ms
                    </span>
                  )}
                </p>
              ) : (
                <p className="text-xs text-slate-400 dark:text-slate-500">
                  No tests run yet
                </p>
              )}
            </div>
          </div>

          <button
            onClick={handleTest}
            disabled={testing}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 disabled:opacity-50 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
          >
            {testing ? (
              <>
                <Loader2 className="w-3.5 h-3.5 animate-spin" strokeWidth={2} />
                Testing…
              </>
            ) : (
              <>
                <Zap className="w-3.5 h-3.5" strokeWidth={2} />
                Test Connection
              </>
            )}
          </button>
        </div>
      </div>

      {/* Configuration fields */}
      <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
        <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-700/50">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">
            Configuration
          </h3>
        </div>
        <div className="px-4 divide-y divide-slate-100 dark:divide-slate-700/50">
          <Field icon={Globe} label="Base URL" value={provider.baseUrl} mono />
          <Field icon={Server} label="Type" value={provider.type} />
          <Field
            icon={Key}
            label="API Key Env"
            value={provider.apiKeyEnv}
            mono
            masked={!!provider.apiKeyEnv}
          />
          <Field icon={Timer} label="Timeout" value={provider.timeout} />
          {provider.rateLimitSeconds !== null && (
            <Field
              icon={Gauge}
              label="Rate Limit"
              value={`${provider.rateLimitSeconds}s between requests`}
            />
          )}
          {provider.models.length > 0 && (
            <div className="flex items-start gap-3 py-2.5">
              <Hash
                className="w-4 h-4 text-slate-400 dark:text-slate-500 mt-0.5 shrink-0"
                strokeWidth={1.75}
              />
              <div className="min-w-0 flex-1">
                <p className="text-[11px] font-medium uppercase tracking-wider text-slate-400 dark:text-slate-500 mb-1.5">
                  Pinned Models
                </p>
                <div className="flex flex-wrap gap-1.5">
                  {provider.models.map((model) => (
                    <span
                      key={model}
                      className="px-2 py-0.5 text-xs font-mono rounded-md bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300"
                    >
                      {model}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Model configurations toggle + table */}
      <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
        <div className="px-4 py-3 flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-slate-800 dark:text-slate-200">
              Model Configurations
            </p>
            <p className="text-xs text-slate-400 dark:text-slate-500 mt-0.5">
              Enable for providers where each model has its own endpoint, API key, or type override
            </p>
          </div>
          <button
            onClick={() => onToggleModelConfigs(!provider.hasModelConfigs)}
            className={`relative w-9 h-5 rounded-full transition-colors shrink-0 ${
              provider.hasModelConfigs
                ? 'bg-indigo-500 dark:bg-indigo-400'
                : 'bg-slate-200 dark:bg-slate-600'
            }`}
          >
            <span
              className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow-sm transition-transform ${
                provider.hasModelConfigs ? 'translate-x-4' : 'translate-x-0'
              }`}
            />
          </button>
        </div>
      </div>

      {provider.hasModelConfigs && (
        <ModelConfigTable
          configs={provider.modelConfigs}
          onAdd={onAddModelConfig}
          onEdit={onEditModelConfig}
          onToggle={onToggleModelConfig}
          onDelete={onDeleteModelConfig}
        />
      )}
    </div>
  )
}
