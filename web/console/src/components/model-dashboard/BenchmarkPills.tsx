import type { BenchmarkCategory } from '@/types/model-dashboard'

const categoryLabels: Record<BenchmarkCategory, string> = {
  coding: 'coding',
  creative: 'creative',
  factual: 'factual',
  instruction: 'instruction',
  reasoning: 'reasoning',
  summarization: 'summarization',
}

const categoryOrder: BenchmarkCategory[] = [
  'coding',
  'creative',
  'factual',
  'instruction',
  'reasoning',
  'summarization',
]

function scoreColor(score: number): string {
  if (score >= 0.7) return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
  if (score >= 0.4) return 'bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
  return 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-400'
}

interface BenchmarkPillsProps {
  scores: Partial<Record<BenchmarkCategory, number>>
}

export function BenchmarkPills({ scores }: BenchmarkPillsProps) {
  const entries = categoryOrder.filter((cat) => cat in scores)

  if (entries.length === 0) {
    return (
      <span className="text-xs text-slate-400 dark:text-slate-500 italic">
        none
      </span>
    )
  }

  return (
    <div className="flex flex-wrap gap-1">
      {entries.map((cat) => {
        const score = scores[cat]!
        return (
          <span
            key={cat}
            className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] font-medium leading-none ${scoreColor(score)}`}
          >
            {categoryLabels[cat]}{' '}
            <span className="font-mono tabular-nums">
              {Math.round(score * 100)}%
            </span>
          </span>
        )
      })}
    </div>
  )
}
