import { useState } from 'react'
import { X, Play, Check } from 'lucide-react'

interface RunBenchmarksDialogProps {
  modelNames: string[]
  onRun: (selectedModels: string[]) => void
  onClose: () => void
}

export function RunBenchmarksDialog({
  modelNames,
  onRun,
  onClose,
}: RunBenchmarksDialogProps) {
  const [selected, setSelected] = useState<Set<string>>(new Set(modelNames))

  function toggle(name: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(name)) next.delete(name)
      else next.add(name)
      return next
    })
  }

  function toggleAll() {
    if (selected.size === modelNames.length) {
      setSelected(new Set())
    } else {
      setSelected(new Set(modelNames))
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 dark:bg-black/60"
        onClick={onClose}
      />

      {/* Dialog */}
      <div className="relative w-full max-w-md mx-4 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-100 dark:border-slate-700/50">
          <div>
            <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
              Run Benchmarks
            </h3>
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
              Select models to evaluate against enabled tasks
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:text-slate-500 dark:hover:text-slate-300 dark:hover:bg-slate-700 transition-colors"
          >
            <X className="w-4 h-4" strokeWidth={2} />
          </button>
        </div>

        {/* Model list */}
        <div className="px-5 py-3">
          <button
            onClick={toggleAll}
            className="text-xs font-medium text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-300 transition-colors mb-2"
          >
            {selected.size === modelNames.length ? 'Deselect All' : 'Select All'}
          </button>

          <div className="space-y-1 max-h-60 overflow-y-auto">
            {modelNames.map((name) => {
              const isSelected = selected.has(name)
              return (
                <button
                  key={name}
                  onClick={() => toggle(name)}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-colors ${
                    isSelected
                      ? 'bg-indigo-50 dark:bg-indigo-900/20'
                      : 'hover:bg-slate-50 dark:hover:bg-slate-700/30'
                  }`}
                >
                  <span
                    className={`flex-shrink-0 w-4 h-4 rounded border-2 flex items-center justify-center transition-colors ${
                      isSelected
                        ? 'bg-indigo-500 border-indigo-500 dark:bg-indigo-400 dark:border-indigo-400'
                        : 'border-slate-300 dark:border-slate-600'
                    }`}
                  >
                    {isSelected && <Check className="w-3 h-3 text-white" strokeWidth={3} />}
                  </span>
                  <span className="text-sm font-mono text-slate-700 dark:text-slate-300 truncate">
                    {name}
                  </span>
                </button>
              )
            })}
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-5 py-4 border-t border-slate-100 dark:border-slate-700/50">
          <span className="text-xs text-slate-500 dark:text-slate-400">
            {selected.size} of {modelNames.length} selected
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={onClose}
              className="px-3 py-2 text-sm font-medium rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={() => onRun([...selected])}
              disabled={selected.size === 0}
              className="inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 transition-colors shadow-sm disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Play className="w-3.5 h-3.5" strokeWidth={2} />
              Run ({selected.size})
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
