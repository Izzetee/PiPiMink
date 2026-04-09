import { useState } from 'react'
import type { BenchmarkTask, BenchmarkResult, JudgeCriterion } from '@/types/config-and-benchmarks'
import {
  ChevronDown,
  ChevronRight,
  Pencil,
  Trash2,
  Clock,
  Beaker,
  Shield,
  Sparkles,
} from 'lucide-react'

interface TaskCardProps {
  task: BenchmarkTask
  results: BenchmarkResult[]
  isExpanded: boolean
  onExpand: () => void
  onEdit?: () => void
  onDelete?: () => void
  onToggle?: (enabled: boolean) => void
}

const categoryColors: Record<string, string> = {
  coding: 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300',
  reasoning: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
  'instruction-following': 'bg-teal-100 text-teal-700 dark:bg-teal-900/30 dark:text-teal-300',
  'creative-writing': 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300',
  summarization: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
  'factual-qa': 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300',
  'coding-security': 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300',
}

const scoringIcons: Record<string, typeof Beaker> = {
  'llm-judge': Sparkles,
  deterministic: Shield,
  format: Beaker,
}

function scoreBarColor(score: number): string {
  if (score >= 0.9) return 'bg-emerald-500 dark:bg-emerald-400'
  if (score >= 0.7) return 'bg-amber-500 dark:bg-amber-400'
  if (score >= 0.5) return 'bg-orange-500 dark:bg-orange-400'
  return 'bg-red-500 dark:bg-red-400'
}

