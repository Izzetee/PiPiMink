import { useState } from 'react'
import type {
  Group,
  UserRole,
  RoutingRule,
} from '@/types/auth'
import {
  ChevronDown,
  ChevronRight,
  Users,
  ShieldCheck,
  RefreshCw,
  Plus,
  X,
  Ban,
  Check,
} from 'lucide-react'

interface GroupsTabProps {
  groups: Group[]
  onChangeRole?: (groupId: string, role: UserRole) => void
  onAddRule?: (groupId: string, rule: Omit<RoutingRule, 'id'>) => void
  onRemoveRule?: (groupId: string, ruleId: string) => void
}

export function GroupsTab({
  groups,
  onChangeRole,
  onAddRule,
  onRemoveRule,
}: GroupsTabProps) {
  return (
    <div className="space-y-4">
      {/* Sync info */}
      <div className="flex items-center justify-between">
        <p className="text-xs text-slate-400 dark:text-slate-500">
          {groups.length} group{groups.length !== 1 ? 's' : ''} synced from
          identity provider
        </p>
        <button className="inline-flex items-center gap-1.5 text-xs text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-300 font-medium">
          <RefreshCw className="w-3 h-3" />
          Sync Now
        </button>
      </div>

      {groups.map((group) => (
        <GroupCard
          key={group.id}
          group={group}
          onChangeRole={onChangeRole}
          onAddRule={onAddRule}
          onRemoveRule={onRemoveRule}
        />
      ))}
    </div>
  )
}

// --- Group Card ---

function GroupCard({
  group,
  onChangeRole,
  onAddRule,
  onRemoveRule,
}: {
  group: Group
  onChangeRole?: (groupId: string, role: UserRole) => void
  onAddRule?: (groupId: string, rule: Omit<RoutingRule, 'id'>) => void
  onRemoveRule?: (groupId: string, ruleId: string) => void
}) {
  const [isExpanded, setIsExpanded] = useState(false)
  const [showAddRule, setShowAddRule] = useState(false)

  return (
    <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm overflow-hidden">
      {/* Header */}
      <div className="px-4 sm:px-6 py-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-amber-50 dark:bg-amber-900/20 flex items-center justify-center">
            <Users className="w-5 h-5 text-amber-600 dark:text-amber-400" strokeWidth={1.5} />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
                {group.name}
              </h3>
              <span className="text-[10px] font-mono text-slate-400 dark:text-slate-500 uppercase">
                {group.source}
              </span>
            </div>
            <p className="text-xs text-slate-400 dark:text-slate-500">
              {group.memberCount} member{group.memberCount !== 1 ? 's' : ''} · Last
              synced{' '}
              {new Date(group.syncedAt).toLocaleDateString('en-US', {
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit',
              })}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Role dropdown */}
          <div className="flex items-center gap-2">
            <ShieldCheck className="w-4 h-4 text-slate-400 dark:text-slate-500" />
            <select
              value={group.role}
              onChange={(e) =>
                onChangeRole?.(group.id, e.target.value as UserRole)
              }
              className={`text-xs font-medium rounded-full px-2.5 py-1 border-0 cursor-pointer focus:outline-none focus:ring-2 focus:ring-indigo-500/30 ${
                group.role === 'admin'
                  ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300'
                  : 'bg-slate-100 dark:bg-slate-700/50 text-slate-600 dark:text-slate-400'
              }`}
            >
              <option value="admin">Admin</option>
              <option value="user">User</option>
            </select>
          </div>
        </div>
      </div>

      {/* Routing Rules Toggle */}
      <button
        onClick={() => setIsExpanded(!isExpanded)}
        className="w-full px-4 sm:px-6 py-2.5 flex items-center justify-between border-t border-slate-100 dark:border-slate-700/50 hover:bg-slate-50 dark:hover:bg-slate-700/30 transition-colors"
      >
        <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
          Routing Rules
          {group.routingRules.length > 0 && (
            <span className="ml-1.5 inline-flex items-center justify-center w-4 h-4 text-[10px] font-bold rounded-full bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400">
              {group.routingRules.length}
            </span>
          )}
        </span>
        {isExpanded ? (
          <ChevronDown className="w-4 h-4 text-slate-400" />
        ) : (
          <ChevronRight className="w-4 h-4 text-slate-400" />
        )}
      </button>

      {/* Expanded Rules */}
      {isExpanded && (
        <div className="px-4 sm:px-6 pb-4 space-y-2">
          {group.routingRules.length === 0 ? (
            <p className="text-xs text-slate-400 dark:text-slate-500 italic py-2">
              No routing rules — members have default access to all
              models and providers.
            </p>
          ) : (
            group.routingRules.map((rule) => (
              <div
                key={rule.id}
                className="flex items-start justify-between gap-2 px-3 py-2.5 rounded-lg bg-slate-50 dark:bg-slate-700/30 border border-slate-100 dark:border-slate-700/50"
              >
                <div className="flex items-start gap-2">
                  <RuleIcon type={rule.type} />
                  <div>
                    <p className="text-xs font-medium text-slate-700 dark:text-slate-300">
                      {rule.description}
                    </p>
                    <p className="text-[10px] font-mono text-slate-400 dark:text-slate-500 mt-0.5">
                      {rule.type.replace(/_/g, ' ')}
                      {rule.providers &&
                        ` · ${rule.providers.join(', ')}`}
                      {rule.models &&
                        ` · ${rule.models.join(', ')}`}
                    </p>
                  </div>
                </div>
                <button
                  onClick={() => onRemoveRule?.(group.id, rule.id)}
                  className="p-1 rounded hover:bg-red-50 dark:hover:bg-red-900/20 text-slate-400 hover:text-red-500 dark:hover:text-red-400 transition-colors shrink-0"
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              </div>
            ))
          )}

          {!showAddRule ? (
            <button
              onClick={() => setShowAddRule(true)}
              className="inline-flex items-center gap-1.5 text-xs font-medium text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-300 mt-1"
            >
              <Plus className="w-3.5 h-3.5" />
              Add Routing Rule
            </button>
          ) : (
            <AddRuleForm
              onAdd={(rule) => {
                onAddRule?.(group.id, rule)
                setShowAddRule(false)
              }}
              onCancel={() => setShowAddRule(false)}
            />
          )}
        </div>
      )}
    </div>
  )
}

