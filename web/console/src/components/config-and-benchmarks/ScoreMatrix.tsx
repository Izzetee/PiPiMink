import { useMemo } from 'react'
import type { ScoreMatrixEntry } from '@/types/config-and-benchmarks'

interface ScoreMatrixProps {
  entries: ScoreMatrixEntry[]
}

function scoreColor(score: number): string {
  if (score >= 0.9) return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300'
  if (score >= 0.7) return 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
  if (score >= 0.5) return 'bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-300'
  return 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300'
}

function avgScore(entry: ScoreMatrixEntry): number {
  const vals = Object.values(entry.scores)
  if (vals.length === 0) return 0
  return vals.reduce((a, b) => a + b, 0) / vals.length
}

export function ScoreMatrix({ entries }: ScoreMatrixProps) {
  const categories = useMemo(() => {
    const catSet = new Set<string>()
    for (const entry of entries) {
      for (const key of Object.keys(entry.scores)) {
        catSet.add(key)
      }
    }
    return [...catSet]
  }, [entries])

  const sorted = [...entries].sort((a, b) => avgScore(b) - avgScore(a))

  return (
    <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 overflow-x-auto">
      <table className="w-full text-xs">
        <thead>
          <tr className="border-b border-slate-100 dark:border-slate-700/50">
            <th className="px-3 py-2.5 text-left font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 min-w-[180px]">
              Model
            </th>
            {categories.map((cat) => (
              <th
                key={cat}
                className="px-2 py-2.5 text-center font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 w-20"
              >
                {cat.length > 8 ? cat.slice(0, 7) + '\u2026' : cat}
              </th>
            ))}
            <th className="px-2 py-2.5 text-center font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 w-16">
              Avg
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-50 dark:divide-slate-700/30">
          {sorted.map((entry) => {
            const avg = avgScore(entry)
            return (
              <tr
                key={entry.modelName}
                className="hover:bg-slate-50/80 dark:hover:bg-slate-700/20 transition-colors"
              >
                <td className="px-3 py-2.5 font-mono font-medium text-slate-800 dark:text-slate-200">
                  {entry.modelName}
                </td>
                {categories.map((cat) => {
                  const score = entry.scores[cat]
                  return (
                    <td key={cat} className="px-2 py-2.5 text-center">
                      {score != null ? (
                        <span
                          className={`inline-block px-2 py-0.5 rounded-md font-mono font-medium tabular-nums ${scoreColor(score)}`}
                        >
                          {Math.round(score * 100)}%
                        </span>
                      ) : (
                        <span className="text-slate-300 dark:text-slate-600">—</span>
                      )}
                    </td>
                  )
                })}
                <td className="px-2 py-2.5 text-center">
                  <span className="font-mono font-semibold tabular-nums text-slate-700 dark:text-slate-300">
                    {Math.round(avg * 100)}%
                  </span>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
