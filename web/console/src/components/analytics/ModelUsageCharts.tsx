import { useMemo } from 'react'
import type { ModelUsageEntry } from '@/types/analytics'

interface ModelUsageChartsProps {
  usage: ModelUsageEntry[]
}

const MODEL_COLORS = [
  { bar: 'bg-indigo-500', ring: '#6366f1', label: 'text-indigo-600 dark:text-indigo-400' },
  { bar: 'bg-amber-500', ring: '#f59e0b', label: 'text-amber-600 dark:text-amber-400' },
  { bar: 'bg-emerald-500', ring: '#10b981', label: 'text-emerald-600 dark:text-emerald-400' },
  { bar: 'bg-rose-500', ring: '#f43f5e', label: 'text-rose-600 dark:text-rose-400' },
  { bar: 'bg-cyan-500', ring: '#06b6d4', label: 'text-cyan-600 dark:text-cyan-400' },
  { bar: 'bg-violet-500', ring: '#8b5cf6', label: 'text-violet-600 dark:text-violet-400' },
]

export function ModelUsageCharts({ usage }: ModelUsageChartsProps) {
  const maxCount = useMemo(
    () => Math.max(...usage.map((u) => u.requestCount)),
    [usage]
  )

  const donutSegments = useMemo(() => {
    const circumference = 2 * Math.PI * 70
    let offset = 0
    return usage.map((entry, i) => {
      const c = MODEL_COLORS[i % MODEL_COLORS.length]!
      const length = (entry.percentage / 100) * circumference
      const segment = {
        offset,
        length,
        color: c.ring,
      }
      offset += length
      return segment
    })
  }, [usage])

  const circumference = 2 * Math.PI * 70

  return (
    <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
      <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-700/50">
        <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
          Model Usage
        </h3>
        <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
          Request distribution across models
        </p>
      </div>

      <div className="p-4 grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Bar chart */}
        <div className="space-y-3">
          {usage.map((entry, i) => {
            const color = MODEL_COLORS[i % MODEL_COLORS.length]!
            const pct = (entry.requestCount / maxCount) * 100
            return (
              <div key={entry.modelName}>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs font-mono font-medium text-slate-700 dark:text-slate-300 truncate max-w-[180px]">
                    {entry.modelName}
                  </span>
                  <span className="text-xs font-mono text-slate-500 dark:text-slate-400 flex-shrink-0 ml-2">
                    {entry.requestCount.toLocaleString()}
                  </span>
                </div>
                <div className="h-2.5 rounded-full bg-slate-100 dark:bg-slate-700 overflow-hidden">
                  <div
                    className={`h-full rounded-full ${color.bar} transition-all duration-500`}
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
            )
          })}
        </div>

        {/* Donut chart */}
        <div className="flex flex-col items-center justify-center">
          <div className="relative">
            <svg width="180" height="180" viewBox="0 0 180 180">
              {/* Background ring */}
              <circle
                cx="90"
                cy="90"
                r="70"
                fill="none"
                className="stroke-slate-100 dark:stroke-slate-700"
                strokeWidth="20"
              />
              {/* Segments */}
              {donutSegments.map((seg, i) => (
                <circle
                  key={i}
                  cx="90"
                  cy="90"
                  r="70"
                  fill="none"
                  stroke={seg.color}
                  strokeWidth="20"
                  strokeDasharray={`${seg.length} ${circumference - seg.length}`}
                  strokeDashoffset={-seg.offset}
                  strokeLinecap="butt"
                  transform="rotate(-90 90 90)"
                  className="transition-all duration-500"
                />
              ))}
            </svg>
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              <span className="text-2xl font-semibold text-slate-800 dark:text-slate-100">
                {usage.reduce((sum, u) => sum + u.requestCount, 0).toLocaleString()}
              </span>
              <span className="text-[10px] font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wide">
                Total
              </span>
            </div>
          </div>
          {/* Legend */}
          <div className="flex flex-wrap justify-center gap-x-4 gap-y-1.5 mt-4">
            {usage.map((entry, i) => {
              const color = MODEL_COLORS[i % MODEL_COLORS.length]!
              return (
                <div key={entry.modelName} className="flex items-center gap-1.5">
                  <div
                    className={`w-2.5 h-2.5 rounded-full ${color.bar}`}
                  />
                  <span className="text-[10px] font-mono text-slate-600 dark:text-slate-400">
                    {entry.modelName.split('/').pop()} ({entry.percentage}%)
                  </span>
                </div>
              )
            })}
          </div>
        </div>
      </div>
    </div>
  )
}