function RuleIcon({ type }: { type: string }) {
  if (type.startsWith('allow'))
    return (
      <Check className="w-4 h-4 text-emerald-500 dark:text-emerald-400 shrink-0 mt-0.5" />
    )
  return (
    <Ban className="w-4 h-4 text-red-500 dark:text-red-400 shrink-0 mt-0.5" />
  )
}

// --- Add Rule Form ---

function AddRuleForm({
  onAdd,
  onCancel,
}: {
  onAdd: (rule: Omit<RoutingRule, 'id'>) => void
  onCancel: () => void
}) {
  const [type, setType] = useState<string>('allow_providers')
  const [description, setDescription] = useState('')
  const [values, setValues] = useState('')

  function handleSubmit() {
    if (!description.trim()) return
    const rule: Omit<RoutingRule, 'id'> = {
      type: type as RoutingRule['type'],
      description: description.trim(),
    }
    const parsedValues = values
      .split(',')
      .map((v) => v.trim())
      .filter(Boolean)
    if (type.includes('provider') && parsedValues.length > 0) {
      rule.providers = parsedValues
    }
    if (type.includes('model') && parsedValues.length > 0) {
      rule.models = parsedValues
    }
    onAdd(rule)
  }

  return (
    <div className="p-3 rounded-lg border border-indigo-200 dark:border-indigo-800/50 bg-indigo-50/50 dark:bg-indigo-900/10 space-y-2.5">
      <div className="flex items-center gap-2">
        <select
          value={type}
          onChange={(e) => setType(e.target.value)}
          className="flex-1 px-2.5 py-1.5 text-xs rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
        >
          <option value="allow_all">Allow All</option>
          <option value="allow_providers">Allow Providers</option>
          <option value="deny_providers">Deny Providers</option>
          <option value="allow_models">Allow Models</option>
          <option value="deny_models">Deny Models</option>
        </select>
      </div>

      {type !== 'allow_all' && (
        <input
          type="text"
          value={values}
          onChange={(e) => setValues(e.target.value)}
          placeholder={
            type.includes('provider')
              ? 'e.g., OpenAI, Anthropic'
              : 'e.g., gpt-4o, claude-3.5-sonnet'
          }
          className="w-full px-2.5 py-1.5 text-xs font-mono rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
        />
      )}

      <input
        type="text"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        placeholder="Description of this rule"
        className="w-full px-2.5 py-1.5 text-xs rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
      />

      <div className="flex justify-end gap-2">
        <button
          onClick={onCancel}
          className="px-3 py-1.5 text-xs font-medium rounded-lg border border-slate-200 dark:border-slate-600 text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
        >
          Cancel
        </button>
        <button
          onClick={handleSubmit}
          disabled={!description.trim()}
          className="px-3 py-1.5 text-xs font-medium rounded-lg bg-indigo-600 dark:bg-indigo-500 text-white hover:bg-indigo-700 dark:hover:bg-indigo-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          Add Rule
        </button>
      </div>
    </div>
  )
}
