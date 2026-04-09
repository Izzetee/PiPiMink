import { useState, useMemo } from 'react'
import type {
  LatencyPerModelEntry,
  LatencyTimeSeriesPoint,
  LatencyPercentilesEntry,
} from '@/types/analytics'
import { BarChart3, TrendingUp, Table2 } from 'lucide-react'

interface LatencySectionProps {
  perModel: LatencyPerModelEntry[]
  timeSeries: LatencyTimeSeriesPoint[]
  percentiles: LatencyPercentilesEntry[]
}

type LatencyView = 'per-model' | 'time-series' | 'percentiles'

function latencyBarColor(ms: number): string {
  if (ms < 250) return 'bg-emerald-500'
  if (ms < 500) return 'bg-amber-500'
  return 'bg-red-500'
}

function formatHour(ts: string): string {
  const d = new Date(ts)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

export function LatencySection({
  perModel,
  timeSeries,
  percentiles,
}: LatencySectionProps) {
  const [view, setView] = useState<LatencyView>('per-model')

  const maxLatency = useMemo(
    () => Math.max(...perModel.map((m) => m.avgLatencyMs)),
    [perModel]
  )

  const timeSeriesMax = useMemo(
    () => Math.max(...timeSeries.map((p) => Math.max(p.avgLatencyMs, p.p95LatencyMs))),
    [timeSeries]
  )

  const views: { key: LatencyView; label: string; icon: typeof BarChart3 }[] = [
    { key: 'per-model', label: 'Per Model', icon: BarChart3 },
    { key: 'time-series', label: 'Over Time', icon: TrendingUp },
    { key: 'percentiles', label: 'Percentiles', icon: Table2 },
  ]

  // Build SVG polyline points for time series
  const chartWidth = 100
  const chartHeight = 100
  const padding = { top: 5, bottom: 5, left: 0, right: 0 }
  const innerW = chartWidth - padding.left - padding.right
  const innerH = chartHeight - padding.top - padding.bottom

  function toPoints(values: number[]): string {
    return values
      .map((v, i) => {
        const x = padding.left + (i / (values.length - 1)) * innerW
        const y = padding.top + innerH - (v / timeSeriesMax) * innerH
        return `${x},${y}`
      })
      .join(' ')
  }

  const avgPoints = toPoints(timeSeries.map((p) => p.avgLatencyMs))
  const p95Points = toPoints(timeSeries.map((p) => p.p95LatencyMs))

  return (
    <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
      <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-700/50 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
            Latency
          </h3>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            Response time across all users
          </p>
        </div>
        <div className="inline-flex rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/60 p-0.5">
          {views.map((v) => (
            <button
              key={v.key}
              onClick={() => setView(v.key)}
              className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                view === v.key
                  ? 'bg-indigo-600 text-white shadow-sm'
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
              }`}
            >
              <v.icon className="w-3 h-3" strokeWidth={2} />
              {v.label}
            </button>
          ))}
        </div>
      </div>

      <div className="p-4">
        {/* Per-model bars */}
        {view === 'per-model' && (
          <div className="space-y-3">
            {perModel.map((m) => {
              const pct = (m.avgLatencyMs / maxLatency) * 100
              return (
                <div key={m.modelName} className="flex items-center gap-3">
                  <span className="text-xs font-mono font-medium text-slate-700 dark:text-slate-300 truncate w-48 flex-shrink-0 text-right">
                    {m.modelName}
                  </span>
                  <div className="flex-1 h-3 rounded-full bg-slate-100 dark:bg-slate-700 overflow-hidden">
                    <div
                      className={`h-full rounded-full ${latencyBarColor(m.avgLatencyMs)} transition-all duration-500`}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <span className="text-xs font-mono font-medium text-slate-600 dark:text-slate-400 w-14 text-right flex-shrink-0">
                    {m.avgLatencyMs}ms
                  </span>
                </div>
              )
            })}
            <div className="flex items-center gap-4 mt-3 pt-3 border-t border-slate-100 dark:border-slate-700/50">
              <div className="flex items-center gap-1.5">
                <div className="w-3 h-1.5 rounded-full bg-emerald-500" />
                <span className="text-[10px] text-slate-500 dark:text-slate-400">&lt; 250ms</span>
              </div>
              <div className="flex items-center gap-1.5">
                <div className="w-3 h-1.5 rounded-full bg-amber-500" />
                <span className="text-[10px] text-slate-500 dark:text-slate-400">250–500ms</span>
              </div>
              <div className="flex items-center gap-1.5">
                <div className="w-3 h-1.5 rounded-full bg-red-500" />
                <span className="text-[10px] text-slate-500 dark:text-slate-400">&gt; 500ms</span>
              </div>
            </div>
          </div>
        )}

        {/* Time series line chart */}
        {view === 'time-series' && (
          <div>
            <div className="relative aspect-[3/1] w-full">
              <svg
                viewBox={`0 0 ${chartWidth} ${chartHeight}`}
                preserveAspectRatio="none"
                className="w-full h-full"
              >
                {/* Grid lines */}
                {[0, 0.25, 0.5, 0.75, 1].map((frac) => {
                  const y = padding.top + innerH - frac * innerH
                  return (
                    <line
                      key={frac}
                      x1={padding.left}
                      y1={y}
                      x2={chartWidth - padding.right}
                      y2={y}
                      className="stroke-slate-200 dark:stroke-slate-700"
                      strokeWidth="0.3"
                    />
                  )
                })}
                {/* P95 area fill */}
                <polygon
                  points={`${padding.left},${padding.top + innerH} ${p95Points} ${chartWidth - padding.right},${padding.top + innerH}`}
                  className="fill-rose-100/50 dark:fill-rose-900/20"
                />
                {/* P95 line */}
                <polyline
                  points={p95Points}
                  fill="none"
                  className="stroke-rose-400 dark:stroke-rose-500"
                  strokeWidth="0.8"
                  strokeLinejoin="round"
                />
                {/* Avg area fill */}
                <polygon
                  points={`${padding.left},${padding.top + innerH} ${avgPoints} ${chartWidth - padding.right},${padding.top + innerH}`}
                  className="fill-indigo-100/50 dark:fill-indigo-900/20"
                />
                {/* Avg line */}
                <polyline
                  points={avgPoints}
                  fill="none"
                  className="stroke-indigo-500 dark:stroke-indigo-400"
                  strokeWidth="1"
                  strokeLinejoin="round"
                />
                {/* Dots for avg */}
                {timeSeries.map((p, i) => {
                  const x = padding.left + (i / (timeSeries.length - 1)) * innerW
                  const y = padding.top + innerH - (p.avgLatencyMs / timeSeriesMax) * innerH
                  return (
                    <circle
                      key={i}
                      cx={x}
                      cy={y}
                      r="1.2"
                      className="fill-indigo-500 dark:fill-indigo-400"
                    />
                  )
                })}
              </svg>
            </div>
            {/* X-axis labels */}
            <div className="flex justify-between mt-1 px-1">
              {timeSeries.filter((_, i) => i % 2 === 0 || i === timeSeries.length - 1).map((p) => (
                <span key={p.timestamp} className="text-[10px] font-mono text-slate-400">
                  {formatHour(p.timestamp)}
                </span>
              ))}
            </div>
            {/* Legend */}
            <div className="flex items-center gap-4 mt-3 pt-3 border-t border-slate-100 dark:border-slate-700/50">
              <div className="flex items-center gap-1.5">
                <div className="w-4 h-0.5 rounded-full bg-indigo-500" />
                <span className="text-[10px] text-slate-500 dark:text-slate-400">Avg Latency</span>
              </div>
              <div className="flex items-center gap-1.5">
                <div className="w-4 h-0.5 rounded-full bg-rose-400" />
                <span className="text-[10px] text-slate-500 dark:text-slate-400">P95 Latency</span>
              </div>
            </div>
          </div>
        )}

        {/* Percentiles table */}
        {view === 'percentiles' && (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left">
                  <th className="pb-3 pr-4 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wide">
                    Model
                  </th>
                  <th className="pb-3 px-4 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wide text-right">
                    P50
                  </th>
                  <th className="pb-3 px-4 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wide text-right">
                    P95
                  </th>
                  <th className="pb-3 pl-4 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wide text-right">
                    P99
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-slate-700/50">
                {percentiles.map((row) => (
                  <tr key={row.modelName} className="hover:bg-slate-50/80 dark:hover:bg-slate-700/20 transition-colors">
                    <td className="py-2.5 pr-4 text-xs font-mono font-medium text-slate-800 dark:text-slate-200 truncate max-w-[200px]">
                      {row.modelName}
                    </td>
                    <td className="py-2.5 px-4 text-xs font-mono text-right text-emerald-600 dark:text-emerald-400">
                      {row.p50Ms}ms
                    </td>
                    <td className="py-2.5 px-4 text-xs font-mono text-right text-amber-600 dark:text-amber-400">
                      {row.p95Ms}ms
                    </td>
                    <td className="py-2.5 pl-4 text-xs font-mono text-right text-red-600 dark:text-red-400">
                      {row.p99Ms}ms
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="flex items-center gap-4 mt-3 pt-3 border-t border-slate-100 dark:border-slate-700/50">
              <div className="flex items-center gap-1.5">
                <div className="w-2.5 h-2.5 rounded-full bg-emerald-500" />
                <span className="text-[10px] text-slate-500 dark:text-slate-400">P50 (median)</span>
              </div>
              <div className="flex items-center gap-1.5">
                <div className="w-2.5 h-2.5 rounded-full bg-amber-500" />
                <span className="text-[10px] text-slate-500 dark:text-slate-400">P95</span>
              </div>
              <div className="flex items-center gap-1.5">
                <div className="w-2.5 h-2.5 rounded-full bg-red-500" />
                <span className="text-[10px] text-slate-500 dark:text-slate-400">P99 (tail)</span>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
