import { useEffect, useState, useCallback, useRef } from 'react'
import { ConfigBenchmarks } from '@/components/config-and-benchmarks'
import {
  fetchBenchmarkTasks,
  fetchLeaderboard,
  fetchAllBenchmarkResults,
  upsertBenchmarkTask,
  deleteBenchmarkTask,
  fetchTaggingPrompts,
  saveTaggingPrompts,
  runBenchmarks,
  testTaggingPrompt,
  fetchModels,
} from '@/api'
import type {
  BenchmarkTask,
  BenchmarkResult,
  ScoreMatrixEntry,
  TaggingPrompts,
  ModelTagResult,
  TagPreviewResult,
} from '@/types/config-and-benchmarks'
import { Loader2, AlertCircle, Key } from 'lucide-react'
import { getApiKey, setApiKey } from '@/api'

function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
}

export function ConfigPage() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [benchmarkTasks, setBenchmarkTasks] = useState<BenchmarkTask[]>([])
  const [benchmarkResults, setBenchmarkResults] = useState<BenchmarkResult[]>([])
  const [scoreMatrix, setScoreMatrix] = useState<ScoreMatrixEntry[]>([])
  const [taggingPrompts, setTaggingPrompts] = useState<TaggingPrompts>({
    systemPrompt: '',
    userPromptWithSystemRole: '',
    userPromptWithoutSystemRole: '',
  })
  const [modelTagResults, setModelTagResults] = useState<ModelTagResult[]>([])
  const [tagPreviewResult, setTagPreviewResult] = useState<TagPreviewResult | null>(null)
  const [benchmarkJudge, setBenchmarkJudge] = useState<string | undefined>()
  const [apiKeyInput, setApiKeyInput] = useState('')
  const [showApiKeyPrompt, setShowApiKeyPrompt] = useState(false)

  // modelSources: modelName -> source for lookup
  const modelSourcesRef = useRef<Map<string, string>>(new Map())

  const loadAll = useCallback(async () => {
    try {
      setError(null)

      const [tasks, leaderboard, prompts, modelsData] = await Promise.all([
        fetchBenchmarkTasks(),
        fetchLeaderboard(),
        fetchTaggingPrompts(),
        fetchModels(),
      ])

      // Build model sources from models data
      const sources = new Map<string, string>()
      for (const m of modelsData.models) {
        sources.set(m.name, m.provider)
      }
      // Also merge leaderboard sources (may have models not in current model list)
      for (const [name, source] of leaderboard.modelSources) {
        if (!sources.has(name)) sources.set(name, source)
      }
      modelSourcesRef.current = sources

      // Fetch all benchmark results for models in the leaderboard
      const results = await fetchAllBenchmarkResults(leaderboard.modelSources)

      // Enrich tasks with resultCount
      const resultCountMap = new Map<string, number>()
      for (const r of results) {
        resultCountMap.set(r.taskId, (resultCountMap.get(r.taskId) ?? 0) + 1)
      }
      const enrichedTasks = tasks.map((t) => ({
        ...t,
        resultCount: resultCountMap.get(t.id) ?? 0,
      }))

      // Build model tag results from tagged models
      const tagResults: ModelTagResult[] = modelsData.models
        .filter((m) => m.tags.strengths.length > 0 || m.tags.weaknesses.length > 0)
        .map((m) => ({
          modelName: m.name,
          strengths: m.tags.strengths,
          weaknesses: m.tags.weaknesses,
        }))

      setBenchmarkTasks(enrichedTasks)
      setBenchmarkResults(results)
      setScoreMatrix(leaderboard.scoreMatrix)
      setBenchmarkJudge(leaderboard.benchmarkJudge)
      setTaggingPrompts(prompts)
      setModelTagResults(tagResults)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadAll()
  }, [loadAll])

  useEffect(() => {
    if (!getApiKey()) setShowApiKeyPrompt(true)
  }, [])

  const handleCreateTask = useCallback(
    async (data: Omit<BenchmarkTask, 'id' | 'resultCount'>) => {
      try {
        const id = slugify(data.name)
        await upsertBenchmarkTask({ ...data, id })
        await loadAll()
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to create task')
      }
    },
    [loadAll]
  )

  const handleEditTask = useCallback(
    async (id: string, updates: Partial<BenchmarkTask>) => {
      try {
        const existing = benchmarkTasks.find((t) => t.id === id)
        if (!existing) return
        const merged = { ...existing, ...updates }
        await upsertBenchmarkTask(merged)
        await loadAll()
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update task')
      }
    },
    [benchmarkTasks, loadAll]
  )

  const handleDeleteTask = useCallback(
    async (id: string) => {
      try {
        await deleteBenchmarkTask(id)
        await loadAll()
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to delete task')
      }
    },
    [loadAll]
  )

  const handleToggleTask = useCallback(
    async (id: string, enabled: boolean) => {
      // Optimistic update
      setBenchmarkTasks((prev) =>
        prev.map((t) => (t.id === id ? { ...t, enabled } : t))
      )
      try {
        const existing = benchmarkTasks.find((t) => t.id === id)
        if (!existing) return
        await upsertBenchmarkTask({ ...existing, enabled })
      } catch {
        loadAll()
      }
    },
    [benchmarkTasks, loadAll]
  )

  const handleRunBenchmarks = useCallback(
    async (modelNames: string[]) => {
      try {
        const targets = modelNames
          .map((name) => {
            const source = modelSourcesRef.current.get(name)
            return source ? { name, source } : null
          })
          .filter((t): t is { name: string; source: string } => t !== null)
        if (targets.length === 0) return
        await runBenchmarks(targets)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to start benchmarks')
      }
    },
    []
  )

  const handleSavePrompts = useCallback(
    async (prompts: TaggingPrompts) => {
      try {
        await saveTaggingPrompts(prompts)
        setTaggingPrompts(prompts)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to save prompts')
      }
    },
    []
  )

  const handleTestPrompt = useCallback(
    async (modelName: string) => {
      try {
        const source = modelSourcesRef.current.get(modelName)
        if (!source) return
        await testTaggingPrompt(modelName, source)
        // Re-fetch models to get updated tags
        const modelsData = await fetchModels()
        const model = modelsData.models.find((m) => m.name === modelName)
        if (model) {
          setTagPreviewResult({
            modelName: model.name,
            strengths: model.tags.strengths,
            weaknesses: model.tags.weaknesses,
          })
          // Also update the tag results list
          setModelTagResults((prev) => {
            const updated = prev.filter((r) => r.modelName !== modelName)
            if (model.tags.strengths.length > 0 || model.tags.weaknesses.length > 0) {
              updated.push({
                modelName: model.name,
                strengths: model.tags.strengths,
                weaknesses: model.tags.weaknesses,
              })
            }
            return updated
          })
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to test prompt')
      }
    },
    []
  )

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
                Set your ADMIN_API_KEY to manage benchmarks and prompts.
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

      <ConfigBenchmarks
        benchmarkTasks={benchmarkTasks}
        benchmarkResults={benchmarkResults}
        scoreMatrix={scoreMatrix}
        benchmarkJudge={benchmarkJudge}
        taggingPrompts={taggingPrompts}
        modelTagResults={modelTagResults}
        onCreateTask={handleCreateTask}
        onEditTask={handleEditTask}
        onDeleteTask={handleDeleteTask}
        onToggleTask={handleToggleTask}
        onRunBenchmarks={handleRunBenchmarks}
        onSavePrompts={handleSavePrompts}
        onTestPrompt={handleTestPrompt}
        tagPreviewResult={tagPreviewResult}
      />
    </>
  )
}
