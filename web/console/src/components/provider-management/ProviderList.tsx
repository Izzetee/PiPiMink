import type { Provider } from '@/types/provider-management'
import { CheckCircle2, XCircle, AlertCircle } from 'lucide-react'

interface ProviderListProps {
  providers: Provider[]
  selectedId: string | null
  onSelect: (id: string) => void
}

function typeBadge(type: string): { label: string; bg: string; text: string } {
  switch (type) {
    case 'anthropic':
      return {
        label: 'Anthropic',
        bg: 'bg-amber-50 dark:bg-amber-900/30',
        text: 'text-amber-700 dark:text-amber-300',
      }
    default:
      return {
        label: 'OpenAI',
        bg: 'bg-indigo-50 dark:bg-indigo-900/30',
        text: 'text-indigo-700 dark:text-indigo-300',
      }
  }
}

function statusIndicator(provider: Provider) {
  if (!provider.enabled) {
    return (
      <span className="flex items-center gap-1 text-[10px] text-slate-400 dark:text-slate-500">
        <span className="w-1.5 h-1.5 rounded-full bg-slate-300 dark:bg-slate-600" />
        Disabled
      </span>
    )
  }
  if (provider.lastTestResult === 'success') {
    return (
      <span className="flex items-center gap-1 text-[10px] text-emerald-600 dark:text-emerald-400">
        <CheckCircle2 className="w-3 h-3" strokeWidth={2} />
        {provider.lastTestLatencyMs}ms
      </span>
    )
  }
  if (provider.lastTestResult === 'error') {
    return (
      <span className="flex items-center gap-1 text-[10px] text-red-500 dark:text-red-400">
        <XCircle className="w-3 h-3" strokeWidth={2} />
        Error
      </span>
    )
  }
  return (
    <span className="flex items-center gap-1 text-[10px] text-slate-400 dark:text-slate-500">
      <AlertCircle className="w-3 h-3" strokeWidth={2} />
      Not tested
    </span>
  )
}

export function ProviderList({ providers, selectedId, onSelect }: ProviderListProps) {
  return (
    <div className="py-1">
      {providers.map((provider) => {
        const badge = typeBadge(provider.type)
        const isSelected = provider.id === selectedId

        return (
          <button
            key={provider.id}
            onClick={() => onSelect(provider.id)}
            className={`w-full text-left px-4 py-3 transition-colors border-l-2 ${
              isSelected
                ? 'bg-indigo-50/60 dark:bg-indigo-950/30 border-l-indigo-500 dark:border-l-indigo-400'
                : 'border-l-transparent hover:bg-slate-50 dark:hover:bg-slate-700/30'
            }`}
          >
            <div className="flex items-center justify-between gap-2">
              <span
                className={`text-sm font-medium font-mono truncate ${
                  isSelected
                    ? 'text-indigo-700 dark:text-indigo-300'
                    : provider.enabled
                      ? 'text-slate-800 dark:text-slate-200'
                      : 'text-slate-400 dark:text-slate-500'
                }`}
              >
                {provider.name}
              </span>
              <span
                className={`shrink-0 px-1.5 py-0.5 text-[10px] font-medium rounded ${badge.bg} ${badge.text}`}
              >
                {badge.label}
              </span>
            </div>

            <div className="flex items-center justify-between mt-1.5">
              {statusIndicator(provider)}
              <span className="text-[10px] text-slate-400 dark:text-slate-500 tabular-nums">
                {provider.modelCount} model{provider.modelCount !== 1 ? 's' : ''}
                {provider.modelConfigs.length > 0 && (
                  <span className="ml-1 text-amber-500 dark:text-amber-400">
                    +{provider.modelConfigs.length} cfg
                  </span>
                )}
              </span>
            </div>
          </button>
        )
      })}
    </div>
  )
}
