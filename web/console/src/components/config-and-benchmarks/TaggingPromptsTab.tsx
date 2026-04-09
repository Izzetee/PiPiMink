import { useState } from 'react'
import type {
  TaggingPrompts,
  TagPreviewResult,
  ModelTagResult,
} from '@/types/config-and-benchmarks'
import {
  Save,
  FlaskConical,
  Loader2,
  ChevronDown,
  ChevronRight,
  ThumbsUp,
  ThumbsDown,
} from 'lucide-react'

interface TaggingPromptsTabProps {
  prompts: TaggingPrompts
  modelTagResults: ModelTagResult[]
  onSavePrompts?: (prompts: TaggingPrompts) => void
  onTestPrompt?: (modelName: string) => void
  tagPreviewResult?: TagPreviewResult | null
}

const promptFields: {
  key: keyof TaggingPrompts
  label: string
  description: string
}[] = [
  {
    key: 'systemPrompt',
    label: 'System Prompt',
    description: 'The system-level instruction for the tagging evaluator.',
  },
  {
    key: 'userPromptWithSystemRole',
    label: 'User Prompt (with system role)',
    description:
      'Used when the model supports system role messages. Use {{model_name}} and {{responses}} placeholders.',
  },
  {
    key: 'userPromptWithoutSystemRole',
    label: 'User Prompt (without system role)',
    description:
      'Fallback prompt that includes the system instruction inline. Use {{model_name}} and {{responses}} placeholders.',
  },
]

