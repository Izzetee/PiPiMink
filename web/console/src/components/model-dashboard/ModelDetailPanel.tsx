import { useState } from 'react'
import type {
  Model,
  DetailTab,
  ExpandedModelBenchmarks,
  BenchmarkCategory,
} from '@/types/model-dashboard'
import { Tag, FlaskConical, Brain, ChevronUp, RotateCcw, Trash2 } from 'lucide-react'

const categoryOrder: BenchmarkCategory[] = [
  'coding',
  'creative',
  'factual',
  'instruction',
  'reasoning',
  'summarization',
]

function scoreColor(score: number): string {
  if (score >= 0.7) return 'text-emerald-600 dark:text-emerald-400'
  if (score >= 0.4) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

function scoreBg(score: number): string {
  if (score >= 0.7) return 'bg-emerald-500 dark:bg-emerald-400'
  if (score >= 0.4) return 'bg-amber-500 dark:bg-amber-400'
  return 'bg-red-500 dark:bg-red-400'
}

interface ModelDetailPanelProps {
  model: Model
  benchmarks: ExpandedModelBenchmarks | null
  benchmarkJudge?: string
  onRetagModel?: (modelId: string) => void
  onRebenchmarkModel?: (modelId: string) => void
  onToggleReasoning?: (modelId: string, hasReasoning: boolean) => void
  onResetModel?: (modelId: string) => void
  onDeleteModel?: (modelId: string) => void
  onCollapse?: () => void
}

export function ModelDetailPanel({
  model,
  benchmarks,
  benchmarkJudge,
  onRetagModel,
  onRebenchmarkModel,
  onToggleReasoning,
  onResetModel,
  onDeleteModel,
  onCollapse,
}: ModelDetailPanelProps) {
  const [activeTab, setActiveTab] = useState<DetailTab>('overview')
  const [confirmAction, setConfirmAction] = useState<'reset' | 'delete' | null>(null)

  const tabs: { key: DetailTab; label: string }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'benchmarks', label: 'Benchmarks' },
  ]

  const resultsByCategory = benchmarks
    ? benchmarks.results.reduce<Record<string, typeof benchmarks.results>>(
        (acc, r) => {
          const cat = r.category
          if (!acc[cat]) acc[cat] = []
          acc[cat].push(r)
          return acc
        },
        {}
      )
    : {}

  return (
    <tr>
      <td
        colSpan={9}
        className="p-0 border-b border-slate-100 dark:border-slate-700/50"
      >
        <div className="bg-slate-50/50 dark:bg-slate-800/50 border-t border-slate-100 dark:border-slate-700/30">
          <div className="flex items-center justify-between px-5 border-b border-slate-200/60 dark:border-slate-700/40">
            <div className="flex gap-0">
              {tabs.map((tab) => (
                <button
                  key={tab.key}
                  onClick={() => setActiveTab(tab.key)}
                  className={`
                    px-4 py-2.5 text-sm font-medium transition-colors relative
                    ${activeTab === tab.key
                      ? 'text-indigo-600 dark:text-indigo-400'
                      : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300'
                    }
                  `}
                >
                  {tab.label}
                  {activeTab === tab.key && (
                    <span className="absolute bottom-0 left-2 right-2 h-[2px] bg-indigo-500 dark:bg-indigo-400 rounded-full" />
                  )}
                </button>
              ))}
            </div>
            <button
              onClick={onCollapse}
              className="p-1.5 rounded-md text-slate-400 hover:text-slate-600 hover:bg-slate-200/50 dark:text-slate-500 dark:hover:text-slate-300 dark:hover:bg-slate-700/50 transition-colors"
            >
              <ChevronUp className="w-4 h-4" strokeWidth={1.75} />
            </button>
          </div>

          <div className="p-5">
            {activeTab === 'overview' && (
              <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                <div>
                  <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 mb-3">
                    Benchmark Scores
                  </h4>
                  {categoryOrder.map((cat) => {
                    const score = model.benchmarkScores[cat]
                    if (score === undefined) return null
                    return (
                      <div key={cat} className="flex items-center gap-3 mb-2">
                        <span className="text-xs text-slate-500 dark:text-slate-400 w-24 capitalize">
                          {cat}
                        </span>
                        <div className="flex-1 h-1.5 rounded-full bg-slate-200 dark:bg-slate-700 overflow-hidden">
                          <div
                            className={`h-full rounded-full transition-all ${scoreBg(score)}`}
                            style={{ width: `${score * 100}%` }}
                          />
                        </div>
                        <span
                          className={`text-xs font-mono tabular-nums font-medium w-10 text-right ${scoreColor(score)}`}
                        >
                          {Math.round(score * 100)}%
                        </span>
                      </div>
                    )
                  })}
                  {Object.keys(model.benchmarkScores).length === 0 && (
                    <p className="text-sm text-slate-400 dark:text-slate-500 italic">
                      No benchmarks yet
                    </p>
                  )}
                  {Object.keys(model.benchmarkScores).length > 0 && benchmarkJudge && (
                    <p className="mt-3 text-[11px] text-slate-400 dark:text-slate-500">
                      Judge: <span className="font-mono">{benchmarkJudge}</span>
                    </p>
                  )}
                </div>

                <div>
                  <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 mb-3">
                    Capability Tags
                  </h4>
                  {model.tags.strengths.length > 0 && (
                    <div className="mb-3">
                      <p className="text-[11px] font-medium text-emerald-600 dark:text-emerald-400 mb-1.5 uppercase tracking-wider">
                        Strengths
                      </p>
                      <div className="flex flex-wrap gap-1.5">
                        {model.tags.strengths.map((tag) => (
                          <span
                            key={tag}
                            className="px-2 py-0.5 text-xs rounded-full bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                  {model.tags.weaknesses.length > 0 && (
                    <div>
                      <p className="text-[11px] font-medium text-red-500 dark:text-red-400 mb-1.5 uppercase tracking-wider">
                        Weaknesses
                      </p>
                      <div className="flex flex-wrap gap-1.5">
                        {model.tags.weaknesses.map((tag) => (
                          <span
                            key={tag}
                            className="px-2 py-0.5 text-xs rounded-full bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300"
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                  {model.tags.strengths.length === 0 &&
                    model.tags.weaknesses.length === 0 && (
                      <p className="text-sm text-slate-400 dark:text-slate-500 italic">
                        Not tagged yet
                      </p>
                    )}
                  {model.taggedBy && (
                    <p className="mt-3 text-[11px] text-slate-400 dark:text-slate-500">
                      Self-assessed
                    </p>
                  )}
                </div>

                <div>
                  <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 mb-3">
                    Info
                  </h4>
                  <dl className="space-y-2 text-sm mb-5">
                    <div className="flex justify-between">
                      <dt className="text-slate-500 dark:text-slate-400">Provider</dt>
                      <dd className="font-mono text-slate-700 dark:text-slate-300 text-xs">
                        {model.provider}
                      </dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-slate-500 dark:text-slate-400">Status</dt>
                      <dd className="text-slate-700 dark:text-slate-300 capitalize text-xs">
                        {model.status}
                      </dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-slate-500 dark:text-slate-400">Reasoning</dt>
                      <dd className="text-slate-700 dark:text-slate-300 text-xs">
                        {model.hasReasoning ? 'Yes' : 'No'}
                      </dd>
                    </div>
                    {model.avgResponseTime !== null && (
                      <div className="flex justify-between">
                        <dt className="text-slate-500 dark:text-slate-400">Avg Latency</dt>
                        <dd className="font-mono text-slate-700 dark:text-slate-300 text-xs">
                          {model.avgResponseTime.toFixed(1)}s
                        </dd>
                      </div>
                    )}
                  </dl>

                  <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 mb-2.5">
                    Quick Actions
                  </h4>
                  <div className="flex flex-wrap gap-2">
                    <button
                      onClick={() => onRetagModel?.(model.id)}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
                    >
                      <Tag className="w-3.5 h-3.5" strokeWidth={1.75} />
                      Re-tag
                    </button>
                    <button
                      onClick={() => onRebenchmarkModel?.(model.id)}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
                    >
                      <FlaskConical className="w-3.5 h-3.5" strokeWidth={1.75} />
                      Re-benchmark
                    </button>
                    <button
                      onClick={() =>
                        onToggleReasoning?.(model.id, !model.hasReasoning)
                      }
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-100 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
                    >
                      <Brain className="w-3.5 h-3.5" strokeWidth={1.75} />
                      {model.hasReasoning ? 'Remove reasoning' : 'Mark reasoning'}
                    </button>
                  </div>

                  {/* Danger zone */}
                  <div className="mt-4 pt-4 border-t border-red-200/50 dark:border-red-500/10">
                    <h4 className="text-xs font-semibold uppercase tracking-wider text-red-400 dark:text-red-500 mb-2.5">
                      Danger Zone
                    </h4>
                    <div className="flex flex-wrap gap-2">
                      {confirmAction === 'reset' ? (
                        <div className="flex items-center gap-1.5">
                          <span className="text-[10px] text-red-600 dark:text-red-400">Reset all data?</span>
                          <button
                            onClick={() => { onResetModel?.(model.id); setConfirmAction(null) }}
                            className="px-2 py-1 text-[10px] font-medium rounded-md bg-red-500 text-white hover:bg-red-600 transition-colors"
                          >
                            Confirm
                          </button>
                          <button
                            onClick={() => setConfirmAction(null)}
                            className="px-2 py-1 text-[10px] font-medium rounded-md text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
                          >
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => setConfirmAction('reset')}
                          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border border-red-200 text-red-600 hover:bg-red-50 dark:border-red-500/30 dark:text-red-400 dark:hover:bg-red-900/20 transition-colors"
                        >
                          <RotateCcw className="w-3.5 h-3.5" strokeWidth={1.75} />
                          Reset Model
                        </button>
                      )}
                      {confirmAction === 'delete' ? (
                        <div className="flex items-center gap-1.5">
                          <span className="text-[10px] text-red-600 dark:text-red-400">Delete permanently?</span>
                          <button
                            onClick={() => { onDeleteModel?.(model.id); setConfirmAction(null) }}
                            className="px-2 py-1 text-[10px] font-medium rounded-md bg-red-500 text-white hover:bg-red-600 transition-colors"
                          >
                            Confirm
                          </button>
                          <button
                            onClick={() => setConfirmAction(null)}
                            className="px-2 py-1 text-[10px] font-medium rounded-md text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
                          >
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => setConfirmAction('delete')}
                          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border border-red-200 text-red-600 hover:bg-red-50 dark:border-red-500/30 dark:text-red-400 dark:hover:bg-red-900/20 transition-colors"
                        >
                          <Trash2 className="w-3.5 h-3.5" strokeWidth={1.75} />
                          Delete Model
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'benchmarks' && (
              <div>
                {Object.keys(resultsByCategory).length > 0 && (benchmarks?.benchmarkJudge || benchmarkJudge) && (
                  <p className="text-[11px] text-slate-400 dark:text-slate-500 mb-3">
                    Judge: <span className="font-mono">{benchmarks?.benchmarkJudge || benchmarkJudge}</span>
                  </p>
                )}
                {Object.keys(resultsByCategory).length === 0 ? (
                  <p className="text-sm text-slate-400 dark:text-slate-500 italic py-4 text-center">
                    No detailed benchmark results available
                  </p>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                    {categoryOrder.map((cat) => {
                      const results = resultsByCategory[cat]
                      if (!results) return null
                      const avgScore =
                        results.reduce((s, r) => s + r.score, 0) / results.length
                      return (
                        <div
                          key={cat}
                          className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 overflow-hidden"
                        >
                          <div className="px-3.5 py-2.5 border-b border-slate-100 dark:border-slate-700/50 flex items-center justify-between">
                            <span className="text-sm font-medium text-slate-700 dark:text-slate-300 capitalize">
                              {cat.replace('-', ' ')}
                            </span>
                            <span
                              className={`text-xs font-mono font-semibold tabular-nums ${scoreColor(avgScore)}`}
                            >
                              {Math.round(avgScore * 100)}%
                            </span>
                          </div>
                          <div className="divide-y divide-slate-50 dark:divide-slate-700/30">
                            {results.map((r) => (
                              <div
                                key={r.taskId}
                                className="px-3.5 py-2 flex items-center justify-between"
                              >
                                <span className="text-xs font-mono text-slate-600 dark:text-slate-400 truncate mr-3">
                                  {r.taskId}
                                </span>
                                <div className="flex items-center gap-3 shrink-0">
                                  <span className="text-[11px] text-slate-400 dark:text-slate-500 font-mono tabular-nums">
                                    {r.latency.toFixed(1)}s
                                  </span>
                                  <div className="flex items-center gap-1.5">
                                    <div className="w-12 h-1 rounded-full bg-slate-100 dark:bg-slate-700 overflow-hidden">
                                      <div
                                        className={`h-full rounded-full ${scoreBg(r.score)}`}
                                        style={{ width: `${r.score * 100}%` }}
                                      />
                                    </div>
                                    <span
                                      className={`text-xs font-mono tabular-nums font-medium w-8 text-right ${scoreColor(r.score)}`}
                                    >
                                      {Math.round(r.score * 100)}%
                                    </span>
                                  </div>
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </td>
    </tr>
  )
}
