import { Tag, FlaskConical, X } from 'lucide-react'

interface FloatingActionBarProps {
  selectedCount: number
  onTagSelected?: () => void
  onBenchmarkSelected?: () => void
  onDeselectAll?: () => void
}

export function FloatingActionBar({
  selectedCount,
  onTagSelected,
  onBenchmarkSelected,
  onDeselectAll,
}: FloatingActionBarProps) {
  if (selectedCount === 0) return null

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-40 animate-fade-in">
      <div className="flex items-center gap-2 px-4 py-2.5 rounded-xl border border-slate-200 bg-white/95 backdrop-blur-sm shadow-lg dark:border-slate-600 dark:bg-slate-800/95">
        <span className="text-sm font-medium text-slate-700 dark:text-slate-300 mr-1">
          <span className="inline-flex items-center justify-center w-5 h-5 rounded-full bg-indigo-100 text-indigo-700 dark:bg-indigo-900/50 dark:text-indigo-300 text-xs font-semibold tabular-nums mr-1.5">
            {selectedCount}
          </span>
          selected
        </span>

        <div className="w-px h-5 bg-slate-200 dark:bg-slate-600 mx-1" />

        <button
          onClick={onTagSelected}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg bg-indigo-50 text-indigo-700 hover:bg-indigo-100 dark:bg-indigo-900/40 dark:text-indigo-300 dark:hover:bg-indigo-900/60 transition-colors"
        >
          <Tag className="w-3.5 h-3.5" strokeWidth={1.75} />
          Tag
        </button>

        <button
          onClick={onBenchmarkSelected}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg bg-amber-50 text-amber-700 hover:bg-amber-100 dark:bg-amber-900/40 dark:text-amber-300 dark:hover:bg-amber-900/60 transition-colors"
        >
          <FlaskConical className="w-3.5 h-3.5" strokeWidth={1.75} />
          Benchmark
        </button>

        <button
          onClick={onDeselectAll}
          className="p-1.5 rounded-md text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:text-slate-500 dark:hover:text-slate-300 dark:hover:bg-slate-700 transition-colors ml-1"
          title="Deselect all"
        >
          <X className="w-4 h-4" strokeWidth={1.75} />
        </button>
      </div>
    </div>
  )
}