export function TaggingPromptsTab({
  prompts,
  modelTagResults,
  onSavePrompts,
  onTestPrompt,
  tagPreviewResult,
}: TaggingPromptsTabProps) {
  const [draft, setDraft] = useState<TaggingPrompts>({ ...prompts })
  const [selectedModel, setSelectedModel] = useState(
    modelTagResults[0]?.modelName ?? ''
  )
  const [testing, setTesting] = useState(false)
  const [expandedModel, setExpandedModel] = useState<string | null>(null)

  const hasChanges =
    draft.systemPrompt !== prompts.systemPrompt ||
    draft.userPromptWithSystemRole !== prompts.userPromptWithSystemRole ||
    draft.userPromptWithoutSystemRole !== prompts.userPromptWithoutSystemRole

  function handleTest() {
    setTesting(true)
    onTestPrompt?.(selectedModel)
    setTimeout(() => setTesting(false), 1500)
  }

  return (
    <div className="space-y-5">
      {/* Model Tag Overview */}
      <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
        <div className="px-4 py-3 border-b border-slate-100 dark:border-slate-700/50">
          <h4 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
            Tagged Models
          </h4>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            Current capability tags assigned to each model. Click to expand.
          </p>
        </div>
        <div className="divide-y divide-slate-100 dark:divide-slate-700/50">
          {modelTagResults.map((result) => {
            const isExpanded = expandedModel === result.modelName
            return (
              <div key={result.modelName}>
                <button
                  onClick={() =>
                    setExpandedModel(isExpanded ? null : result.modelName)
                  }
                  className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-slate-50/80 dark:hover:bg-slate-700/20 transition-colors"
                >
                  <span className="flex-shrink-0 text-slate-400 dark:text-slate-500">
                    {isExpanded ? (
                      <ChevronDown className="w-4 h-4" strokeWidth={2} />
                    ) : (
                      <ChevronRight className="w-4 h-4" strokeWidth={2} />
                    )}
                  </span>
                  <span className="flex-1 text-sm font-mono font-medium text-slate-800 dark:text-slate-200 truncate">
                    {result.modelName}
                  </span>
                  <span className="flex items-center gap-3 flex-shrink-0">
                    <span className="inline-flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
                      <ThumbsUp className="w-3 h-3" strokeWidth={2} />
                      {result.strengths.length}
                    </span>
                    <span className="inline-flex items-center gap-1 text-xs text-red-500 dark:text-red-400">
                      <ThumbsDown className="w-3 h-3" strokeWidth={2} />
                      {result.weaknesses.length}
                    </span>
                  </span>
                </button>

                {isExpanded && (
                  <div className="px-4 pb-4 pl-11">
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                      {/* Strengths */}
                      <div>
                        <div className="flex items-center gap-1.5 mb-2">
                          <ThumbsUp className="w-3.5 h-3.5 text-emerald-500" strokeWidth={2} />
                          <span className="text-xs font-semibold text-emerald-700 dark:text-emerald-300">
                            Strengths
                          </span>
                        </div>
                        <div className="flex flex-wrap gap-1.5">
                          {result.strengths.map((tag) => (
                            <span
                              key={tag}
                              className="px-2 py-1 text-[11px] font-medium rounded-md bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300 border border-emerald-100 dark:border-emerald-800/40"
                            >
                              {tag}
                            </span>
                          ))}
                        </div>
                      </div>

                      {/* Weaknesses */}
                      <div>
                        <div className="flex items-center gap-1.5 mb-2">
                          <ThumbsDown className="w-3.5 h-3.5 text-red-500" strokeWidth={2} />
                          <span className="text-xs font-semibold text-red-700 dark:text-red-300">
                            Weaknesses
                          </span>
                        </div>
                        <div className="flex flex-wrap gap-1.5">
                          {result.weaknesses.map((tag) => (
                            <span
                              key={tag}
                              className="px-2 py-1 text-[11px] font-medium rounded-md bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300 border border-red-100 dark:border-red-800/40"
                            >
                              {tag}
                            </span>
                          ))}
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      {/* Prompt editors */}
      <div className="space-y-4">
        {promptFields.map((field) => (
          <div
            key={field.key}
            className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4"
          >
            <div className="flex items-start justify-between mb-2">
              <div>
                <h4 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
                  {field.label}
                </h4>
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                  {field.description}
                </p>
              </div>
            </div>
            <textarea
              value={draft[field.key]}
              onChange={(e) =>
                setDraft((prev) => ({ ...prev, [field.key]: e.target.value }))
              }
              rows={field.key === 'systemPrompt' ? 3 : 5}
              className="w-full px-3 py-2.5 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-slate-50 dark:bg-slate-900/40 text-slate-800 dark:text-slate-200 font-mono leading-relaxed focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors resize-none"
            />
          </div>
        ))}
      </div>

      {/* Save button */}
      <div className="flex items-center justify-between">
        <span className="text-xs text-slate-500 dark:text-slate-400">
          {hasChanges ? 'You have unsaved changes' : 'All changes saved'}
        </span>
        <button
          onClick={() => onSavePrompts?.(draft)}
          disabled={!hasChanges}
          className="inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-600 transition-colors shadow-sm disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Save className="w-3.5 h-3.5" strokeWidth={2} />
          Save Prompts
        </button>
      </div>

      {/* Test Preview */}
      <div className="rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
        <h4 className="text-sm font-semibold text-slate-800 dark:text-slate-200 mb-3">
          Test Preview
        </h4>
        <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
          Run the tagging prompt against a model to preview the generated capability tags.
        </p>

        <div className="flex flex-col sm:flex-row items-start sm:items-end gap-3">
          <div className="flex-1 w-full sm:w-auto">
            <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
              Model
            </label>
            <select
              value={selectedModel}
              onChange={(e) => setSelectedModel(e.target.value)}
              className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-800 dark:text-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            >
              {modelTagResults.map((m) => (
                <option key={m.modelName} value={m.modelName}>
                  {m.modelName}
                </option>
              ))}
            </select>
          </div>
          <button
            onClick={handleTest}
            disabled={testing}
            className="inline-flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-400 dark:hover:bg-slate-700 transition-colors disabled:opacity-50"
          >
            {testing ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" strokeWidth={2} />
            ) : (
              <FlaskConical className="w-3.5 h-3.5" strokeWidth={2} />
            )}
            {testing ? 'Running...' : 'Test Prompt'}
          </button>
        </div>

        {/* Preview results */}
        {tagPreviewResult && (
          <div className="mt-4 pt-4 border-t border-slate-100 dark:border-slate-700/50">
            <div className="flex items-center gap-2 mb-3">
              <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
                Results for
              </span>
              <span className="text-xs font-mono font-medium text-slate-700 dark:text-slate-300">
                {tagPreviewResult.modelName}
              </span>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <div className="flex items-center gap-1.5 mb-2">
                  <ThumbsUp className="w-3.5 h-3.5 text-emerald-500" strokeWidth={2} />
                  <span className="text-xs font-semibold text-emerald-700 dark:text-emerald-300">
                    Strengths
                  </span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {tagPreviewResult.strengths.map((tag) => (
                    <span
                      key={tag}
                      className="px-2 py-1 text-[11px] font-medium rounded-md bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300 border border-emerald-100 dark:border-emerald-800/40"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              </div>
              <div>
                <div className="flex items-center gap-1.5 mb-2">
                  <ThumbsDown className="w-3.5 h-3.5 text-red-500" strokeWidth={2} />
                  <span className="text-xs font-semibold text-red-700 dark:text-red-300">
                    Weaknesses
                  </span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {tagPreviewResult.weaknesses.map((tag) => (
                    <span
                      key={tag}
                      className="px-2 py-1 text-[11px] font-medium rounded-md bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300 border border-red-100 dark:border-red-800/40"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
