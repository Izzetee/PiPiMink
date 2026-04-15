import { useState } from 'react'
import type {
  BenchmarkTask,
  ScoringMethod,
  JudgeCriterion,
} from '@/types/config-and-benchmarks'
import { X, Save, Plus } from 'lucide-react'

interface TaskFormProps {
  task?: BenchmarkTask | null
  existingCategories: string[]
  onSave: (data: Partial<BenchmarkTask>) => void
  onCancel: () => void
}

const scoringMethods: { value: ScoringMethod; label: string; description: string }[] = [
  { value: 'llm-judge', label: 'LLM Judge', description: 'Scored by an LLM evaluator using structured criteria' },
  { value: 'deterministic', label: 'Deterministic', description: 'Exact match against expected answer' },
  { value: 'format', label: 'Format', description: 'Built-in format validator (builtin only)' },
]

export function TaskForm({ task, existingCategories, onSave, onCancel }: TaskFormProps) {
  const [name, setName] = useState(task?.name ?? '')
  const [category, setCategory] = useState(task?.category ?? '')
  const [prompt, setPrompt] = useState(task?.prompt ?? '')
  const [scoringMethod, setScoringMethod] = useState<ScoringMethod>(task?.scoringMethod ?? 'llm-judge')
  const [judgeCriteria, setJudgeCriteria] = useState<JudgeCriterion[]>(
    task?.judgeCriteria ?? [{ name: '', description: '' }]
  )
  const [expectedAnswer, setExpectedAnswer] = useState(task?.expectedAnswer ?? '')
  const [difficulty, setDifficulty] = useState(task?.difficulty ?? 3)
  const [enabled, setEnabled] = useState(task?.enabled ?? true)

  const isEditing = !!task

  function addCriterion() {
    setJudgeCriteria([...judgeCriteria, { name: '', description: '' }])
  }

  function removeCriterion(index: number) {
    setJudgeCriteria(judgeCriteria.filter((_, i) => i !== index))
  }

  function updateCriterion(index: number, field: keyof JudgeCriterion, value: string) {
    setJudgeCriteria(
      judgeCriteria.map((c, i) => (i === index ? { ...c, [field]: value } : c))
    )
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    onSave({
      name,
      category,
      prompt,
      scoringMethod,
      judgeCriteria:
        scoringMethod === 'llm-judge'
          ? judgeCriteria.filter((c) => c.name.trim() || c.description.trim())
          : null,
      expectedAnswer: scoringMethod === 'deterministic' ? expectedAnswer || null : null,
      difficulty,
      enabled,
      builtin: task?.builtin ?? false,
    })
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-xl border border-indigo-200 dark:border-indigo-800/50 bg-indigo-50/30 dark:bg-indigo-950/20 p-4 space-y-4"
    >
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
          {isEditing ? 'Edit Task' : 'New Benchmark Task'}
        </h3>
        <button
          type="button"
          onClick={onCancel}
          className="p-1 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:text-slate-500 dark:hover:text-slate-300 dark:hover:bg-slate-700 transition-colors"
        >
          <X className="w-4 h-4" strokeWidth={2} />
        </button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        {/* Name */}
        <div>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
            Task Name
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            placeholder="e.g. Python function generation"
            className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
          />
        </div>

        {/* Category — free text with datalist suggestions */}
        <div>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
            Category
          </label>
          <input
            type="text"
            list="category-suggestions"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            required
            placeholder="e.g. coding, reasoning, or a new category"
            className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
          />
          <datalist id="category-suggestions">
            {existingCategories.map((c) => (
              <option key={c} value={c} />
            ))}
          </datalist>
        </div>
      </div>

      {/* Prompt */}
      <div>
        <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
          Prompt
        </label>
        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          required
          rows={3}
          placeholder="The evaluation prompt sent to the model..."
          className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors resize-none font-mono"
        />
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        {/* Scoring Method */}
        <div>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
            Scoring Method
          </label>
          <select
            value={scoringMethod}
            onChange={(e) => setScoringMethod(e.target.value as ScoringMethod)}
            className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
          >
            {scoringMethods.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </select>
          <p className="mt-0.5 text-[10px] text-slate-400 dark:text-slate-500">
            {scoringMethods.find((m) => m.value === scoringMethod)?.description}
          </p>
        </div>

        {/* Difficulty */}
        <div>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
            Difficulty (1–5)
          </label>
          <div className="flex items-center gap-2">
            <input
              type="range"
              min={1}
              max={5}
              value={difficulty}
              onChange={(e) => setDifficulty(Number(e.target.value))}
              className="flex-1 accent-indigo-500"
            />
            <span className="w-6 text-center text-sm font-mono font-medium text-slate-700 dark:text-slate-300">
              {difficulty}
            </span>
          </div>
        </div>

        {/* Enabled */}
        <div>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
            Status
          </label>
          <button
            type="button"
            onClick={() => setEnabled(!enabled)}
            className={`inline-flex items-center gap-2 px-3 py-2 text-sm rounded-lg border transition-colors ${
              enabled
                ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-900/20 dark:text-emerald-300'
                : 'border-slate-200 bg-slate-50 text-slate-500 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-400'
            }`}
          >
            <span
              className={`w-2 h-2 rounded-full ${
                enabled ? 'bg-emerald-500' : 'bg-slate-400'
              }`}
            />
            {enabled ? 'Enabled' : 'Disabled'}
          </button>
        </div>
      </div>

      {/* LLM Judge: structured criteria rows */}
      {scoringMethod === 'llm-judge' && (
        <div>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-2">
            Judge Criteria
          </label>
          <div className="space-y-2">
            {judgeCriteria.map((criterion, i) => (
              <div key={i} className="flex items-start gap-2">
                <input
                  type="text"
                  value={criterion.name}
                  onChange={(e) => updateCriterion(i, 'name', e.target.value)}
                  placeholder="Criterion name"
                  className="w-40 flex-shrink-0 px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
                />
                <input
                  type="text"
                  value={criterion.description}
                  onChange={(e) => updateCriterion(i, 'description', e.target.value)}
                  placeholder="Description of what to evaluate..."
                  className="flex-1 px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
                />
                <button
                  type="button"
                  onClick={() => removeCriterion(i)}
                  disabled={judgeCriteria.length <= 1}
                  className="flex-shrink-0 p-2 rounded-lg text-slate-400 hover:text-red-500 hover:bg-red-50 dark:text-slate-500 dark:hover:text-red-400 dark:hover:bg-red-900/20 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                >
                  <X className="w-3.5 h-3.5" strokeWidth={2} />
                </button>
              </div>
            ))}
          </div>
          <button
            type="button"
            onClick={addCriterion}
            className="mt-2 inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border border-dashed border-slate-300 text-slate-500 hover:border-indigo-300 hover:text-indigo-600 dark:border-slate-600 dark:text-slate-400 dark:hover:border-indigo-700 dark:hover:text-indigo-400 transition-colors"
          >
            <Plus className="w-3 h-3" strokeWidth={2} />
            Add Criterion
          </button>
        </div>
      )}

      {/* Deterministic: expected answer */}
      {scoringMethod === 'deterministic' && (
        <div>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
            Expected Answer
          </label>
          <input
            type="text"
            value={expectedAnswer}
            onChange={(e) => setExpectedAnswer(e.target.value)}
            placeholder="The exact expected answer..."
            className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors font-mono"
          />
        </div>
      )}

      {/* Format: no extra fields needed */}

      {/* Actions */}
      <div className="flex items-center justify-end gap-2 pt-1">
        <button
          type="button"
          onClick={onCancel}
          className="px-3 py-2 text-sm font-medium rounded-lg text-slate-500 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          className="inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 transition-colors shadow-sm"
        >
          <Save className="w-3.5 h-3.5" strokeWidth={2} />
          {isEditing ? 'Update Task' : 'Create Task'}
        </button>
      </div>
    </form>
  )
}
