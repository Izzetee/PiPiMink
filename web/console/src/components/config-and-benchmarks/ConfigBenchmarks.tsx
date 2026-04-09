import { useState } from 'react'
import type { ConfigBenchmarksProps } from '@/types/config-and-benchmarks'
import { BenchmarksTab } from './BenchmarksTab'
import { TaggingPromptsTab } from './TaggingPromptsTab'
import { FlaskConical, MessageSquareCode } from 'lucide-react'

type Tab = 'benchmarks' | 'tagging-prompts'

export function ConfigBenchmarks(props: ConfigBenchmarksProps) {
  const [tab, setTab] = useState<Tab>('benchmarks')

  return (
    <div className="p-5 lg:p-6">
      {/* Tabs */}
      <div className="flex items-center gap-1 border-b border-slate-200 dark:border-slate-700 mb-5">
        <button
          onClick={() => setTab('benchmarks')}
          className={`inline-flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors -mb-px ${
            tab === 'benchmarks'
              ? 'border-indigo-500 text-indigo-700 dark:border-indigo-400 dark:text-indigo-300'
              : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
          }`}
        >
          <FlaskConical className="w-4 h-4" strokeWidth={1.75} />
          Benchmarks
        </button>
        <button
          onClick={() => setTab('tagging-prompts')}
          className={`inline-flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors -mb-px ${
            tab === 'tagging-prompts'
              ? 'border-indigo-500 text-indigo-700 dark:border-indigo-400 dark:text-indigo-300'
              : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
          }`}
        >
          <MessageSquareCode className="w-4 h-4" strokeWidth={1.75} />
          Tagging Prompts
        </button>
      </div>

      {/* Tab content */}
      {tab === 'benchmarks' ? (
        <BenchmarksTab
          tasks={props.benchmarkTasks}
          results={props.benchmarkResults}
          scoreMatrix={props.scoreMatrix}
          benchmarkJudge={props.benchmarkJudge}
          onCreateTask={props.onCreateTask}
          onEditTask={props.onEditTask}
          onDeleteTask={props.onDeleteTask}
          onToggleTask={props.onToggleTask}
          onRunBenchmarks={props.onRunBenchmarks}
        />
      ) : (
        <TaggingPromptsTab
          prompts={props.taggingPrompts}
          modelTagResults={props.modelTagResults}
          onSavePrompts={props.onSavePrompts}
          onTestPrompt={props.onTestPrompt}
          tagPreviewResult={props.tagPreviewResult}
        />
      )}
    </div>
  )
}
