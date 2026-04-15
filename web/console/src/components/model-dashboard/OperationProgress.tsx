import type { Operation } from '@/types/model-dashboard'
import { Loader2 } from 'lucide-react'

interface OperationProgressProps {
  operation: Operation
}

const operationLabels: Record<string, string> = {
  tag: 'Tagging models',
  benchmark: 'Running benchmarks',
  discover: 'Discovering models',
}

export function OperationProgress({ operation }: OperationProgressProps) {
  const modelProgress =
    operation.totalModels > 0
      ? (operation.completedModels / operation.totalModels) * 100
      : 0
  const label = operationLabels[operation.type] || 'Processing'

  const showTaskBar =
    operation.type === 'benchmark' &&
    (operation.totalTasks ?? 0) > 0

  const taskProgress = showTaskBar
    ? ((operation.completedTasks ?? 0) / (operation.totalTasks ?? 1)) * 100
    : 0

  return (
    <div className="rounded-xl border border-indigo-200 bg-indigo-50/60 dark:border-indigo-500/20 dark:bg-indigo-950/30 px-4 py-3">
      {/* Model-level progress */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Loader2
            className="w-4 h-4 text-indigo-500 dark:text-indigo-400 animate-spin"
            strokeWidth={2}
          />
          <span className="text-sm font-medium text-indigo-700 dark:text-indigo-300">
            {label}
          </span>
        </div>
        <div className="flex items-center gap-3 text-xs text-indigo-600/70 dark:text-indigo-400/60">
          <span className="font-mono tabular-nums">
            {operation.completedModels}/{operation.totalModels} models
          </span>
          <span className="font-mono tabular-nums font-medium">
            {Math.round(modelProgress)}%
          </span>
        </div>
      </div>

      <div className="h-1.5 rounded-full bg-indigo-100 dark:bg-indigo-900/50 overflow-hidden">
        <div
          className="h-full rounded-full bg-indigo-500 dark:bg-indigo-400 transition-all duration-500 ease-out"
          style={{ width: `${modelProgress}%` }}
        />
      </div>

      {operation.currentModel && (
        <p className="mt-1.5 text-xs text-indigo-500/70 dark:text-indigo-400/50 font-mono truncate">
          {operation.currentModel}
        </p>
      )}

      {/* Task-level progress (benchmark only) */}
      {showTaskBar && (
        <div className="mt-2.5">
          <div className="flex items-center justify-between mb-1">
            <span className="text-[11px] text-indigo-500/60 dark:text-indigo-400/40">
              Tasks
            </span>
            <div className="flex items-center gap-3 text-[11px] text-indigo-500/60 dark:text-indigo-400/40">
              <span className="font-mono tabular-nums">
                {operation.completedTasks ?? 0}/{operation.totalTasks ?? 0} tasks
              </span>
              <span className="font-mono tabular-nums font-medium">
                {Math.round(taskProgress)}%
              </span>
            </div>
          </div>

          <div className="h-1 rounded-full bg-indigo-100 dark:bg-indigo-900/50 overflow-hidden">
            <div
              className="h-full rounded-full bg-indigo-300 dark:bg-indigo-600 transition-all duration-500 ease-out"
              style={{ width: `${taskProgress}%` }}
            />
          </div>

          {operation.currentTask && (
            <p className="mt-1 text-[11px] text-indigo-400/50 dark:text-indigo-500/40 font-mono truncate">
              {operation.currentTask}
            </p>
          )}
        </div>
      )}
    </div>
  )
}
