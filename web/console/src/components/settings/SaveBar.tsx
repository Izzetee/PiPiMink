import type { PendingChange } from '@/types/settings'
import { Save, Undo2, AlertCircle } from 'lucide-react'

interface SaveBarProps {
  pendingChanges: PendingChange[]
  onSave?: (changes: PendingChange[]) => void
  onDiscard?: () => void
}

export function SaveBar({ pendingChanges, onSave, onDiscard }: SaveBarProps) {
  if (pendingChanges.length === 0) return null

  return (
    <div className="fixed bottom-0 left-0 right-0 z-40">
      <div className="mx-auto max-w-5xl px-4 pb-4">
        <div className="flex items-center justify-between gap-4 px-5 py-3 rounded-xl bg-slate-900 dark:bg-slate-700 text-white shadow-2xl border border-slate-700 dark:border-slate-600">
          <div className="flex items-center gap-3 min-w-0">
            <div className="w-8 h-8 rounded-full bg-amber-500/20 flex items-center justify-center shrink-0">
              <AlertCircle className="w-4 h-4 text-amber-400" strokeWidth={1.5} />
            </div>
            <div className="min-w-0">
              <p className="text-sm font-medium">
                {pendingChanges.length} unsaved{' '}
                {pendingChanges.length === 1 ? 'change' : 'changes'}
              </p>
              <p className="text-xs text-slate-400 truncate">
                {pendingChanges.map((c) => c.key).join(', ')}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            <button
              onClick={() => onDiscard?.()}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg text-slate-300 hover:text-white hover:bg-slate-700 dark:hover:bg-slate-600 transition-colors"
            >
              <Undo2 className="w-3.5 h-3.5" strokeWidth={1.5} />
              Discard
            </button>
            <button
              onClick={() => onSave?.(pendingChanges)}
              className="inline-flex items-center gap-1.5 px-4 py-1.5 text-sm font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-500 transition-colors shadow-sm"
            >
              <Save className="w-3.5 h-3.5" strokeWidth={1.5} />
              Save Changes
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
