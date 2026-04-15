import { useEffect, useState, useCallback, useRef } from 'react'
import { ModelDashboard } from '@/components/model-dashboard'
import {
  fetchModels,
  toggleModel,
  discoverModels,
  tagModels,
  fetchTagStatus,
  benchmarkModels,
  fetchBenchmarkStatus,
  resetModel,
  deleteModel,
  updateModelReasoning,
  fetchModelBenchmarkResults,
} from '@/api'
import type {
  Model,
  DashboardStats,
  Operation,
  ExpandedModelBenchmarks,
} from '@/types/model-dashboard'
import { Loader2, AlertCircle, Key } from 'lucide-react'
import { getApiKey, setApiKey } from '@/api'

const EMPTY_STATS: DashboardStats = {
  total: 0,
  enabled: 0,
  taggedAndEnabled: 0,
  disabled: 0,
  discovered: 0,
  benchmarked: 0,
}

export function ModelsPage() {
  const [models, setModels] = useState<Model[]>([])
  const [stats, setStats] = useState<DashboardStats>(EMPTY_STATS)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeOperation, setActiveOperation] = useState<Operation | null>(null)
  const [expandedBenchmarks, setExpandedBenchmarks] = useState<ExpandedModelBenchmarks | null>(null)
  const [benchmarkJudge, setBenchmarkJudge] = useState<string | undefined>()
  const [apiKeyInput, setApiKeyInput] = useState('')
  const [showApiKeyPrompt, setShowApiKeyPrompt] = useState(false)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const loadModels = useCallback(async () => {
    try {
      setError(null)
      const data = await fetchModels()
      setModels(data.models)
      setStats(data.stats)
      setBenchmarkJudge(data.benchmarkJudge)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load models')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadModels()
  }, [loadModels])

  // Check if API key is configured
  useEffect(() => {
    if (!getApiKey()) setShowApiKeyPrompt(true)
  }, [])

  const findModel = useCallback(
    (id: string) => models.find((m) => m.id === id),
    [models]
  )

  const handleToggleModel = useCallback(
    async (modelId: string, enabled: boolean) => {
      const model = findModel(modelId)
      if (!model) return
      // Optimistic update
      setModels((prev) =>
        prev.map((m) => (m.id === modelId ? { ...m, enabled } : m))
      )
      setStats((prev) => {
        const updated = models.map((m) =>
          m.id === modelId ? { ...m, enabled } : m
        )
        return {
          ...prev,
          taggedAndEnabled: updated.filter((m) => m.enabled && m.status === 'tagged').length,
          disabled: updated.filter((m) => !m.enabled).length,
        }
      })
      try {
        await toggleModel(model.name, model.provider, enabled)
      } catch {
        // Revert on failure
        loadModels()
      }
    },
    [findModel, models, loadModels]
  )

  const handleDiscover = useCallback(async () => {
    setActiveOperation({
      type: 'discover',
      status: 'running',
      totalModels: 0,
      completedModels: 0,
      currentModel: 'Scanning providers...',
      startedAt: new Date().toISOString(),
    })
    try {
      await discoverModels()
      await loadModels()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Discovery failed')
    } finally {
      setActiveOperation(null)
    }
  }, [loadModels])

  const startPolling = useCallback(
    (opType: 'tag' | 'benchmark', fetcher: typeof fetchTagStatus) => {
      if (pollRef.current) clearInterval(pollRef.current)
      pollRef.current = setInterval(async () => {
        try {
          const s = await fetcher()
          if (s.status === 'running') {
            setActiveOperation({
              type: opType,
              status: 'running',
              totalModels: s.total,
              completedModels: s.completed,
              currentModel: s.currentModel,
              startedAt: s.startedAt,
              totalTasks: s.totalTasks ?? 0,
              completedTasks: s.completedTasks ?? 0,
              currentTask: s.currentTask ?? '',
              logEntries: s.logEntries ?? [],
            })
          } else {
            setActiveOperation({
              type: opType,
              status: 'completed',
              totalModels: s.total,
              completedModels: s.completed,
              currentModel: '',
              startedAt: s.startedAt,
            })
            if (pollRef.current) clearInterval(pollRef.current)
            pollRef.current = null
            loadModels()
            setTimeout(() => setActiveOperation(null), 2000)
          }
        } catch {
          if (pollRef.current) clearInterval(pollRef.current)
          pollRef.current = null
        }
      }, 1500)
    },
    [loadModels]
  )

  // Clean up polling on unmount
  useEffect(() => {
    return () => {
      if (pollRef.current) clearInterval(pollRef.current)
    }
  }, [])

  // Resume polling if an operation is already running (e.g. user navigated away and back)
  useEffect(() => {
    if (loading) return
    const checkRunning = async () => {
      for (const [opType, fetcher] of [
        ['tag', fetchTagStatus],
        ['benchmark', fetchBenchmarkStatus],
      ] as const) {
        try {
          const s = await fetcher()
          if (s.status === 'running') {
            setActiveOperation({
              type: opType,
              status: 'running',
              totalModels: s.total,
              completedModels: s.completed,
              currentModel: s.currentModel,
              startedAt: s.startedAt,
            })
            startPolling(opType, fetcher)
            return // only one operation at a time
          }
        } catch { /* ignore */ }
      }
    }
    checkRunning()
  }, [loading, startPolling])

  const handleTagSelected = useCallback(
    async (modelIds: string[]) => {
      const targets = modelIds
        .map((id) => findModel(id))
        .filter((m): m is Model => m !== undefined)
        .map((m) => ({ name: m.name, source: m.provider }))
      if (targets.length === 0) return

      setActiveOperation({
        type: 'tag',
        status: 'running',
        totalModels: targets.length,
        completedModels: 0,
        currentModel: targets[0]?.name ?? '',
        startedAt: new Date().toISOString(),
      })
      try {
        await tagModels(targets)
        startPolling('tag', fetchTagStatus)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Tagging failed')
        setActiveOperation(null)
      }
    },
    [findModel, startPolling]
  )

  const handleBenchmarkSelected = useCallback(
    async (modelIds: string[]) => {
      const targets = modelIds
        .map((id) => findModel(id))
        .filter((m): m is Model => m !== undefined)
        .map((m) => ({ name: m.name, source: m.provider }))
      if (targets.length === 0) return

      setActiveOperation({
        type: 'benchmark',
        status: 'running',
        totalModels: targets.length,
        completedModels: 0,
        currentModel: targets[0]?.name ?? '',
        startedAt: new Date().toISOString(),
      })
      try {
        await benchmarkModels(targets)
        startPolling('benchmark', fetchBenchmarkStatus)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Benchmarking failed')
        setActiveOperation(null)
      }
    },
    [findModel, startPolling]
  )

  const handleToggleReasoning = useCallback(
    async (modelId: string, hasReasoning: boolean) => {
      const model = findModel(modelId)
      if (!model) return
      setModels((prev) =>
        prev.map((m) => (m.id === modelId ? { ...m, hasReasoning } : m))
      )
      try {
        await updateModelReasoning(model.name, model.provider, hasReasoning)
      } catch {
        loadModels()
      }
    },
    [findModel, loadModels]
  )

  const handleRetagModel = useCallback(
    async (modelId: string) => {
      const model = findModel(modelId)
      if (!model) return
      try {
        await tagModels([{ name: model.name, source: model.provider }])
        setTimeout(loadModels, 2000)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Re-tag failed')
      }
    },
    [findModel, loadModels]
  )

  const handleRebenchmarkModel = useCallback(
    async (modelId: string) => {
      const model = findModel(modelId)
      if (!model) return
      try {
        await benchmarkModels([{ name: model.name, source: model.provider }])
        setTimeout(loadModels, 2000)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Re-benchmark failed')
      }
    },
    [findModel, loadModels]
  )

  const handleResetModel = useCallback(
    async (modelId: string) => {
      const model = findModel(modelId)
      if (!model) return
      try {
        await resetModel(model.name, model.provider)
        loadModels()
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Reset failed')
      }
    },
    [findModel, loadModels]
  )

  const handleDeleteModel = useCallback(
    async (modelId: string) => {
      const model = findModel(modelId)
      if (!model) return
      try {
        await deleteModel(model.name, model.provider)
        loadModels()
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Delete failed')
      }
    },
    [findModel, loadModels]
  )

  const handleExpandModel = useCallback(
    async (modelId: string) => {
      const model = findModel(modelId)
      if (!model) return
      setExpandedBenchmarks(null)
      try {
        const results = await fetchModelBenchmarkResults(model.name, model.provider)
        setExpandedBenchmarks(results)
      } catch {
        // Non-critical — detail panel will show "no results"
        setExpandedBenchmarks({ modelId, results: [] })
      }
    },
    [findModel]
  )

  const handleCollapseModel = useCallback(() => {
    setExpandedBenchmarks(null)
  }, [])

  const handleSaveApiKey = () => {
    setApiKey(apiKeyInput.trim())
    setShowApiKeyPrompt(false)
    setApiKeyInput('')
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-6 h-6 text-indigo-500 animate-spin" strokeWidth={2} />
      </div>
    )
  }

  return (
    <>
      {/* API Key prompt */}
      {showApiKeyPrompt && (
        <div className="mx-5 mt-5 lg:mx-6 lg:mt-6 rounded-xl border border-amber-200 bg-amber-50/60 dark:border-amber-500/20 dark:bg-amber-950/30 px-4 py-3">
          <div className="flex items-start gap-3">
            <Key className="w-4 h-4 text-amber-600 dark:text-amber-400 mt-0.5 shrink-0" strokeWidth={1.75} />
            <div className="flex-1">
              <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
                Admin API key required
              </p>
              <p className="text-xs text-amber-600/80 dark:text-amber-400/60 mt-0.5 mb-2">
                Set your ADMIN_API_KEY to enable model operations (discover, tag, benchmark, enable/disable).
              </p>
              <div className="flex gap-2">
                <input
                  type="password"
                  placeholder="Enter API key..."
                  value={apiKeyInput}
                  onChange={(e) => setApiKeyInput(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleSaveApiKey()}
                  className="flex-1 max-w-xs px-3 py-1.5 text-sm rounded-lg border border-amber-200 bg-white dark:border-amber-600/30 dark:bg-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-amber-500/20"
                />
                <button
                  onClick={handleSaveApiKey}
                  className="px-3 py-1.5 text-sm font-medium rounded-lg bg-amber-600 text-white hover:bg-amber-700 dark:bg-amber-500 dark:hover:bg-amber-600 transition-colors"
                >
                  Save
                </button>
                <button
                  onClick={() => setShowApiKeyPrompt(false)}
                  className="px-3 py-1.5 text-sm font-medium rounded-lg text-amber-700 hover:bg-amber-100 dark:text-amber-400 dark:hover:bg-amber-900/30 transition-colors"
                >
                  Skip
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Error banner */}
      {error && (
        <div className="mx-5 mt-5 lg:mx-6 lg:mt-6 rounded-xl border border-red-200 bg-red-50/60 dark:border-red-500/20 dark:bg-red-950/30 px-4 py-3">
          <div className="flex items-center gap-2">
            <AlertCircle className="w-4 h-4 text-red-500 dark:text-red-400 shrink-0" strokeWidth={1.75} />
            <p className="text-sm text-red-700 dark:text-red-300">{error}</p>
            <button
              onClick={() => setError(null)}
              className="ml-auto text-xs text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
            >
              Dismiss
            </button>
          </div>
        </div>
      )}

      <ModelDashboard
        stats={stats}
        models={models}
        activeOperation={activeOperation}
        expandedModelBenchmarks={expandedBenchmarks}
        benchmarkJudge={benchmarkJudge}
        onToggleModel={handleToggleModel}
        onTagSelected={handleTagSelected}
        onBenchmarkSelected={handleBenchmarkSelected}
        onDiscoverModels={handleDiscover}
        onRefresh={loadModels}
        onToggleReasoning={handleToggleReasoning}
        onRetagModel={handleRetagModel}
        onRebenchmarkModel={handleRebenchmarkModel}
        onResetModel={handleResetModel}
        onDeleteModel={handleDeleteModel}
        onExpandModel={handleExpandModel}
        onCollapseModel={handleCollapseModel}
      />
    </>
  )
}
