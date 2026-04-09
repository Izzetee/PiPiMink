import type { DashboardStats, StatFilter } from '@/types/model-dashboard'
import { Box, ToggleRight, EyeOff, Radar, FlaskConical } from 'lucide-react'

interface StatCardDef {
  key: StatFilter
  label: string
  icon: typeof Box
  valueKey: keyof DashboardStats
}

const cards: StatCardDef[] = [
  { key: 'all', label: 'Total', icon: Box, valueKey: 'total' },
  { key: 'taggedAndEnabled', label: 'Tagged & Enabled', icon: ToggleRight, valueKey: 'taggedAndEnabled' },
  { key: 'disabled', label: 'Disabled', icon: EyeOff, valueKey: 'disabled' },
  { key: 'discovered', label: 'Discovered', icon: Radar, valueKey: 'discovered' },
  { key: 'benchmarked', label: 'Benchmarked', icon: FlaskConical, valueKey: 'benchmarked' },
  // Note: 'enabled' filter is used by the tab row but not shown as a stat card
]

interface StatCardsProps {
  stats: DashboardStats
  activeFilter: StatFilter
  onFilterChange?: (filter: StatFilter) => void
}

export function StatCards({ stats, activeFilter, onFilterChange }: StatCardsProps) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
      {cards.map(({ key, label, icon: Icon, valueKey }) => {
        const isActive = activeFilter === key
        return (
          <button
            key={key}
            onClick={() => onFilterChange?.(key)}
            className={`
              group relative rounded-xl border px-4 py-3.5 text-left transition-all duration-200 outline-none
              ${isActive
                ? 'border-indigo-200 bg-indigo-50/80 dark:border-indigo-500/30 dark:bg-indigo-950/40 ring-1 ring-indigo-200 dark:ring-indigo-500/20'
                : 'border-slate-200 bg-white hover:border-slate-300 hover:shadow-sm dark:border-slate-700 dark:bg-slate-800 dark:hover:border-slate-600'
              }
            `}
          >
            <div className="flex items-center justify-between mb-2">
              <Icon
                className={`w-4 h-4 ${
                  isActive
                    ? 'text-indigo-500 dark:text-indigo-400'
                    : 'text-slate-400 dark:text-slate-500'
                }`}
                strokeWidth={1.75}
              />
              {isActive && (
                <span className="w-1.5 h-1.5 rounded-full bg-indigo-500 dark:bg-indigo-400" />
              )}
            </div>
            <p
              className={`text-2xl font-semibold tabular-nums tracking-tight ${
                isActive
                  ? 'text-indigo-700 dark:text-indigo-300'
                  : 'text-slate-900 dark:text-slate-100'
              }`}
            >
              {stats[valueKey]}
            </p>
            <p
              className={`text-xs font-medium mt-0.5 ${
                isActive
                  ? 'text-indigo-500/80 dark:text-indigo-400/70'
                  : 'text-slate-500 dark:text-slate-400'
              }`}
            >
              {label}
            </p>
          </button>
        )
      })}
    </div>
  )
}
