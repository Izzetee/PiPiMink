import { apiGet, apiPost, apiPatch, apiDelete } from './client'
import type {
  Model,
  DashboardStats,
  BenchmarkCategory,
  BenchmarkTaskResult,
  ExpandedModelBenchmarks,
  ModelStatus,
} from '@/types/model-dashboard'

// --- Category name mapping ---
// Backend uses hyphenated names; frontend uses short names.

const backendToFrontendCategory: Record<string, BenchmarkCategory> = {
  'coding': 'coding',
  'creative-writing': 'creative',
  'factual-qa': 'factual',
  'instruction-following': 'instruction',
  'reasoning': 'reasoning',
  'summarization': 'summarization',
}

function mapScores(
  raw: Record<string, number> | null | undefined
): Partial<Record<BenchmarkCategory, number>> {
  if (!raw) return {}
  const mapped: Partial<Record<BenchmarkCategory, number>> = {}
  for (const [key, value] of Object.entries(raw)) {
    const frontendKey = backendToFrontendCategory[key]
    if (frontendKey) mapped[frontendKey] = value
  }
  return mapped
}

function mapCategory(backendCategory: string): string {
  return backendToFrontendCategory[backendCategory] ?? backendCategory
}

// --- Backend response types ---

interface BackendModel {
  name: string
  source: string
  enabled: boolean
  tagged: boolean
  hasReasoning: boolean
  updatedAt: string
  benchmarkScores: Record<string, number> | null
  avgLatencyMs: number | null
  tags: { strengths: string[]; weaknesses: string[] }
  taggedBy?: string
}

interface ModelsResponse {
  models: BackendModel[]
  count: number
  benchmarkJudge?: string
}

// --- Transform ---

function deriveStatus(m: BackendModel): ModelStatus {
  if (m.tagged) return 'tagged'
  return 'discovered'
}

function transformModel(m: BackendModel): Model {
  return {
    id: m.name,
    name: m.name,
    provider: m.source,
    status: deriveStatus(m),
    enabled: m.enabled,
    hasReasoning: m.hasReasoning,
    tags: {
      strengths: m.tags?.strengths ?? [],
      weaknesses: m.tags?.weaknesses ?? [],
    },
    benchmarkScores: mapScores(m.benchmarkScores),
    avgResponseTime: m.avgLatencyMs !== null ? m.avgLatencyMs / 1000 : null,
    updatedAt: m.updatedAt,
    taggedBy: m.taggedBy,
  }
}

function computeStats(models: Model[]): DashboardStats {
  return {
    total: models.length,
    enabled: models.filter((m) => m.enabled).length,
    taggedAndEnabled: models.filter((m) => m.enabled && m.status === 'tagged').length,
    disabled: models.filter((m) => !m.enabled).length,
    discovered: models.filter((m) => m.status === 'discovered').length,
    benchmarked: models.filter((m) => Object.keys(m.benchmarkScores).length > 0).length,
  }
}

// --- Public API ---

export async function fetchModels(): Promise<{ models: Model[]; stats: DashboardStats; benchmarkJudge?: string }> {
  const data = await apiGet<ModelsResponse>('/models')
  const models = data.models.map(transformModel)
  return { models, stats: computeStats(models), benchmarkJudge: data.benchmarkJudge }
}

export async function toggleModel(
  name: string,
  source: string,
  enabled: boolean
): Promise<void> {
  await apiPatch(`/models/${encodeURIComponent(name)}/enable`, { source, enabled })
}

export async function discoverModels(): Promise<{ providers: number; discovered: number }> {
  return apiPost('/models/discover')
}

export async function tagModels(
  targets: { name: string; source: string }[]
): Promise<void> {
  await apiPost('/models/tag', { models: targets })
}

export interface BackendOperationStatus {
  status: 'idle' | 'running' | 'completed' | 'failed'
  total: number
  completed: number
  currentModel: string
  startedAt: string
  failedModels?: string[]
  // Task-level (benchmark only)
  totalTasks?: number
  completedTasks?: number
  currentTask?: string
  logEntries?: { model: string; task: string; category: string; score: number; ok: boolean }[]
}

export async function fetchTagStatus(): Promise<BackendOperationStatus> {
  return apiGet<BackendOperationStatus>('/models/tag/status')
}

export async function fetchBenchmarkStatus(): Promise<BackendOperationStatus> {
  return apiGet<BackendOperationStatus>('/models/benchmark/status')
}

export async function benchmarkModels(
  targets: { name: string; source: string }[]
): Promise<void> {
  await apiPost('/models/benchmark', { models: targets })
}

export async function resetModel(name: string, source: string): Promise<void> {
  await apiPost(`/models/${encodeURIComponent(name)}/reset`, { source })
}

export async function deleteModel(name: string, source: string): Promise<void> {
  await apiDelete(`/models/${encodeURIComponent(name)}`, { source })
}

export async function updateModelReasoning(
  name: string,
  source: string,
  hasReasoning: boolean
): Promise<void> {
  await apiPost('/models/reasoning/update', {
    model: name,
    source,
    has_reasoning: hasReasoning,
  })
}

interface BackendBenchmarkResult {
  taskId: string
  category: string
  score: number
  latencyMs: number
  scoredAt: string
  judgeModel?: string
  response?: string
}

interface BenchmarkResultsResponse {
  model: string
  source: string
  results: BackendBenchmarkResult[]
  benchmarkJudge?: string
}

export async function fetchModelBenchmarkResults(
  name: string,
  source: string
): Promise<ExpandedModelBenchmarks> {
  const data = await apiGet<BenchmarkResultsResponse>(
    `/models/${encodeURIComponent(name)}/benchmark-results?source=${encodeURIComponent(source)}`
  )
  const results: BenchmarkTaskResult[] = (data.results ?? []).map((r) => ({
    taskId: r.taskId,
    category: mapCategory(r.category),
    score: r.score,
    latency: r.latencyMs / 1000,
    scoredAt: r.scoredAt,
    judgeModel: r.judgeModel,
    response: r.response,
  }))
  return { modelId: name, results, benchmarkJudge: data.benchmarkJudge }
}
