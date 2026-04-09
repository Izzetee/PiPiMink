import { useState } from 'react'
import type { RoutingDecision } from '@/types/analytics'
import {
  ChevronLeft,
  ChevronRight,
  X,
  CheckCircle2,
  XCircle,
  Clock,
  Cpu,
  MessageSquare,
  Tag,
  Lightbulb,
  Server,
  Gauge,
  Zap,
} from 'lucide-react'

interface RoutingDecisionsListProps {
  decisions: RoutingDecision[]
  onViewDecision?: (id: string) => void
  onNextPage?: () => void
  onPrevPage?: () => void
}

function formatTime(ts: string): string {
  const d = new Date(ts)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatDate(ts: string): string {
  const d = new Date(ts)
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

function latencyColor(ms: number): string {
  if (ms < 200) return 'text-emerald-600 dark:text-emerald-400'
  if (ms < 500) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

export function RoutingDecisionsList({
  decisions,
  onViewDecision,
  onNextPage,
  onPrevPage,
}: RoutingDecisionsListProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const selected = decisions.find((d) => d.id === selectedId)

  function handleSelect(id: string) {
    setSelectedId(id)
    onViewDecision?.(id)
  }

  return (
    <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 overflow-hidden">
      <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-700/50 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
            Your Routing Decisions
          </h3>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            Only your prompts and decisions are shown
          </p>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={onPrevPage}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            <ChevronLeft className="w-4 h-4" strokeWidth={2} />
          </button>
          <button
            onClick={onNextPage}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            <ChevronRight className="w-4 h-4" strokeWidth={2} />
          </button>
        </div>
      </div>

      <div className="flex">
        {/* Decision list */}
        <div className={`flex-1 min-w-0 divide-y divide-slate-100 dark:divide-slate-700/50 ${selected ? 'hidden sm:block sm:max-w-[55%] lg:max-w-[60%]' : ''}`}>
          {decisions.map((d) => (
            <button
              key={d.id}
              onClick={() => handleSelect(d.id)}
              className={`w-full text-left px-4 py-3 hover:bg-slate-50/80 dark:hover:bg-slate-700/20 transition-colors ${
                selectedId === d.id ? 'bg-indigo-50/60 dark:bg-indigo-950/20 border-l-2 border-indigo-500' : 'border-l-2 border-transparent'
              }`}
            >
              <div className="flex items-start gap-3">
                <div className="flex-shrink-0 mt-0.5">
                  {d.status === 'success' ? (
                    <CheckCircle2 className="w-4 h-4 text-emerald-500" strokeWidth={2} />
                  ) : (
                    <XCircle className="w-4 h-4 text-red-500" strokeWidth={2} />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-slate-800 dark:text-slate-200 truncate">
                    {d.promptSnippet}
                  </p>
                  <div className="flex flex-wrap items-center gap-x-4 gap-y-1 mt-2">
                    <span className="text-[11px] font-mono text-slate-500 dark:text-slate-400">
                      {formatDate(d.timestamp)} {formatTime(d.timestamp)}
                    </span>
                    <span className="inline-block w-px h-3 bg-slate-200 dark:bg-slate-600" />
                    <span className="text-[11px] font-mono font-medium text-indigo-600 dark:text-indigo-400 truncate max-w-[180px]">
                      {d.selectedModel}
                    </span>
                    <span className="inline-block w-px h-3 bg-slate-200 dark:bg-slate-600" />
                    <span className={`text-[11px] font-mono font-medium ${latencyColor(d.latencyMs)}`}>
                      {d.latencyMs}ms
                    </span>
                    {d.cacheHit && (
                      <>
                        <span className="inline-block w-px h-3 bg-slate-200 dark:bg-slate-600" />
                        <span className="text-[10px] font-medium text-emerald-600 dark:text-emerald-400">
                          cached
                        </span>
                      </>
                    )}
                  </div>
                  <div className="flex flex-wrap gap-1 mt-1.5">
                    {d.analyzedTags.map((tag) => (
                      <span
                        key={tag}
                        className="px-1.5 py-0.5 text-[10px] font-medium rounded bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            </button>
          ))}
        </div>

        {/* Detail panel */}
        {selected && (
          <div className="flex-1 sm:flex-none sm:w-[45%] lg:w-[40%] border-l border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
            <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-700/50 flex items-center justify-between">
              <h4 className="text-xs font-semibold text-slate-800 dark:text-slate-200 uppercase tracking-wide">
                Decision Detail
              </h4>
              <button
                onClick={() => setSelectedId(null)}
                className="p-1 rounded-md text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-200/60 dark:hover:bg-slate-700 transition-colors"
              >
                <X className="w-3.5 h-3.5" strokeWidth={2} />
              </button>
            </div>
            <div className="p-4 space-y-4 overflow-y-auto max-h-[calc(100vh-200px)]">
              {/* Status + Timestamp + Cache */}
              <div className="flex flex-wrap items-center gap-2">
                {selected.status === 'success' ? (
                  <span className="inline-flex items-center gap-1 text-xs font-medium text-emerald-600 dark:text-emerald-400">
                    <CheckCircle2 className="w-3.5 h-3.5" strokeWidth={2} />
                    Success
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1 text-xs font-medium text-red-600 dark:text-red-400">
                    <XCircle className="w-3.5 h-3.5" strokeWidth={2} />
                    Error
                  </span>
                )}
                <span className="text-xs text-slate-300 dark:text-slate-600">|</span>
                <span className="text-xs font-mono text-slate-600 dark:text-slate-300">
                  {formatDate(selected.timestamp)} {formatTime(selected.timestamp)}
                </span>
                <span className="text-xs text-slate-300 dark:text-slate-600">|</span>
                <span className={`inline-flex items-center gap-1 text-[11px] font-medium px-1.5 py-0.5 rounded ${
                  selected.cacheHit
                    ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                    : 'bg-slate-100 text-slate-500 dark:bg-slate-700 dark:text-slate-400'
                }`}>
                  <Zap className="w-3 h-3" strokeWidth={2} />
                  {selected.cacheHit ? 'Cache hit' : 'Cache miss'}
                </span>
              </div>

              {/* Full Prompt */}
              <div>
                <div className="flex items-center gap-1.5 mb-1.5">
                  <MessageSquare className="w-3.5 h-3.5 text-slate-500 dark:text-slate-400" strokeWidth={2} />
                  <span className="text-xs font-semibold text-slate-700 dark:text-slate-200">Prompt</span>
                </div>
                <p className="text-xs text-slate-800 dark:text-slate-200 leading-relaxed bg-slate-50 dark:bg-slate-900/50 rounded-lg p-3 border border-slate-200 dark:border-slate-700 font-mono whitespace-pre-wrap max-h-32 overflow-y-auto">
                  {selected.fullPrompt}
                </p>
              </div>

              {/* Tags with relevance scores */}
              <div>
                <div className="flex items-center gap-1.5 mb-1.5">
                  <Tag className="w-3.5 h-3.5 text-slate-500 dark:text-slate-400" strokeWidth={2} />
                  <span className="text-xs font-semibold text-slate-700 dark:text-slate-200">Matching Tags</span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {selected.analyzedTags.map((tag) => (
                    <span
                      key={tag}
                      className="inline-flex items-center gap-1.5 px-2 py-1 text-[11px] font-medium rounded-md bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300 border border-indigo-100 dark:border-indigo-800/40"
                    >
                      {tag}
                      {selected.tagRelevance[tag] != null && (
                        <span className="text-[10px] font-mono bg-indigo-100 dark:bg-indigo-800/50 text-indigo-600 dark:text-indigo-300 px-1 rounded">
                          {selected.tagRelevance[tag]}/10
                        </span>
                      )}
                    </span>
                  ))}
                </div>
              </div>

              {/* Routing Reason — highlighted */}
              <div className="rounded-lg bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800/40 p-3">
                <div className="flex items-center gap-1.5 mb-1.5">
                  <Lightbulb className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" strokeWidth={2} />
                  <span className="text-xs font-semibold text-amber-800 dark:text-amber-200">Routing Reason</span>
                </div>
                <p className="text-sm text-amber-900 dark:text-amber-100 leading-relaxed font-medium">
                  {selected.routingReason}
                </p>
              </div>

              {/* Selected Model + Provider */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <div className="flex items-center gap-1.5 mb-1.5">
                    <Cpu className="w-3.5 h-3.5 text-slate-500 dark:text-slate-400" strokeWidth={2} />
                    <span className="text-xs font-semibold text-slate-700 dark:text-slate-200">Model</span>
                  </div>
                  <span className="text-xs font-mono font-medium text-slate-900 dark:text-slate-100">
                    {selected.selectedModel}
                  </span>
                </div>
                <div>
                  <div className="flex items-center gap-1.5 mb-1.5">
                    <Server className="w-3.5 h-3.5 text-slate-500 dark:text-slate-400" strokeWidth={2} />
                    <span className="text-xs font-semibold text-slate-700 dark:text-slate-200">Provider</span>
                  </div>
                  <span className="text-xs font-mono font-medium text-slate-900 dark:text-slate-100">
                    {selected.provider}
                  </span>
                </div>
              </div>

              {/* Timing row: Evaluation + Response */}
              <div className="grid grid-cols-2 gap-3">
                <div className="rounded-lg bg-slate-50 dark:bg-slate-900/50 border border-slate-200 dark:border-slate-700 p-3">
                  <div className="flex items-center gap-1.5 mb-1">
                    <Gauge className="w-3.5 h-3.5 text-slate-500 dark:text-slate-400" strokeWidth={2} />
                    <span className="text-[11px] font-medium text-slate-600 dark:text-slate-300">Eval Time</span>
                  </div>
                  <span className="text-base font-semibold font-mono text-slate-800 dark:text-slate-100">
                    {selected.evaluationTimeMs >= 1000
                      ? `${(selected.evaluationTimeMs / 1000).toFixed(1)}s`
                      : `${selected.evaluationTimeMs}ms`}
                  </span>
                  <p className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 font-mono">
                    via {selected.evaluatorModel}
                  </p>
                </div>
                <div className="rounded-lg bg-slate-50 dark:bg-slate-900/50 border border-slate-200 dark:border-slate-700 p-3">
                  <div className="flex items-center gap-1.5 mb-1">
                    <Clock className="w-3.5 h-3.5 text-slate-500 dark:text-slate-400" strokeWidth={2} />
                    <span className="text-[11px] font-medium text-slate-600 dark:text-slate-300">Response</span>
                  </div>
                  <span className={`text-base font-semibold font-mono ${latencyColor(selected.latencyMs)}`}>
                    {selected.latencyMs >= 1000
                      ? `${(selected.latencyMs / 1000).toFixed(1)}s`
                      : `${selected.latencyMs}ms`}
                  </span>
                  <p className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5">
                    Total: {((selected.evaluationTimeMs + selected.latencyMs) / 1000).toFixed(1)}s
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
