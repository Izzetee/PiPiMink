import { useState } from 'react'
import type { AuditEntry, AuditAction } from '@/types/auth'
import {
  Shield,
  UserPlus,
  UserMinus,
  ArrowRightLeft,
  Settings,
  Wifi,
  CheckCircle2,
  Filter,
  MessageSquare,
} from 'lucide-react'

interface AuditLogTabProps {
  entries: AuditEntry[]
}

const ACTION_CONFIG: Record<
  AuditAction,
  { icon: React.ElementType; label: string; color: string }
> = {
  provider_configured: {
    icon: Settings,
    label: 'Provider Configured',
    color: 'text-indigo-600 dark:text-indigo-400 bg-indigo-50 dark:bg-indigo-900/30',
  },
  provider_verified: {
    icon: CheckCircle2,
    label: 'Provider Verified',
    color: 'text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-900/30',
  },
  user_created: {
    icon: UserPlus,
    label: 'User Created',
    color: 'text-sky-600 dark:text-sky-400 bg-sky-50 dark:bg-sky-900/30',
  },
  user_deleted: {
    icon: UserMinus,
    label: 'User Deleted',
    color: 'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/30',
  },
  role_changed: {
    icon: ArrowRightLeft,
    label: 'Role Changed',
    color: 'text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/30',
  },
  group_role_changed: {
    icon: Shield,
    label: 'Group Role Changed',
    color: 'text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/30',
  },
  group_routing_updated: {
    icon: Wifi,
    label: 'Routing Updated',
    color: 'text-violet-600 dark:text-violet-400 bg-violet-50 dark:bg-violet-900/30',
  },
}

export function AuditLogTab({ entries }: AuditLogTabProps) {
  const [filterAction, setFilterAction] = useState<'all' | AuditAction>('all')

  const filtered = entries.filter(
    (e) => filterAction === 'all' || e.action === filterAction
  )

  const uniqueActions = [...new Set(entries.map((e) => e.action))]

  return (
    <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm overflow-hidden">
      {/* Toolbar */}
      <div className="px-4 sm:px-6 py-3 flex items-center justify-between border-b border-slate-100 dark:border-slate-700/50">
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-slate-400" />
          <select
            value={filterAction}
            onChange={(e) =>
              setFilterAction(e.target.value as typeof filterAction)
            }
            className="px-3 py-1.5 text-xs rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
          >
            <option value="all">All Actions</option>
            {uniqueActions.map((action) => (
              <option key={action} value={action}>
                {ACTION_CONFIG[action]?.label ?? action}
              </option>
            ))}
          </select>
        </div>
        <span className="text-xs text-slate-400 dark:text-slate-500">
          {filtered.length} entr{filtered.length !== 1 ? 'ies' : 'y'}
        </span>
      </div>

      {/* Timeline */}
      <div className="divide-y divide-slate-100 dark:divide-slate-700/50">
        {filtered.map((entry) => {
          const config = ACTION_CONFIG[entry.action]
          const Icon = config?.icon ?? Settings
          const color = config?.color ?? 'text-slate-500 bg-slate-100'

          return (
            <div
              key={entry.id}
              className="px-4 sm:px-6 py-4 hover:bg-slate-50 dark:hover:bg-slate-700/20 transition-colors"
            >
              <div className="flex gap-3">
                {/* Icon */}
                <div
                  className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 ${color}`}
                >
                  <Icon className="w-4 h-4" />
                </div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                  <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
                        {entry.actor}
                      </span>
                      <span
                        className={`text-[10px] font-medium rounded-full px-2 py-0.5 ${color}`}
                      >
                        {config?.label ?? entry.action}
                      </span>
                      <span className="text-sm text-slate-500 dark:text-slate-400">
                        {entry.target}
                      </span>
                    </div>
                    <time className="text-xs text-slate-400 dark:text-slate-500 whitespace-nowrap">
                      {new Date(entry.timestamp).toLocaleDateString('en-US', {
                        month: 'short',
                        day: 'numeric',
                        year: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit',
                      })}
                    </time>
                  </div>

                  <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                    {entry.details}
                  </p>

                  {entry.reason && (
                    <div className="mt-2 flex items-start gap-1.5 text-xs text-slate-500 dark:text-slate-400">
                      <MessageSquare className="w-3 h-3 mt-0.5 shrink-0" />
                      <span className="italic">{entry.reason}</span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
