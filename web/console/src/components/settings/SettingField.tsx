import { useState, useRef, useEffect } from 'react'
import type { Setting, ProviderOption } from '@/types/settings'
import { Eye, EyeOff, ChevronDown, Search, Check } from 'lucide-react'

interface SettingFieldProps {
  setting: Setting
  providerOptions?: ProviderOption[]
  /** Current value of the provider-select this model-select depends on */
  dependsOnValue?: string
  onChange?: (key: string, value: string | number | boolean) => void
  isModified?: boolean
}

export function SettingField({
  setting,
  providerOptions = [],
  dependsOnValue,
  onChange,
  isModified = false,
}: SettingFieldProps) {
  const [showSecret, setShowSecret] = useState(false)

  const baseLabel = (
    <div className="flex items-center gap-2 mb-1.5">
      <label className="text-sm font-medium text-slate-700 dark:text-slate-300">
        {setting.label}
        {setting.required && (
          <span className="text-red-500 ml-0.5">*</span>
        )}
      </label>
      {isModified && (
        <span className="inline-flex items-center px-1.5 py-0.5 text-[10px] font-semibold rounded bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400">
          Modified
        </span>
      )}
    </div>
  )

  const description = (
    <p className="mt-1 text-xs text-slate-400 dark:text-slate-500">
      {setting.description}
    </p>
  )

  const envKey = (
    <span className="mt-0.5 block text-[10px] font-mono text-slate-400 dark:text-slate-600">
      {setting.key}
    </span>
  )

  if (setting.type === 'toggle') {
    return (
      <div className="flex items-start justify-between gap-4 py-3">
        <div className="flex-1 min-w-0">
          {baseLabel}
          <p className="text-xs text-slate-400 dark:text-slate-500">
            {setting.description}
          </p>
          {envKey}
        </div>
        <button
          onClick={() => onChange?.(setting.key, !setting.value)}
          className={`relative mt-0.5 inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ${
            setting.value
              ? 'bg-indigo-600 dark:bg-indigo-500'
              : 'bg-slate-200 dark:bg-slate-600'
          }`}
        >
          <span
            className={`pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow-sm ring-0 transition-transform duration-200 ${
              setting.value ? 'translate-x-5' : 'translate-x-0'
            }`}
          />
        </button>
      </div>
    )
  }

  if (setting.type === 'provider-select') {
    return (
      <div className="py-3">
        {baseLabel}
        <SearchableSelect
          options={providerOptions.map((p) => ({ id: p.id, label: p.name }))}
          value={String(setting.value)}
          placeholder="Select provider..."
          onChange={(val) => onChange?.(setting.key, val)}
        />
        {description}
        {envKey}
      </div>
    )
  }

  if (setting.type === 'model-select') {
    const provider = providerOptions.find((p) => p.id === dependsOnValue)
    const models = provider?.models ?? []
    return (
      <div className="py-3">
        {baseLabel}
        <SearchableSelect
          options={models.map((m) => ({ id: m.id, label: m.name }))}
          value={String(setting.value)}
          placeholder={
            dependsOnValue ? 'Select model...' : 'Select a provider first'
          }
          disabled={!dependsOnValue}
          onChange={(val) => onChange?.(setting.key, val)}
        />
        {description}
        {envKey}
      </div>
    )
  }

  if (setting.type === 'secret') {
    return (
      <div className="py-3">
        {baseLabel}
        <div className="relative">
          <input
            type={showSecret ? 'text' : 'password'}
            value={String(setting.value)}
            onChange={(e) => onChange?.(setting.key, e.target.value)}
            className="w-full px-3 py-2 pr-10 text-sm font-mono rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 transition-colors"
          />
          <button
            onClick={() => setShowSecret(!showSecret)}
            className="absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
          >
            {showSecret ? (
              <EyeOff className="w-4 h-4" strokeWidth={1.5} />
            ) : (
              <Eye className="w-4 h-4" strokeWidth={1.5} />
            )}
          </button>
        </div>
        {description}
        {envKey}
      </div>
    )
  }

  if (setting.type === 'number') {
    return (
      <div className="py-3">
        {baseLabel}
        <input
          type="number"
          value={Number(setting.value)}
          min={setting.validation?.min}
          max={setting.validation?.max}
          step={setting.validation?.step}
          onChange={(e) => onChange?.(setting.key, Number(e.target.value))}
          className="w-full max-w-xs px-3 py-2 text-sm font-mono rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 transition-colors"
        />
        {description}
        {envKey}
      </div>
    )
  }

  // text, url, duration — all render as text inputs
  return (
    <div className="py-3">
      {baseLabel}
      <input
        type="text"
        value={String(setting.value)}
        onChange={(e) => onChange?.(setting.key, e.target.value)}
        className={`w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500/40 focus:border-indigo-500 transition-colors ${
          setting.type === 'duration' || setting.type === 'url'
            ? 'font-mono'
            : ''
        }`}
        placeholder={
          setting.type === 'duration'
            ? 'e.g., 30s, 2m, 1h'
            : setting.type === 'url'
              ? 'e.g., postgres://...'
              : ''
        }
      />
      {description}
      {envKey}
    </div>
  )
}

