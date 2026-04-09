import { useState, useMemo } from 'react'
import type { AuthAndUsersProps } from '@/types/auth'
import { AuthProviderTab } from './AuthProviderTab'
import { UsersTab } from './UsersTab'
import { GroupsTab } from './GroupsTab'
import { AuditLogTab } from './AuditLogTab'
import { Shield, Users, FolderKey, ScrollText } from 'lucide-react'

type TabId = 'providers' | 'users' | 'groups' | 'audit'

interface TabDef {
  id: TabId
  label: string
  icon: React.ElementType
}

export function AuthAndUsers({
  authProviders,
  users,
  groups,
  auditLog,
  onSaveProvider,
  onTestProvider,
  onChangeUserRole,
  onDeleteUser,
  onAddLocalUser,
  onChangeGroupRole,
  onAddRoutingRule,
  onRemoveRoutingRule,
}: AuthAndUsersProps) {
  const hasExternalProvider = authProviders.some(
    (p) => (p.type === 'oauth' || p.type === 'ldap') && p.status === 'connected'
  )

  const TABS: TabDef[] = useMemo(() => {
    const tabs: TabDef[] = [
      { id: 'providers', label: 'Auth Provider', icon: Shield },
      { id: 'users', label: 'Users', icon: Users },
    ]
    if (hasExternalProvider) {
      tabs.push({ id: 'groups', label: 'Groups', icon: FolderKey })
    }
    tabs.push({ id: 'audit', label: 'Audit Log', icon: ScrollText })
    return tabs
  }, [hasExternalProvider])

  const [activeTab, setActiveTab] = useState<TabId>('providers')

  return (
    <div className="h-full flex flex-col bg-slate-50 dark:bg-slate-900">
      {/* Header */}
      <div className="shrink-0 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 pt-5 pb-0">
          <h2 className="text-lg font-semibold text-slate-800 dark:text-slate-200 mb-4">
            Auth & Users
          </h2>

          {/* Tabs */}
          <div className="flex gap-0 overflow-x-auto scrollbar-hide -mb-px">
            {TABS.map((tab) => {
              const Icon = tab.icon
              const isActive = activeTab === tab.id
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
                </button>
              )
            })}
          </div>
        </div>
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-y-auto">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 py-6">
          {activeTab === 'providers' && (
            <AuthProviderTab
              providers={authProviders}
              onSave={onSaveProvider}
              onTest={onTestProvider}
            />
          )}
          {activeTab === 'users' && (
            <UsersTab
              users={users}
              hasExternalProvider={hasExternalProvider}
              onChangeRole={onChangeUserRole}
              onDelete={onDeleteUser}
              onAddLocalUser={onAddLocalUser}
            />
          )}
          {activeTab === 'groups' && (
            <GroupsTab
              groups={groups}
              onChangeRole={onChangeGroupRole}
              onAddRule={onAddRoutingRule}
              onRemoveRule={onRemoveRoutingRule}
            />
          )}
          {activeTab === 'audit' && <AuditLogTab entries={auditLog} />}
        </div>
      </div>
    </div>
  )
}
