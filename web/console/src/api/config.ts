import { apiGet, apiPost, apiPut, apiDelete } from './client'
import type {
  BenchmarkTask,
  BenchmarkResult,
  ScoreMatrixEntry,
  TaggingPrompts,
  ScoringMethod,
  JudgeCriterion,
} from '@/types/config-and-benchmarks'

// --- Backend response types ---

interface BackendJudgeCriterion {
  Name: string
  Description: string
}

interface BackendBenchmarkTask {
  task_id: string
  category: string
  prompt: string
  scoring_method: string
  expected_answer: string
  judge_criteria: BackendJudgeCriterion[] | null
  enabled: boolean
  is_builtin: boolean
  updated_at: string
}

interface BackendLeaderboardEntry {
  model: string
  source: string
  scores: Record<string, number>
  avg_score: number
}

interface BackendLeaderboardResponse {
  leaderboard: BackendLeaderboardEntry[] | null
  categories: string[]
  count: number
  benchmarkJudge?: string
}

interface BackendBenchmarkResult {
  taskId: string
  category: string
  score: number
  latencyMs: number
  scoredAt: string
  response?: string
}

interface BackendBenchmarkResultsResponse {
  model: string
  source: string
  results: BackendBenchmarkResult[] | null
}

interface BackendSystemPromptRow {
  key: string
  value: string
  description: string
  updated_at: string
}

// --- Transforms ---

function transformTask(t: BackendBenchmarkTask): BenchmarkTask {
  return {
    id: t.task_id,
    name: t.task_id,
    category: t.category,
    prompt: t.prompt,
    scoringMethod: t.scoring_method as ScoringMethod,
    judgeCriteria: t.judge_criteria
      ? t.judge_criteria.map((c) => ({ name: c.Name, description: c.Description }))
      : null,
    expectedAnswer: t.expected_answer || null,
    difficulty: 3,
    enabled: t.enabled,
    builtin: t.is_builtin,
    resultCount: 0,
  }
}

function toBackendTask(
  task: Omit<BenchmarkTask, 'resultCount'> & { id?: string }
): BackendBenchmarkTask {
  const criteria: BackendJudgeCriterion[] | null =
    task.judgeCriteria && task.judgeCriteria.length > 0
      ? task.judgeCriteria.map((c: JudgeCriterion) => ({
          Name: c.name,
          Description: c.description,
        }))
      : null
  return {
    task_id: task.id ?? slugify(task.name),
    category: task.category,
    prompt: task.prompt,
    scoring_method: task.scoringMethod,
    expected_answer: task.expectedAnswer ?? '',
    judge_criteria: criteria,
    enabled: task.enabled,
    is_builtin: task.builtin,
    updated_at: '',
  }
}

function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
}

// --- Public API ---

export async function fetchBenchmarkTasks(): Promise<BenchmarkTask[]> {
  const data = await apiGet<BackendBenchmarkTask[]>('/admin/benchmark-tasks')
  return data.map(transformTask)
}

export async function fetchLeaderboard(): Promise<{
  scoreMatrix: ScoreMatrixEntry[]
  modelSources: Map<string, string>
  benchmarkJudge?: string
}> {
  const data = await apiGet<BackendLeaderboardResponse>('/benchmarks/leaderboard')
  const entries = data.leaderboard ?? []
  const scoreMatrix: ScoreMatrixEntry[] = entries.map((e) => ({
    modelName: e.model,
    scores: e.scores,
  }))
  const modelSources = new Map<string, string>()
  for (const e of entries) {
    modelSources.set(e.model, e.source)
  }
  return { scoreMatrix, modelSources, benchmarkJudge: data.benchmarkJudge }
}

export async function fetchAllBenchmarkResults(
  modelSources: Map<string, string>
): Promise<BenchmarkResult[]> {
  const entries = [...modelSources.entries()]
  if (entries.length === 0) return []

  const allResults = await Promise.all(
    entries.map(async ([model, source]) => {
      const data = await apiGet<BackendBenchmarkResultsResponse>(
        `/models/${encodeURIComponent(model)}/benchmark-results?source=${encodeURIComponent(source)}`
      )
      return (data.results ?? []).map(
        (r): BenchmarkResult => ({
          id: `${model}:${r.taskId}`,
          taskId: r.taskId,
          modelName: model,
          score: r.score,
          latencyMs: r.latencyMs,
          timestamp: r.scoredAt,
          response: r.response,
        })
      )
    })
  )
  return allResults.flat()
}

export async function upsertBenchmarkTask(
  task: Omit<BenchmarkTask, 'resultCount'>
): Promise<void> {
  await apiPost('/admin/benchmark-tasks', toBackendTask(task))
}

export async function deleteBenchmarkTask(id: string): Promise<void> {
  await apiDelete(`/admin/benchmark-tasks/${encodeURIComponent(id)}`)
}

export async function fetchTaggingPrompts(): Promise<TaggingPrompts> {
  const data = await apiGet<Record<string, BackendSystemPromptRow>>(
    '/admin/system-prompts'
  )
  return {
    systemPrompt: data['tagging_system']?.value ?? '',
    userPromptWithSystemRole: data['tagging_user']?.value ?? '',
    userPromptWithoutSystemRole: data['tagging_user_nosys']?.value ?? '',
  }
}

export async function saveTaggingPrompts(prompts: TaggingPrompts): Promise<void> {
  await Promise.all([
    apiPut('/admin/system-prompts/tagging_system', {
      value: prompts.systemPrompt,
      description: 'System message sent to every model during capability tagging',
    }),
    apiPut('/admin/system-prompts/tagging_user', {
      value: prompts.userPromptWithSystemRole,
      description: 'User message sent to every model during capability tagging',
    }),
    apiPut('/admin/system-prompts/tagging_user_nosys', {
      value: prompts.userPromptWithoutSystemRole,
      description:
        'Combined system+user message used for models that do not support system messages',
    }),
  ])
}

export async function runBenchmarks(
  models: { name: string; source: string }[]
): Promise<void> {
  await apiPost('/models/benchmark', { models })
}

export async function testTaggingPrompt(
  modelName: string,
  source: string
): Promise<void> {
  await apiPost('/models/tag', { models: [{ name: modelName, source }] })
}