/* ── Searchable Select ────────────────────────────────────────────── */

interface SelectOption {
  id: string
  label: string
}

function SearchableSelect({
  options,
  value,
  placeholder,
  disabled,
  onChange,
}: {
  options: SelectOption[]
  value: string
  placeholder: string
  disabled?: boolean
  onChange?: (val: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setSearch('')
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const filtered = options.filter((o) =>
    o.label.toLowerCase().includes(search.toLowerCase())
  )
  const selectedLabel = options.find((o) => o.id === value)?.label

  return (
    <div ref={ref} className="relative w-full max-w-sm">
      <button
        onClick={() => !disabled && setOpen(!open)}
        disabled={disabled}
        className={`w-full flex items-center justify-between px-3 py-2 text-sm rounded-lg border transition-colors ${
          disabled
            ? 'border-slate-100 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-slate-400 dark:text-slate-500 cursor-not-allowed'
            : 'border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 hover:border-slate-300 dark:hover:border-slate-500'
        } ${open ? 'ring-2 ring-indigo-500/40 border-indigo-500' : ''}`}
      >
        <span className={selectedLabel ? '' : 'text-slate-400 dark:text-slate-500'}>
          {selectedLabel || placeholder}
        </span>
        <ChevronDown
          className={`w-4 h-4 text-slate-400 transition-transform ${open ? 'rotate-180' : ''}`}
          strokeWidth={1.5}
        />
      </button>

      {open && (
        <div className="absolute z-50 mt-1 w-full rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 shadow-lg overflow-hidden">
          {options.length > 4 && (
            <div className="p-2 border-b border-slate-100 dark:border-slate-600">
              <div className="relative">
                <Search
                  className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-400"
                  strokeWidth={1.5}
                />
                <input
                  autoFocus
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search..."
                  className="w-full pl-8 pr-3 py-1.5 text-sm rounded-md bg-slate-50 dark:bg-slate-600 border-0 text-slate-800 dark:text-slate-200 placeholder:text-slate-400 focus:outline-none"
                />
              </div>
            </div>
          )}
          <div className="max-h-48 overflow-y-auto py-1">
            {filtered.length === 0 ? (
              <div className="px-3 py-2 text-sm text-slate-400 dark:text-slate-500">
                No results
              </div>
            ) : (
              filtered.map((opt) => (
                <button
                  key={opt.id}
                  onClick={() => {
                    onChange?.(opt.id)
                    setOpen(false)
                    setSearch('')
                  }}
                  className={`w-full flex items-center gap-2 px-3 py-2 text-sm text-left transition-colors ${
                    opt.id === value
                      ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300'
                      : 'text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-600'
                  }`}
                >
                  {opt.id === value && (
                    <Check className="w-3.5 h-3.5 shrink-0" strokeWidth={2} />
                  )}
                  <span className={opt.id === value ? '' : 'ml-5.5'}>
                    {opt.label}
                  </span>
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  )
}
