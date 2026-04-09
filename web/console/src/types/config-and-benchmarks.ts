/** How a benchmark task is scored */
export type ScoringMethod = 'llm-judge' | 'deterministic' | 'format'

/** A single criterion for LLM judge scoring */
export interface JudgeCriterion {
  name: string
  description: string
}

/** A defined benchmark evaluation task */
export interface BenchmarkTask {
  id: string
  name: string
  category: string
  prompt: string
  scoringMethod: ScoringMethod
  /** Used by llm-judge: structured criteria the judge evaluates against */
  judgeCriteria: JudgeCriterion[] | null
  /** Used by deterministic: the exact expected answer */
  expectedAnswer: string | null
  difficulty: number
  enabled: boolean
  builtin: boolean
  resultCount: number
}

/** A scored result for a model on a specific task */
export interface BenchmarkResult {
  id: string
  taskId: string
  modelName: string
  score: number
  latencyMs: number
  timestamp: string
  response?: string
}

/** Aggregated scores per category for a single model */
export interface ScoreMatrixEntry {
  modelName: string
  scores: Record<string, number>
}

/** The three tagging prompt templates */
export interface TaggingPrompts {
  systemPrompt: string
  userPromptWithSystemRole: string
  userPromptWithoutSystemRole: string
}

/** Tagging result for a single model: strengths and weaknesses */
export interface ModelTagResult {
  modelName: string
  strengths: string[]
  weaknesses: string[]
}

/** Preview result from testing a tagging prompt */
export interface TagPreviewResult {
  modelName: string
  strengths: string[]
  weaknesses: string[]
}

export interface ConfigBenchmarksProps {
  benchmarkTasks: BenchmarkTask[]
  benchmarkResults: BenchmarkResult[]
  scoreMatrix: ScoreMatrixEntry[]
  benchmarkJudge?: string
  taggingPrompts: TaggingPrompts
  /** Existing tag results for each model */
  modelTagResults: ModelTagResult[]

  /** Called when a new benchmark task is created */
  onCreateTask?: (task: Omit<BenchmarkTask, 'id' | 'resultCount'>) => void
  /** Called when a benchmark task is updated */
  onEditTask?: (id: string, updates: Partial<BenchmarkTask>) => void
  /** Called when a benchmark task is deleted */
  onDeleteTask?: (id: string) => void
  /** Called when a benchmark task is enabled or disabled */
  onToggleTask?: (id: string, enabled: boolean) => void
  /** Called when a benchmark run is triggered with selected model names */
  onRunBenchmarks?: (modelNames: string[]) => void

  /** Called when tagging prompts are saved */
  onSavePrompts?: (prompts: TaggingPrompts) => void
  /** Called when the user tests a tagging prompt against a model */
  onTestPrompt?: (modelName: string) => void
  /** Preview result returned from a prompt test */
  tagPreviewResult?: TagPreviewResult | null
}
