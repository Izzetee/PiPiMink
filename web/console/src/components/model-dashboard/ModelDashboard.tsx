import { useState, useMemo, useCallback, useRef } from 'react'
import type {
  ModelDashboardProps,
  Model,
  StatFilter,
  BenchmarkCategory,
} from '@/types/model-dashboard'
import { StatCards } from './StatCards'
import { OperationProgress } from './OperationProgress'
import { BenchmarkPills } from './BenchmarkPills'
import { ModelDetailPanel } from './ModelDetailPanel'
import { FloatingActionBar } from './FloatingActionBar'
import { BenchmarkLog } from './BenchmarkLog'
import {
  Search,
  Radar,
  RefreshCw,
  Loader2,
  Brain,
  ChevronDown,
} from 'lucide-react'

function latencyColor(time: number): string {
  if (time < 1) return 'text-emerald-600 dark:text-emerald-400'
  if (time <= 5) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

function statusBadge(status: string): { bg: string; text: string } {
  switch (status) {
    case 'tagged':
      return {
        bg: 'bg-indigo-50 dark:bg-indigo-900/30',
        text: 'text-indigo-700 dark:text-indigo-300',
      }
    case 'discovered':
      return {
        bg: 'bg-amber-50 dark:bg-amber-900/30',
        text: 'text-amber-700 dark:text-amber-300',
      }
    default:
      return {
        bg: 'bg-slate-100 dark:bg-slate-700',
        text: 'text-slate-600 dark:text-slate-400',
      }
  }
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleDateString('en-GB', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}

function filterModels(models: Model[], filter: StatFilter): Model[] {
  switch (filter) {
    case 'enabled':
      return models.filter((m) => m.enabled)
    case 'taggedAndEnabled':
      return models.filter((m) => m.enabled && m.status === 'tagged')
    case 'disabled':
      return models.filter((m) => !m.enabled)
    case 'discovered':
      return models.filter((m) => m.status === 'discovered')
    case 'benchmarked':
      return models.filter(
        (m) => Object.keys(m.benchmarkScores).length > 0
      )
    default:
      return models
  }
}

type SortKey = 'name' | 'provider' | 'status' | 'avgResponseTime' | 'updatedAt'
type SortDir = 'asc' | 'desc'

function sortModels(models: Model[], key: SortKey, dir: SortDir): Model[] {
  return [...models].sort((a, b) => {
    let cmp = 0
    switch (key) {
      case 'name':
        cmp = a.name.localeCompare(b.name)
        break
      case 'provider':
        cmp = a.provider.localeCompare(b.provider)
        break
      case 'status':
        cmp = a.status.localeCompare(b.status)
        break
      case 'avgResponseTime':
        cmp = (a.avgResponseTime ?? 999) - (b.avgResponseTime ?? 999)
        break
      case 'updatedAt':
        cmp = new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime()
        break
    }
    return dir === 'asc' ? cmp : -cmp
  })
}

export function ModelDashboard({
  stats,
  models,
  activeOperation,
  expandedModelBenchmarks,
  benchmarkJudge,
  onToggleModel,
  onTagSelected,
  onBenchmarkSelected,
  onDiscoverModels,
  onRefresh,
  onToggleReasoning,
  onRetagModel,
  onRebenchmarkModel,
  onResetModel,
  onDeleteModel,
  onExpandModel,
  onCollapseModel,
}: ModelDashboardProps) {
  const [statFilter, setStatFilter] = useState<StatFilter>('all')
  const [search, setSearch] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [sortKey, setSortKey] = useState<SortKey>('name')
  const [sortDir, setSortDir] = useState<SortDir>('asc')
  const lastSelectedIndex = useRef<number | null>(null)

  const filteredModels = useMemo(() => {
    let result = filterModels(models, statFilter)
    if (search.trim()) {
      const q = search.toLowerCase()
      result = result.filter(
        (m) =>
          m.name.toLowerCase().includes(q) ||
          m.provider.toLowerCase().includes(q)
      )
    }
    return sortModels(result, sortKey, sortDir)
  }, [models, statFilter, search, sortKey, sortDir])

  const toggleSelect = useCallback(
    (id: string, index: number, shiftKey: boolean) => {
      setSelected((prev) => {
        const next = new Set(prev)
        if (shiftKey && lastSelectedIndex.current !== null) {
          const start = Math.min(lastSelectedIndex.current, index)
          const end = Math.max(lastSelectedIndex.current, index)
          for (let i = start; i <= end; i++) {
            const m = filteredModels[i]
            if (m) next.add(m.id)
          }
        } else {
          if (next.has(id)) next.delete(id)
          else next.add(id)
        }
        lastSelectedIndex.current = index
        return next
      })
    },
    [filteredModels]
  )

  const toggleSelectAll = useCallback(() => {
    setSelected((prev) => {
      if (prev.size === filteredModels.length) return new Set()
      return new Set(filteredModels.map((m) => m.id))
    })
  }, [filteredModels])

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  const handleRowClick = (model: Model) => {
    if (expandedId === model.id) {
      setExpandedId(null)
      onCollapseModel?.()
    } else {
      setExpandedId(model.id)
      onExpandModel?.(model.id)
    }
  }

  const allSelected =
    filteredModels.length > 0 && selected.size === filteredModels.length

  const SortHeader = ({
    label,
    sortable,
    className = '',
  }: {
    label: string
    sortable: SortKey
    className?: string
  }) => (
    <th
      className={`px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 cursor-pointer select-none hover:text-slate-600 dark:hover:text-slate-300 transition-colors ${className}`}
      onClick={() => handleSort(sortable)}
    >
      <span className="inline-flex items-center gap-1">
        {label}
        {sortKey === sortable && (
          <ChevronDown
            className={`w-3 h-3 transition-transform ${
              sortDir === 'asc' ? 'rotate-180' : ''
            }`}
            strokeWidth={2}
          />
        )}
      </span>
    </th>
  )

  return (
    <div className="p-5 lg:p-6 space-y-5">
      {activeOperation && activeOperation.status === 'running' && (
        <OperationProgress operation={activeOperation} />
      )}

      <StatCards
        stats={stats}
        activeFilter={statFilter}
        onFilterChange={setStatFilter}
      />

      <div className="flex flex-col sm:flex-row sm:items-center gap-3">
        <div className="relative flex-1 max-w-md">
          <Search
            className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400 dark:text-slate-500"
            strokeWidth={1.75}
          />
          <input
            type="text"
            placeholder="Search models or providers..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-3 py-2 text-sm rounded-lg border border-slate-200 bg-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-300 dark:border-slate-600 dark:bg-slate-800 dark:placeholder:text-slate-500 dark:focus:ring-indigo-500/30 dark:focus:border-indigo-500/50 dark:text-slate-200 transition-colors"
          />
        </div>

        <div className="flex items-center gap-1.5">
          {(['all', 'enabled', 'disabled'] as StatFilter[]).map(
            (f) => (
              <button
                key={f}
                onClick={() => setStatFilter(f)}
                className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-colors ${
                  statFilter === f
                    ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300'
                    : 'text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700'
                }`}
              >
                {f === 'all'
                  ? 'All'
                  : f === 'enabled'
                    ? 'Enabled'
                    : 'Disabled'}
              </button>
            )
          )}
        </div>

        <div className="flex items-center gap-2 sm:ml-auto">
          <button
            onClick={onDiscoverModels}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 transition-colors shadow-sm"
          >
            <Radar className="w-4 h-4" strokeWidth={1.75} />
            Discover
          </button>
          <button
            onClick={onRefresh}
            className="inline-flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
          >
            <RefreshCw className="w-4 h-4" strokeWidth={1.75} />
            Refresh
          </button>
        </div>
      </div>

      <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 overflow-x-auto">
        {filteredModels.length === 0 ? (
          <div className="py-16 text-center">
            <Radar
              className="w-10 h-10 text-slate-300 dark:text-slate-600 mx-auto mb-3"
              strokeWidth={1}
            />
            <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
              No models found
            </p>
            <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">
              {search
                ? 'Try a different search term'
                : 'Run Discover to scan your providers'}
            </p>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-100 dark:border-slate-700/50">
                <th className="w-10 px-4 py-3">
                  <input
                    type="checkbox"
                    checked={allSelected}
                    onChange={toggleSelectAll}
                    className="w-3.5 h-3.5 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500/30 dark:border-slate-600 dark:bg-slate-700 cursor-pointer"
                  />
                </th>
                <th className="w-14 px-4 py-3 text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">
                  On/Off
                </th>
                <SortHeader label="Model" sortable="name" className="min-w-[180px]" />
                <SortHeader label="Provider" sortable="provider" />
                <SortHeader label="Status" sortable="status" />
                <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 w-10">
                  <Brain className="w-3.5 h-3.5" strokeWidth={1.75} />
                </th>
                <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500 min-w-[260px]">
                  Benchmarks
                </th>
                <SortHeader label="Latency" sortable="avgResponseTime" />
                <SortHeader label="Updated" sortable="updatedAt" />
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-50 dark:divide-slate-700/30">
              {filteredModels.map((model, index) => {
                const isSelected = selected.has(model.id)
                const isExpanded = expandedId === model.id
                const badge = statusBadge(model.status)
                const isBeingProcessed =
                  activeOperation?.status === 'running' &&
                  activeOperation.currentModel === model.name

                return (
                  <ModelRow
                    key={model.id}
                    model={model}
                    index={index}
                    isSelected={isSelected}
                    isExpanded={isExpanded}
                    isBeingProcessed={isBeingProcessed}
                    badge={badge}
                    expandedModelBenchmarks={
                      isExpanded ? expandedModelBenchmarks : null
                    }
                    benchmarkJudge={benchmarkJudge}
                    onToggleSelect={toggleSelect}
                    onToggleModel={onToggleModel}
                    onRowClick={handleRowClick}
                    onRetagModel={onRetagModel}
                    onRebenchmarkModel={onRebenchmarkModel}
                    onToggleReasoning={onToggleReasoning}
                    onResetModel={onResetModel}
                    onDeleteModel={(id) => {
                      onDeleteModel?.(id)
                      setExpandedId(null)
                    }}
                    onCollapseModel={() => {
                      setExpandedId(null)
                      onCollapseModel?.()
                    }}
                  />
                )
              })}
            </tbody>
          </table>
        )}
      </div>

      <FloatingActionBar
        selectedCount={selected.size}
        onTagSelected={() => onTagSelected?.([...selected])}
        onBenchmarkSelected={() => onBenchmarkSelected?.([...selected])}
        onDeselectAll={() => setSelected(new Set())}
      />

      {activeOperation?.type === 'benchmark' &&
        activeOperation.status === 'running' &&
        (activeOperation.logEntries?.length ?? 0) > 0 && (
          <BenchmarkLog logEntries={activeOperation.logEntries!} />
        )}
    </div>
  )
}

interface ModelRowProps {
  model: Model
  index: number
  isSelected: boolean
  isExpanded: boolean
  isBeingProcessed: boolean
  badge: { bg: string; text: string }
  expandedModelBenchmarks: ModelDashboardProps['expandedModelBenchmarks']
  benchmarkJudge?: string
  onToggleSelect: (id: string, index: number, shiftKey: boolean) => void
  onToggleModel?: ModelDashboardProps['onToggleModel']
  onRowClick: (model: Model) => void
  onRetagModel?: ModelDashboardProps['onRetagModel']
  onRebenchmarkModel?: ModelDashboardProps['onRebenchmarkModel']
  onToggleReasoning?: ModelDashboardProps['onToggleReasoning']
  onResetModel?: ModelDashboardProps['onResetModel']
  onDeleteModel?: ModelDashboardProps['onDeleteModel']
  onCollapseModel: () => void
}

function ModelRow({
  model,
  index,
  isSelected,
  isExpanded,
  isBeingProcessed,
  badge,
  expandedModelBenchmarks,
  benchmarkJudge,
  onToggleSelect,
  onToggleModel,
  onRowClick,
  onRetagModel,
  onRebenchmarkModel,
  onToggleReasoning,
  onResetModel,
  onDeleteModel,
  onCollapseModel,
}: ModelRowProps) {
  return (
    <>
      <tr
        className={`group cursor-pointer transition-colors ${
          isExpanded
            ? 'bg-indigo-50/40 dark:bg-indigo-950/20'
            : isSelected
              ? 'bg-indigo-50/30 dark:bg-indigo-950/10'
              : 'hover:bg-slate-50/80 dark:hover:bg-slate-700/20'
        }`}
        onClick={() => onRowClick(model)}
      >
        <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
          <input
            type="checkbox"
            checked={isSelected}
            onChange={(e) => {
              const nativeEvent = e.nativeEvent as MouseEvent
              onToggleSelect(model.id, index, nativeEvent.shiftKey)
            }}
            className="w-3.5 h-3.5 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500/30 dark:border-slate-600 dark:bg-slate-700 cursor-pointer"
          />
        </td>

        <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
          <button
            onClick={() => onToggleModel?.(model.id, !model.enabled)}
            className={`relative w-9 h-5 rounded-full transition-colors ${
              model.enabled
                ? 'bg-indigo-500 dark:bg-indigo-400'
                : 'bg-slate-200 dark:bg-slate-600'
            }`}
          >
            <span
              className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow-sm transition-transform ${
                model.enabled ? 'translate-x-4' : 'translate-x-0'
              }`}
            />
          </button>
        </td>

        <td className="px-4 py-3">
          <span className="font-medium text-slate-800 dark:text-slate-200 font-mono text-xs">
            {model.name}
          </span>
        </td>

        <td className="px-4 py-3 text-slate-500 dark:text-slate-400 text-xs">
          {model.provider}
        </td>

        <td className="px-4 py-3">
          <div className="flex items-center gap-2">
            <span
              className={`px-2 py-0.5 text-[11px] font-medium rounded-full ${badge.bg} ${badge.text}`}
            >
              {model.status}
            </span>
            {isBeingProcessed && (
              <Loader2
                className="w-3.5 h-3.5 text-indigo-500 dark:text-indigo-400 animate-spin"
                strokeWidth={2}
              />
            )}
          </div>
        </td>

        <td className="px-4 py-3">
          {model.hasReasoning && (
            <Brain
              className="w-4 h-4 text-indigo-500 dark:text-indigo-400"
              strokeWidth={1.75}
            />
          )}
        </td>

        <td className="px-4 py-3">
          <BenchmarkPills
            scores={model.benchmarkScores as Partial<Record<BenchmarkCategory, number>>}
          />
        </td>

        <td className="px-4 py-3">
          {model.avgResponseTime !== null ? (
            <span
              className={`font-mono text-xs tabular-nums font-medium ${latencyColor(model.avgResponseTime)}`}
            >
              {model.avgResponseTime.toFixed(1)}s
            </span>
          ) : (
            <span className="text-xs text-slate-400 dark:text-slate-500">
              —
            </span>
          )}
        </td>

        <td className="px-4 py-3 text-xs text-slate-500 dark:text-slate-400 whitespace-nowrap">
          {formatDate(model.updatedAt)}
        </td>
      </tr>

      {isExpanded && (
        <ModelDetailPanel
          model={model}
          benchmarks={expandedModelBenchmarks}
          benchmarkJudge={benchmarkJudge}
          onRetagModel={onRetagModel}
          onRebenchmarkModel={onRebenchmarkModel}
          onToggleReasoning={onToggleReasoning}
          onResetModel={onResetModel}
          onDeleteModel={onDeleteModel}
          onCollapse={onCollapseModel}
        />
      )}
    </>
  )
}
