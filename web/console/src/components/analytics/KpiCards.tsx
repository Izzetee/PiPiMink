import type { KpiSummary } from '@/types/analytics'
import { Activity, Clock, Cpu, AlertTriangle } from 'lucide-react'

interface KpiCardsProps {
  summary: KpiSummary
}

export function KpiCards({ summary }: KpiCardsProps) {
  const cards = [
    {
      label: 'Total Requests',
      value: summary.totalRequests.toLocaleString(),
      icon: Activity,
      accent: 'text-indigo-600 dark:text-indigo-400',
      bg: 'bg-indigo-50 dark:bg-indigo-950/40',
    },
    {
      label: 'Avg Latency',
      value: `${summary.avgLatencyMs}ms`,
      icon: Clock,
      accent: 'text-amber-600 dark:text-amber-400',
      bg: 'bg-amber-50 dark:bg-amber-950/40',
    },
    {
      label: 'Most Used Model',
      value: summary.mostUsedModel,
      icon: Cpu,
      accent: 'text-emerald-600 dark:text-emerald-400',
      bg: 'bg-emerald-50 dark:bg-emerald-950/40',
      mono: true,
    },
    {
      label: 'Error Rate',
      value: `${(summary.errorRate * 100).toFixed(1)}%`,
      icon: AlertTriangle,
      accent: summary.errorRate > 0.05
        ? 'text-red-600 dark:text-red-400'
        : 'text-slate-600 dark:text-slate-400',
      bg: summary.errorRate > 0.05
        ? 'bg-red-50 dark:bg-red-950/40'
        : 'bg-slate-50 dark:bg-slate-800/40',
    },
  ]

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {cards.map((card) => (
        <div
          key={card.label}
          className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 transition-shadow hover:shadow-md"
        >
          <div className="flex items-center gap-3">
            <div className={`flex-shrink-0 w-10 h-10 rounded-lg ${card.bg} flex items-center justify-center`}>
              <card.icon className={`w-5 h-5 ${card.accent}`} strokeWidth={1.75} />
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wide">
                {card.label}
              </p>
              <p className={`text-lg font-semibold text-slate-900 dark:text-slate-100 truncate ${card.mono ? 'font-mono text-sm' : ''}`}>
                {card.value}
              </p>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
