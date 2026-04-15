import { useEffect, useRef, useState } from 'react'
import type { LogEntry } from '@/types/model-dashboard'
import { Terminal, Minus, Maximize2 } from 'lucide-react'

interface BenchmarkLogProps {
  logEntries: LogEntry[]
}

export function BenchmarkLog({ logEntries }: BenchmarkLogProps) {
  const [minimized, setMinimized] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (scrollRef.current && !minimized) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [logEntries, minimized])

  if (logEntries.length === 0) return null

  // Minimized pill
  if (minimized) {
    return (
      <button
        onClick={() => setMinimized(false)}
        className="fixed bottom-4 right-4 z-50 flex items-center gap-2 px-3 py-1.5 rounded-full bg-slate-900/90 dark:bg-slate-950/90 text-slate-300 text-xs font-mono shadow-lg backdrop-blur-sm border border-slate-700/50 hover:bg-slate-800/90 dark:hover:bg-slate-900/90 transition-colors"
      >
        <Terminal className="w-3 h-3" strokeWidth={2} />
        <span>{logEntries.length} tasks completed</span>
        <Maximize2 className="w-3 h-3 ml-1 opacity-50" strokeWidth={2} />
      </button>
    )
  }

  return (
    <div className="fixed bottom-4 right-4 z-50 w-80 rounded-lg shadow-xl border border-slate-700/50 overflow-hidden backdrop-blur-sm">
      {/* Title bar */}
      <div className="flex items-center justify-between px-3 py-1.5 bg-slate-800/95 dark:bg-slate-900/95 border-b border-slate-700/50">
        <div className="flex items-center gap-2">
          <Terminal className="w-3 h-3 text-slate-400" strokeWidth={2} />
          <span className="text-[11px] font-medium text-slate-300">
            Benchmark Log
          </span>
        </div>
        <button
          onClick={() => setMinimized(true)}
          className="p-0.5 rounded text-slate-500 hover:text-slate-300 hover:bg-slate-700/50 transition-colors"
        >
          <Minus className="w-3 h-3" strokeWidth={2} />
        </button>
      </div>

      {/* Log entries */}
      <div
        ref={scrollRef}
        className="max-h-48 overflow-y-auto bg-slate-900/95 dark:bg-slate-950/95 px-3 py-2 space-y-px"
      >
        {logEntries.map((entry, i) => (
          <div
            key={i}
            className="flex items-center gap-1.5 text-[11px] font-mono leading-relaxed"
          >
            <span className={entry.ok ? 'text-emerald-400' : 'text-red-400'}>
              {entry.ok ? '\u2713' : '\u2717'}
            </span>
            <span className="text-slate-400 truncate max-w-[7rem]" title={entry.model}>
              {entry.model}
            </span>
            <span className="text-slate-600">&rsaquo;</span>
            <span className="text-slate-300 truncate flex-1" title={entry.task}>
              {entry.task}
            </span>
            <span className="text-slate-500 tabular-nums shrink-0 ml-auto">
              {entry.score.toFixed(2)}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
