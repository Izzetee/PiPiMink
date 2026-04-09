import { useState, useMemo } from 'react'
import type {
  SettingsProps,
  SettingCategory,
} from '@/types/settings'
import { SettingsPanel } from './SettingsPanel'
import { ApiKeyVault } from './ApiKeyVault'
import { SaveBar } from './SaveBar'
import {
  Route,
  Database,
  Server,
  FlaskConical,
  Activity,
  HardDrive,
  KeyRound,
} from 'lucide-react'

type TabId = SettingCategory | 'apiKeys'

interface TabDef {
  id: TabId
  label: string
  icon: React.ElementType
}

const TABS: TabDef[] = [
  { id: 'routing', label: 'Routing', icon: Route },
  { id: 'cache', label: 'Cache', icon: HardDrive },
  { id: 'database', label: 'Database', icon: Database },
  { id: 'server', label: 'Server', icon: Server },
  { id: 'benchmarking', label: 'Benchmarking', icon: FlaskConical },
  { id: 'observability', label: 'Observability', icon: Activity },
  { id: 'apiKeys', label: 'API Keys', icon: KeyRound },
]

export function Settings({
  settings,
  apiKeys,
  providerOptions,
  pendingChanges,
  onSettingChange,
  onSaveAll,
  onDiscardAll,
  onAddApiKey,
  onEditApiKey,
  onDeleteApiKey,
}: SettingsProps) {
  const [activeTab, setActiveTab] = useState<TabId>('routing')

  const modifiedKeys = useMemo(
    () => new Set(pendingChanges.map((c) => c.key)),
    [pendingChanges]
  )

  // Count modified settings per tab for badges
  const modifiedPerTab = useMemo(() => {
    const counts: Partial<Record<TabId, number>> = {}
    for (const change of pendingChanges) {
      const cat = change.category as TabId
      counts[cat] = (counts[cat] ?? 0) + 1
    }
    return counts
  }, [pendingChanges])

  return (
    <div className="h-full flex flex-col bg-slate-50 dark:bg-slate-900">
      {/* Header */}
      <div className="shrink-0 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 pt-5 pb-0">
          <h2 className="text-lg font-semibold text-slate-800 dark:text-slate-200 mb-4">
            Settings
          </h2>

          {/* Tabs */}
          <div className="flex gap-0 overflow-x-auto scrollbar-hide -mb-px">
            {TABS.map((tab) => {
              const Icon = tab.icon
              const isActive = activeTab === tab.id
              const modCount = modifiedPerTab[tab.id]
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`relative flex items-center gap-1.5 px-3 sm:px-4 py-2.5 text-sm font-medium whitespace-nowrap border-b-2 transition-colors ${
                    isActive
                      ? 'border-indigo-600 dark:border-indigo-400 text-indigo-700 dark:text-indigo-300'
                      : 'border-transparent text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300 hover:border-slate-300 dark:hover:border-slate-600'
                  }`}
                >
                  <Icon className="w-4 h-4 hidden sm:block" strokeWidth={1.5} />
                  {tab.label}
                  {modCount && modCount > 0 && (
                    <span className="ml-1 inline-flex items-center justify-center w-4.5 h-4.5 text-[10px] font-bold rounded-full bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400">
                      {modCount}
                    </span>
                  )}
                </button>
              )
            })}
          </div>
        </div>
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 py-6">
          <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm p-4 sm:p-6">
            {activeTab === 'apiKeys' ? (
              <ApiKeyVault
                apiKeys={apiKeys}
                providerOptions={providerOptions}
                onAdd={onAddApiKey}
                onEdit={onEditApiKey}
                onDelete={onDeleteApiKey}
              />
            ) : (
              <SettingsPanel
                settings={settings[activeTab]}
                providerOptions={providerOptions}
                modifiedKeys={modifiedKeys}
                onChange={onSettingChange}
              />
            )}
          </div>
        </div>

        {/* Spacer for save bar */}
        {pendingChanges.length > 0 && <div className="h-20" />}
      </div>

      {/* Global save bar */}
      <SaveBar
        pendingChanges={pendingChanges}
        onSave={onSaveAll}
        onDiscard={onDiscardAll}
      />
    </div>
  )
}
