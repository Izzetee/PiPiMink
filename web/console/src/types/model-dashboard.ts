export type ModelStatus = 'discovered' | 'tagged' | 'enabled'

export type BenchmarkCategory =
  | 'coding'
  | 'creative'
  | 'factual'
  | 'instruction'
  | 'reasoning'
  | 'summarization'

export type OperationType = 'tag' | 'benchmark' | 'discover'
export type OperationStatus = 'running' | 'completed' | 'failed'

export interface ModelTags {
  strengths: string[]
  weaknesses: string[]
}

export interface Model {
  id: string
  name: string
  provider: string
  status: ModelStatus
  enabled: boolean
  hasReasoning: boolean
  tags: ModelTags
  benchmarkScores: Partial<Record<BenchmarkCategory, number>>
  avgResponseTime: number | null
  updatedAt: string
  taggedBy?: string
}

export interface DashboardStats {
  total: number
  enabled: number
  taggedAndEnabled: number
  disabled: number
  discovered: number
  benchmarked: number
}

export interface BenchmarkTaskResult {
  taskId: string
  category: string
  score: number
  latency: number
  scoredAt: string
  judgeModel?: string
  response?: string
}

export interface ExpandedModelBenchmarks {
  modelId: string
  results: BenchmarkTaskResult[]
  benchmarkJudge?: string
}

export interface LogEntry {
  model: string
  task: string
  category: string
  score: number
  ok: boolean
}

export interface Operation {
  type: OperationType
  status: OperationStatus
  totalModels: number
  completedModels: number
  currentModel: string
  startedAt: string
  // Task-level progress (benchmark only)
  totalTasks?: number
  completedTasks?: number
  currentTask?: string
  logEntries?: LogEntry[]
}

export type StatFilter =
  | 'all'
  | 'enabled'
  | 'taggedAndEnabled'
  | 'disabled'
  | 'discovered'
  | 'benchmarked'

export type DetailTab = 'overview' | 'benchmarks'

export interface ModelDashboardProps {
  stats: DashboardStats
  models: Model[]
  activeOperation: Operation | null
  expandedModelBenchmarks: ExpandedModelBenchmarks | null
  benchmarkJudge?: string

  onToggleModel?: (modelId: string, enabled: boolean) => void
  onTagSelected?: (modelIds: string[]) => void
  onBenchmarkSelected?: (modelIds: string[]) => void
  onDiscoverModels?: () => void
  onRefresh?: () => void
  onToggleReasoning?: (modelId: string, hasReasoning: boolean) => void
  onRetagModel?: (modelId: string) => void
  onRebenchmarkModel?: (modelId: string) => void
  onResetModel?: (modelId: string) => void
  onDeleteModel?: (modelId: string) => void
  onExpandModel?: (modelId: string) => void
  onCollapseModel?: () => void
}