function formatLatency(ms: number): string {
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`
  return `${ms}ms`
}

function formatTimestamp(ts: string): string {
  const d = new Date(ts)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }) +
    ' ' +
    d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
}

function ResponseBlock({ response }: { response: string }) {
  const lines = response.split('\n')
  const [showAll, setShowAll] = useState(false)
  const truncated = !showAll && lines.length > 10
  const displayText = truncated ? lines.slice(0, 10).join('\n') : response

  return (
    <div className="mt-1.5 ml-[calc(12rem+0.75rem)]">
      <p className="text-[10px] text-slate-400 dark:text-slate-500 font-medium mb-1">
        Model Response
      </p>
      <div className="max-h-[400px] overflow-auto rounded-lg bg-slate-50 dark:bg-slate-900/40 p-3">
        <pre className="text-xs font-mono text-slate-600 dark:text-slate-300 whitespace-pre-wrap break-words leading-relaxed">
          {displayText}
        </pre>
        {lines.length > 10 && (
          <button
            onClick={() => setShowAll(!showAll)}
            className="mt-1.5 text-[10px] text-indigo-500 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-300 font-medium transition-colors"
          >
            {showAll ? 'Show less' : `Show all ${lines.length} lines`}
          </button>
        )}
      </div>
    </div>
  )
}

export function TaskCard({
  task,
  results,
  isExpanded,
  onExpand,
  onEdit,
  onDelete,
  onToggle,
}: TaskCardProps) {
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [showResults, setShowResults] = useState(false)
  const [expandedResultId, setExpandedResultId] = useState<string | null>(null)
  const ScoringIcon = scoringIcons[task.scoringMethod] ?? Beaker
  const sortedResults = [...results].sort((a, b) => b.score - a.score)

  return (
    <div
      className={`rounded-xl border transition-colors ${
        task.enabled
          ? 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800'
          : 'border-slate-100 dark:border-slate-700/50 bg-slate-50/50 dark:bg-slate-800/50'
      }`}
    >
      {/* Header */}
      <div className="flex items-center gap-3 px-4 py-3">
        <button
          onClick={onExpand}
          className="flex-shrink-0 text-slate-400 hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300 transition-colors"
        >
          {isExpanded ? (
            <ChevronDown className="w-4 h-4" strokeWidth={2} />
          ) : (
            <ChevronRight className="w-4 h-4" strokeWidth={2} />
          )}
        </button>

        {/* Toggle */}
        <button
          onClick={() => onToggle?.(!task.enabled)}
          className={`relative inline-flex flex-shrink-0 w-9 h-5 rounded-full transition-colors ${
            task.enabled
              ? 'bg-indigo-500 dark:bg-indigo-400'
              : 'bg-slate-300 dark:bg-slate-600'
          }`}
        >
          <span
            className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow-sm transition-transform ${
              task.enabled ? 'translate-x-4' : 'translate-x-0'
            }`}
          />
        </button>

        {/* Name + info */}
        <div className="flex-1 min-w-0 cursor-pointer" onClick={onExpand}>
          <div className="flex items-center gap-2 flex-wrap">
            <span
              className={`text-sm font-medium truncate ${
                task.enabled
                  ? 'text-slate-800 dark:text-slate-200'
                  : 'text-slate-400 dark:text-slate-500'
              }`}
            >
              {task.name}
            </span>
            {task.builtin && (
              <span className="px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider rounded bg-slate-100 text-slate-500 dark:bg-slate-700 dark:text-slate-400">
                Builtin
              </span>
            )}
          </div>
          <div className="flex items-center gap-2 mt-0.5 flex-wrap">
            <span
              className={`px-2 py-0.5 text-[10px] font-semibold rounded-md ${
                categoryColors[task.category] ?? 'bg-slate-100 text-slate-600'
              }`}
            >
              {task.category}
            </span>
            <span className="inline-flex items-center gap-1 text-[10px] text-slate-400 dark:text-slate-500">
              <ScoringIcon className="w-3 h-3" strokeWidth={2} />
              {task.scoringMethod}
            </span>
            <span className="text-[10px] text-slate-400 dark:text-slate-500">
              Diff {task.difficulty}/5
            </span>
            <span className="text-[10px] text-slate-400 dark:text-slate-500">
              {task.resultCount} result{task.resultCount !== 1 ? 's' : ''}
            </span>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1 flex-shrink-0">
          <button
            onClick={onEdit}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:text-slate-500 dark:hover:text-slate-300 dark:hover:bg-slate-700 transition-colors"
          >
            <Pencil className="w-3.5 h-3.5" strokeWidth={2} />
          </button>
          {!task.builtin && (
            <>
              {confirmDelete ? (
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => {
                      onDelete?.()
                      setConfirmDelete(false)
                    }}
                    className="px-2 py-1 text-[10px] font-medium rounded-md bg-red-500 text-white hover:bg-red-600 transition-colors"
                  >
                    Confirm
                  </button>
                  <button
                    onClick={() => setConfirmDelete(false)}
                    className="px-2 py-1 text-[10px] font-medium rounded-md text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
                  >
                    Cancel
                  </button>
                </div>
              ) : (
                <button
                  onClick={() => setConfirmDelete(true)}
                  className="p-1.5 rounded-lg text-slate-400 hover:text-red-500 hover:bg-red-50 dark:text-slate-500 dark:hover:text-red-400 dark:hover:bg-red-900/20 transition-colors"
                >
                  <Trash2 className="w-3.5 h-3.5" strokeWidth={2} />
                </button>
              )}
            </>
          )}
        </div>
      </div>

      {/* Expanded: prompt + results */}
      {isExpanded && (
        <div className="border-t border-slate-100 dark:border-slate-700/50">
          {/* Prompt preview */}
          <div className="px-4 py-3">
            <p className="text-xs text-slate-500 dark:text-slate-400 font-medium mb-1">Prompt</p>
            <p className="text-xs text-slate-600 dark:text-slate-300 leading-relaxed line-clamp-3 font-mono bg-slate-50 dark:bg-slate-900/40 rounded-lg p-3">
              {task.prompt}
            </p>
            {task.judgeCriteria && task.judgeCriteria.length > 0 && (
              <div className="mt-2">
                <p className="text-xs text-slate-500 dark:text-slate-400 font-medium mb-1.5">
                  Judge Criteria
                </p>
                <div className="space-y-1">
                  {task.judgeCriteria.map((c: JudgeCriterion, i: number) => (
                    <div key={i} className="flex gap-2 text-xs">
                      <span className="font-medium text-slate-700 dark:text-slate-300 flex-shrink-0 min-w-[100px]">
                        {c.name}
                      </span>
                      <span className="text-slate-500 dark:text-slate-400">
                        {c.description}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
            {task.expectedAnswer && (
              <div className="mt-2">
                <p className="text-xs text-slate-500 dark:text-slate-400 font-medium mb-1">
                  Expected Answer
                </p>
                <p className="text-xs font-mono text-slate-600 dark:text-slate-300">
                  {task.expectedAnswer}
                </p>
              </div>
            )}
          </div>

          {/* Results table — collapsible, collapsed by default */}
          <div className="px-4 pb-3">
            <button
              onClick={() => setShowResults(!showResults)}
              className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400 font-medium mb-2 hover:text-slate-700 dark:hover:text-slate-300 transition-colors"
            >
              {showResults ? (
                <ChevronDown className="w-3 h-3" strokeWidth={2} />
              ) : (
                <ChevronRight className="w-3 h-3" strokeWidth={2} />
              )}
              Model Results ({sortedResults.length})
            </button>
            {showResults && (
              sortedResults.length > 0 ? (
                <div className="space-y-1.5">
                  {sortedResults.map((r) => {
                    const hasResponse = !!r.response
                    const isResponseExpanded = expandedResultId === r.id
                    return (
                      <div key={r.id}>
                        <div className="flex items-center gap-3 text-xs">
                          {hasResponse ? (
                            <button
                              onClick={() => setExpandedResultId(isResponseExpanded ? null : r.id)}
                              className="flex-shrink-0 text-slate-400 hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300 transition-colors"
                            >
                              {isResponseExpanded ? (
                                <ChevronDown className="w-3 h-3" strokeWidth={2} />
                              ) : (
                                <ChevronRight className="w-3 h-3" strokeWidth={2} />
                              )}
                            </button>
                          ) : (
                            <span className="w-3 flex-shrink-0" />
                          )}
                          <span className="w-48 truncate font-mono text-slate-700 dark:text-slate-300 flex-shrink-0">
                            {r.modelName}
                          </span>
                          <div className="flex-1 flex items-center gap-2 min-w-0">
                            <div className="flex-1 h-2 bg-slate-100 dark:bg-slate-700 rounded-full overflow-hidden">
                              <div
                                className={`h-full rounded-full transition-all ${scoreBarColor(r.score)}`}
                                style={{ width: `${r.score * 100}%` }}
                              />
                            </div>
                            <span className="font-mono font-medium tabular-nums text-slate-700 dark:text-slate-300 w-10 text-right">
                              {Math.round(r.score * 100)}%
                            </span>
                          </div>
                          <span className="inline-flex items-center gap-1 text-slate-400 dark:text-slate-500 w-16 justify-end flex-shrink-0">
                            <Clock className="w-3 h-3" strokeWidth={2} />
                            {formatLatency(r.latencyMs)}
                          </span>
                          <span className="text-slate-400 dark:text-slate-500 w-28 text-right flex-shrink-0 hidden sm:block">
                            {formatTimestamp(r.timestamp)}
                          </span>
                        </div>
                        {isResponseExpanded && r.response && (
                          <ResponseBlock response={r.response} />
                        )}
                      </div>
                    )
                  })}
                </div>
              ) : (
                <p className="text-xs text-slate-400 dark:text-slate-500 italic">
                  No results yet. Run benchmarks to see model scores.
                </p>
              )
            )}
          </div>
        </div>
      )}
    </div>
  )
}
