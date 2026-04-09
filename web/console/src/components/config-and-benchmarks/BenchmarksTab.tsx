import { useState, useMemo } from 'react'
import type {
  BenchmarkTask,
  BenchmarkResult,
  ScoreMatrixEntry,
} from '@/types/config-and-benchmarks'
import { ScoreMatrix } from './ScoreMatrix'
import { TaskCard } from './TaskCard'
import { TaskForm } from './TaskForm'
import { RunBenchmarksDialog } from './RunBenchmarksDialog'
import {
  Plus,
  Play,
  Filter,
} from 'lucide-react'

interface BenchmarksTabProps {
  tasks: BenchmarkTask[]
  results: BenchmarkResult[]
  scoreMatrix: ScoreMatrixEntry[]
  benchmarkJudge?: string
  onCreateTask?: (task: Omit<BenchmarkTask, 'id' | 'resultCount'>) => void
  onEditTask?: (id: string, updates: Partial<BenchmarkTask>) => void
  onDeleteTask?: (id: string) => void
  onToggleTask?: (id: string, enabled: boolean) => void
  onRunBenchmarks?: (modelNames: string[]) => void
}

export function BenchmarksTab({
  tasks,
  results,
  scoreMatrix,
  benchmarkJudge,
  onCreateTask,
  onEditTask,
  onDeleteTask,
  onToggleTask,
  onRunBenchmarks,
}: BenchmarksTabProps) {
  const [categoryFilter, setCategoryFilter] = useState<string>('all')
  const [showBuiltin, setShowBuiltin] = useState(true)
  const [showCustom, setShowCustom] = useState(true)
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [runDialogOpen, setRunDialogOpen] = useState(false)

  const allCategories = useMemo(
    () => [...new Set(tasks.map((t) => t.category))].sort(),
    [tasks]
  )

  const filteredTasks = useMemo(() => {
    return tasks.filter((t) => {
      if (categoryFilter !== 'all' && t.category !== categoryFilter) return false
      if (!showBuiltin && t.builtin) return false
      if (!showCustom && !t.builtin) return false
      return true
    })
  }, [tasks, categoryFilter, showBuiltin, showCustom])

  const modelNames = useMemo(
    () => [...new Set(scoreMatrix.map((e) => e.modelName))],
    [scoreMatrix]
  )

  const taskResults = useMemo(() => {
    const map = new Map<string, BenchmarkResult[]>()
    for (const r of results) {
      const arr = map.get(r.taskId) ?? []
      arr.push(r)
      map.set(r.taskId, arr)
    }
    return map
  }, [results])

  const editingTask = editingId ? tasks.find((t) => t.id === editingId) ?? null : null

  function handleSave(data: Partial<BenchmarkTask>) {
    if (editingTask) {
      onEditTask?.(editingTask.id, data)
    } else {
      onCreateTask?.(data as Omit<BenchmarkTask, 'id' | 'resultCount'>)
    }
    setCreating(false)
    setEditingId(null)
  }

  function handleCancel() {
    setCreating(false)
    setEditingId(null)
  }

  return (
    <div className="space-y-5">
      {/* Score matrix */}
      {benchmarkJudge && scoreMatrix.length > 0 && (
        <p className="text-[11px] text-slate-400 dark:text-slate-500">
          Benchmark Judge: <span className="font-mono">{benchmarkJudge}</span>
        </p>
      )}
      <ScoreMatrix entries={scoreMatrix} />

      {/* Toolbar */}
      <div className="flex flex-col sm:flex-row sm:items-center gap-3">
        {/* Category filter */}
        <div className="flex items-center gap-1.5 flex-wrap">
          <Filter className="w-3.5 h-3.5 text-slate-400 dark:text-slate-500" strokeWidth={2} />
          <button
            onClick={() => setCategoryFilter('all')}
            className={`px-2.5 py-1 text-xs font-medium rounded-lg transition-colors ${
              categoryFilter === 'all'
                ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300'
                : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'
            }`}
          >
            All
          </button>
          {allCategories.map((cat) => (
            <button
              key={cat}
              onClick={() => setCategoryFilter(cat)}
              className={`px-2.5 py-1 text-xs font-medium rounded-lg transition-colors ${
                categoryFilter === cat
                  ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300'
                  : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'
              }`}
            >
              {cat}
            </button>
          ))}
        </div>

        {/* Type toggles */}
        <div className="flex items-center gap-2 text-xs">
          <label className="inline-flex items-center gap-1.5 cursor-pointer select-none text-slate-500 dark:text-slate-400">
            <input
              type="checkbox"
              checked={showBuiltin}
              onChange={(e) => setShowBuiltin(e.target.checked)}
              className="w-3 h-3 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500/30 dark:border-slate-600 dark:bg-slate-700"
            />
            Builtin
          </label>
          <label className="inline-flex items-center gap-1.5 cursor-pointer select-none text-slate-500 dark:text-slate-400">
            <input
              type="checkbox"
              checked={showCustom}
              onChange={(e) => setShowCustom(e.target.checked)}
              className="w-3 h-3 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500/30 dark:border-slate-600 dark:bg-slate-700"
            />
            Custom
          </label>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2 sm:ml-auto">
          <button
            onClick={() => setRunDialogOpen(true)}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 transition-colors shadow-sm"
          >
            <Play className="w-3.5 h-3.5" strokeWidth={2} />
            Run Benchmarks
          </button>
          <button
            onClick={() => {
              setCreating(true)
              setEditingId(null)
            }}
            className="inline-flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" strokeWidth={2} />
            New Task
          </button>
        </div>
      </div>

      {/* Create form — shown at top only for new tasks */}
      {creating && (
        <TaskForm
          existingCategories={allCategories}
          onSave={handleSave}
          onCancel={handleCancel}
        />
      )}

      {/* Task list — edit form renders inline below the selected task */}
      <div className="space-y-2">
        {filteredTasks.map((task) => (
          <div key={task.id}>
            <TaskCard
              task={task}
              results={taskResults.get(task.id) ?? []}
              isExpanded={expandedId === task.id}
              onExpand={() =>
                setExpandedId(expandedId === task.id ? null : task.id)
              }
              onEdit={() => {
                setEditingId(task.id)
                setCreating(false)
              }}
              onDelete={() => onDeleteTask?.(task.id)}
              onToggle={(enabled) => onToggleTask?.(task.id, enabled)}
            />
            {/* Inline edit form below the selected task */}
            {editingId === task.id && editingTask && (
              <div className="mt-2">
                <TaskForm
                  task={editingTask}
                  existingCategories={allCategories}
                  onSave={handleSave}
                  onCancel={handleCancel}
                />
              </div>
            )}
          </div>
        ))}

        {filteredTasks.length === 0 && (
          <div className="py-12 text-center rounded-xl border border-dashed border-slate-200 dark:border-slate-700">
            <p className="text-sm text-slate-500 dark:text-slate-400">
              No benchmark tasks match the current filters.
            </p>
          </div>
        )}
      </div>

      {/* Run benchmarks dialog */}
      {runDialogOpen && (
        <RunBenchmarksDialog
          modelNames={modelNames}
          onRun={(selected) => {
            onRunBenchmarks?.(selected)
            setRunDialogOpen(false)
          }}
          onClose={() => setRunDialogOpen(false)}
        />
      )}
    </div>
  )
}
