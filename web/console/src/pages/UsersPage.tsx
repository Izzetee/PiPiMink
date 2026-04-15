import { useState, useEffect, useCallback } from 'react'
import { Users, Loader2 } from 'lucide-react'
import { AuthAndUsers } from '@/components/auth'
import {
  fetchAuthProviders,
  saveAuthProvider,
  testAuthProvider,
  fetchUsers,
  addLocalUser,
  changeUserRole,
  deleteUser,
  fetchGroups,
  changeGroupRole,
  addRoutingRule,
  removeRoutingRule,
  fetchAuditLog,
} from '@/api/auth'
import type { AuthProvider, User, Group, AuditEntry, UserRole, RoutingRule } from '@/types/auth'

export function UsersPage() {
  const [providers, setProviders] = useState<AuthProvider[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [auditLog, setAuditLog] = useState<AuditEntry[]>([])
  const [isLoading, setIsLoading] = useState(true)

  const loadData = useCallback(async () => {
    try {
      const [p, u, g, a] = await Promise.all([
        fetchAuthProviders(),
        fetchUsers(),
        fetchGroups(),
        fetchAuditLog(),
      ])
      setProviders(p)
      setUsers(u)
      setGroups(g)
      setAuditLog(a)
    } catch (err) {
      console.error('Failed to load auth data:', err)
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => { loadData() }, [loadData])

  const handleSaveProvider = useCallback(async (provider: AuthProvider) => {
    try {
      await saveAuthProvider(provider)
      await loadData()
    } catch (err) {
      console.error('Failed to save provider:', err)
    }
  }, [loadData])

  const handleTestProvider = useCallback(async (providerId: string) => {
    try {
      await testAuthProvider(providerId)
      await loadData()
    } catch (err) {
      console.error('Failed to test provider:', err)
    }
  }, [loadData])

  const handleChangeUserRole = useCallback(async (userId: string, role: UserRole) => {
    try {
      await changeUserRole(userId, role)
      await loadData()
    } catch (err) {
      console.error('Failed to change user role:', err)
    }
  }, [loadData])

  const handleDeleteUser = useCallback(async (userId: string, reason: string) => {
    try {
      await deleteUser(userId, reason)
      await loadData()
    } catch (err) {
      console.error('Failed to delete user:', err)
    }
  }, [loadData])

  const handleAddLocalUser = useCallback(async (name: string, email: string, role: UserRole) => {
    try {
      await addLocalUser(name, email, role)
      await loadData()
    } catch (err) {
      console.error('Failed to add local user:', err)
    }
  }, [loadData])

  const handleChangeGroupRole = useCallback(async (groupId: string, role: UserRole) => {
    try {
      await changeGroupRole(groupId, role)
      await loadData()
    } catch (err) {
      console.error('Failed to change group role:', err)
    }
  }, [loadData])

  const handleAddRoutingRule = useCallback(async (groupId: string, rule: Omit<RoutingRule, 'id'>) => {
    try {
      await addRoutingRule(groupId, rule)
      await loadData()
    } catch (err) {
      console.error('Failed to add routing rule:', err)
    }
  }, [loadData])

  const handleRemoveRoutingRule = useCallback(async (groupId: string, ruleId: string) => {
    try {
      await removeRoutingRule(groupId, ruleId)
      await loadData()
    } catch (err) {
      console.error('Failed to remove routing rule:', err)
    }
  }, [loadData])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-6 h-6 text-indigo-500 animate-spin" />
      </div>
    )
  }

  return (
    <div className="p-6">
      <div className="flex items-center gap-3 mb-4">
        <Users className="w-6 h-6 text-indigo-600 dark:text-indigo-400" strokeWidth={1.75} />
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">Users</h1>
      </div>
      <AuthAndUsers
        authProviders={providers}
        users={users}
        groups={groups}
        auditLog={auditLog}
        onSaveProvider={handleSaveProvider}
        onTestProvider={handleTestProvider}
        onChangeUserRole={handleChangeUserRole}
        onDeleteUser={handleDeleteUser}
        onAddLocalUser={handleAddLocalUser}
        onChangeGroupRole={handleChangeGroupRole}
        onAddRoutingRule={handleAddRoutingRule}
        onRemoveRoutingRule={handleRemoveRoutingRule}
      />
    </div>
  )
}
