import { useState } from 'react'
import type { TimeRange, CustomDateRange } from '@/types/analytics'
import { Calendar } from 'lucide-react'

interface TimeRangeSelectorProps {
  onTimeRangeChange?: (range: TimeRange) => void
  onCustomDateRange?: (range: CustomDateRange) => void
}

const presets: { value: TimeRange; label: string }[] = [
  { value: '1h', label: '1h' },
  { value: '24h', label: '24h' },
  { value: '7d', label: '7d' },
  { value: '30d', label: '30d' },
]

export function TimeRangeSelector({ onTimeRangeChange, onCustomDateRange }: TimeRangeSelectorProps) {
  const [active, setActive] = useState<TimeRange>('24h')
  const [showCustom, setShowCustom] = useState(false)
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')

  function handlePreset(range: TimeRange) {
    setActive(range)
    setShowCustom(false)
    onTimeRangeChange?.(range)
  }

  function handleCustomToggle() {
    setShowCustom(!showCustom)
    if (!showCustom) {
      setActive('custom')
      onTimeRangeChange?.('custom')
    }
  }

  function handleApplyCustom() {
    if (startDate && endDate) {
      onCustomDateRange?.({ start: startDate, end: endDate })
    }
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <div className="inline-flex rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/60 p-0.5">
        {presets.map((p) => (
          <button
            key={p.value}
            onClick={() => handlePreset(p.value)}
            className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
              active === p.value && !showCustom
                ? 'bg-indigo-600 text-white shadow-sm'
                : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
            }`}
          >
            {p.label}
          </button>
        ))}
      </div>

      <button
        onClick={handleCustomToggle}
        className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors ${
          showCustom
            ? 'border-indigo-300 dark:border-indigo-600 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-700 dark:text-indigo-300'
            : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800'
        }`}
      >
        <Calendar className="w-3.5 h-3.5" strokeWidth={2} />
        Custom
      </button>

      {showCustom && (
        <div className="flex items-center gap-2 ml-1">
          <input
            type="date"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            className="px-2 py-1.5 text-xs rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
          />
          <span className="text-xs text-slate-400">to</span>
          <input
            type="date"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="px-2 py-1.5 text-xs rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
          />
          <button
            onClick={handleApplyCustom}
            disabled={!startDate || !endDate}
            className="px-3 py-1.5 text-xs font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Apply
          </button>
        </div>
      )}
    </div>
  )
}
