import { useState } from 'react'
import type { User, UserRole } from '@/types/auth'
import {
  Search,
  MoreVertical,
  Trash2,
  ArrowUpDown,
  Plus,
  X,
  AlertTriangle,
} from 'lucide-react'

interface UsersTabProps {
  users: User[]
  hasExternalProvider: boolean
  onChangeRole?: (userId: string, role: UserRole) => void
  onDelete?: (userId: string, reason: string) => void
  onAddLocalUser?: (name: string, email: string, role: UserRole) => void
}

export function UsersTab({
  users,
  hasExternalProvider,
  onChangeRole,
  onDelete,
  onAddLocalUser,
}: UsersTabProps) {
  const [search, setSearch] = useState('')
  const [filterRole, setFilterRole] = useState<'all' | UserRole>('all')
  const [filterSource, setFilterSource] = useState<'all' | string>('all')
  const [menuOpenId, setMenuOpenId] = useState<string | null>(null)
  const [deleteUser, setDeleteUser] = useState<User | null>(null)
  const [deleteReason, setDeleteReason] = useState('')
  const [showAddUser, setShowAddUser] = useState(false)
  const [sortField, setSortField] = useState<'name' | 'lastLogin' | 'requestCount'>('name')
  const [sortAsc, setSortAsc] = useState(true)

  const filtered = users
    .filter((u) => {
      if (search) {
        const q = search.toLowerCase()
        if (
          !u.name.toLowerCase().includes(q) &&
          !u.email.toLowerCase().includes(q)
        )
          return false
      }
      if (filterRole !== 'all' && u.role !== filterRole) return false
      if (filterSource !== 'all' && u.authSource !== filterSource) return false
      return true
    })
    .sort((a, b) => {
      const dir = sortAsc ? 1 : -1
      if (sortField === 'name') return a.name.localeCompare(b.name) * dir
      if (sortField === 'lastLogin')
        return (
          (new Date(a.lastLogin).getTime() - new Date(b.lastLogin).getTime()) *
          dir
        )
      return (a.requestCount - b.requestCount) * dir
    })

  function toggleSort(field: typeof sortField) {
    if (sortField === field) setSortAsc(!sortAsc)
    else {
      setSortField(field)
      setSortAsc(true)
    }
  }

  function handleDelete() {
    if (deleteUser && deleteReason.trim()) {
      onDelete?.(deleteUser.id, deleteReason.trim())
      setDeleteUser(null)
      setDeleteReason('')
    }
  }

  function formatTokens(n: number) {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
    return String(n)
  }

  function formatDate(d: string) {
    return new Date(d).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    })
  }

  return (
    <>
      {/* Toolbar */}
      <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm">
        <div className="px-4 sm:px-6 py-4 flex flex-col sm:flex-row items-stretch sm:items-center gap-3 border-b border-slate-100 dark:border-slate-700/50">
          {/* Search */}
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder="Search users..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-500"
            />
          </div>

          {/* Filters */}
          <div className="flex items-center gap-2">
            <select
              value={filterRole}
              onChange={(e) => setFilterRole(e.target.value as typeof filterRole)}
              className="px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
            >
              <option value="all">All Roles</option>
              <option value="admin">Admin</option>
              <option value="user">User</option>
            </select>
            <select
              value={filterSource}
              onChange={(e) => setFilterSource(e.target.value)}
              className="px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
            >
              <option value="all">All Sources</option>
              <option value="oauth">OAuth</option>
              <option value="ldap">LDAP</option>
              <option value="local">Local</option>
            </select>

            {!hasExternalProvider && (
              <button
                onClick={() => setShowAddUser(true)}
                className="inline-flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg bg-indigo-600 dark:bg-indigo-500 text-white hover:bg-indigo-700 dark:hover:bg-indigo-600 transition-colors"
              >
                <Plus className="w-4 h-4" />
                <span className="hidden sm:inline">Add User</span>
              </button>
            )}
          </div>
        </div>

        {/* User count */}
        <div className="px-4 sm:px-6 py-2 text-xs text-slate-400 dark:text-slate-500 border-b border-slate-100 dark:border-slate-700/50">
          {filtered.length} user{filtered.length !== 1 ? 's' : ''}
          {search || filterRole !== 'all' || filterSource !== 'all'
            ? ` (filtered from ${users.length})`
            : ''}
        </div>

        {/* Desktop table */}
        <div className="overflow-x-auto max-sm:hidden">
          <table className="w-full text-sm text-left">
            <thead>
              <tr className="text-xs font-medium text-slate-400 dark:text-slate-500 uppercase tracking-wider">
                <th className="px-6 py-3">
                  <button
                    onClick={() => toggleSort('name')}
                    className="flex items-center gap-1 hover:text-slate-600 dark:hover:text-slate-300"
                  >
                    User
                    <ArrowUpDown className="w-3 h-3" />
                  </button>
                </th>
                <th className="px-6 py-3">Role</th>
                <th className="px-6 py-3">Source</th>
                <th className="px-6 py-3">Groups</th>
                <th className="px-6 py-3">
                  <button
                    onClick={() => toggleSort('lastLogin')}
                    className="flex items-center gap-1 hover:text-slate-600 dark:hover:text-slate-300"
                  >
                    Last Login
                    <ArrowUpDown className="w-3 h-3" />
                  </button>
                </th>
                <th className="px-6 py-3">
                  <button
                    onClick={() => toggleSort('requestCount')}
                    className="flex items-center gap-1 hover:text-slate-600 dark:hover:text-slate-300"
                  >
                    Requests
                    <ArrowUpDown className="w-3 h-3" />
                  </button>
                </th>
                <th className="px-6 py-3">Tokens</th>
                <th className="px-6 py-3 w-10"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-slate-700/50">
              {filtered.map((user) => (
                <tr
                  key={user.id}
                  className="hover:bg-slate-50 dark:hover:bg-slate-700/30 transition-colors"
                >
                  <td className="px-6 py-3">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-full bg-slate-200 dark:bg-slate-600 flex items-center justify-center text-xs font-bold text-slate-600 dark:text-slate-300">
                        {user.name
                          .split(' ')
                          .map((n) => n[0])
                          .join('')
                          .slice(0, 2)
                          .toUpperCase()}
                      </div>
                      <div>
                        <p className="font-medium text-slate-800 dark:text-slate-200">
                          {user.name}
                        </p>
                        <p className="text-xs text-slate-400 dark:text-slate-500">
                          {user.email}
                        </p>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-3">
                    <select
                      value={user.role}
                      onChange={(e) =>
                        onChangeRole?.(user.id, e.target.value as UserRole)
                      }
                      className={`text-xs font-medium rounded-full px-2.5 py-1 border-0 cursor-pointer focus:outline-none focus:ring-2 focus:ring-indigo-500/30 ${
                        user.role === 'admin'
                          ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300'
                          : 'bg-slate-100 dark:bg-slate-700/50 text-slate-600 dark:text-slate-400'
                      }`}
                    >
                      <option value="admin">Admin</option>
                      <option value="user">User</option>
                    </select>
                  </td>
                  <td className="px-6 py-3">
                    <span className="text-xs font-mono text-slate-500 dark:text-slate-400 uppercase">
                      {user.authSource}
                    </span>
                  </td>
                  <td className="px-6 py-3">
                    {user.groups.length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {user.groups.map((g) => (
                          <span
                            key={g}
                            className="text-[10px] font-medium px-1.5 py-0.5 rounded bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-400"
                          >
                            {g}
                          </span>
                        ))}
                      </div>
                    ) : (
                      <span className="text-xs text-slate-300 dark:text-slate-600">
                        —
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-3 text-xs text-slate-500 dark:text-slate-400 whitespace-nowrap">
                    {formatDate(user.lastLogin)}
                  </td>
                  <td className="px-6 py-3 text-xs font-mono text-slate-600 dark:text-slate-300">
                    {user.requestCount.toLocaleString()}
                  </td>
                  <td className="px-6 py-3 text-xs font-mono text-slate-600 dark:text-slate-300">
                    {formatTokens(user.tokenUsage)}
                  </td>
                  <td className="px-6 py-3 relative">
                    <button
                      onClick={() =>
                        setMenuOpenId(
                          menuOpenId === user.id ? null : user.id
                        )
                      }
                      className="p-1 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
                    >
                      <MoreVertical className="w-4 h-4" />
                    </button>
                    {menuOpenId === user.id && (
                      <div className="absolute right-6 top-full z-20 w-36 bg-white dark:bg-slate-700 rounded-lg shadow-lg border border-slate-200 dark:border-slate-600 py-1">
                        <button
                          onClick={() => {
                            setDeleteUser(user)
                            setMenuOpenId(null)
                          }}
                          className="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                          Delete User
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Mobile cards */}
        <div className="sm:hidden divide-y divide-slate-100 dark:divide-slate-700/50">
          {filtered.map((user) => (
            <div key={user.id} className="px-4 py-4">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded-full bg-slate-200 dark:bg-slate-600 flex items-center justify-center text-xs font-bold text-slate-600 dark:text-slate-300">
                    {user.name
                      .split(' ')
                      .map((n) => n[0])
                      .join('')
                      .slice(0, 2)
                      .toUpperCase()}
                  </div>
                  <div>
                    <p className="font-medium text-sm text-slate-800 dark:text-slate-200">
                      {user.name}
                    </p>
                    <p className="text-xs text-slate-400 dark:text-slate-500">
                      {user.email}
                    </p>
                  </div>
                </div>
                <span
                  className={`text-[10px] font-medium rounded-full px-2 py-0.5 ${
                    user.role === 'admin'
                      ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300'
                      : 'bg-slate-100 dark:bg-slate-700/50 text-slate-600 dark:text-slate-400'
                  }`}
                >
                  {user.role === 'admin' ? 'Admin' : 'User'}
                </span>
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-slate-400">
                <span className="font-mono uppercase">{user.authSource}</span>
                <span>{formatDate(user.lastLogin)}</span>
                <span>{user.requestCount.toLocaleString()} req</span>
                <span>{formatTokens(user.tokenUsage)} tok</span>
              </div>
              {user.groups.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1">
                  {user.groups.map((g) => (
                    <span
                      key={g}
                      className="text-[10px] font-medium px-1.5 py-0.5 rounded bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-400"
                    >
                      {g}
                    </span>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Delete User Dialog */}
      {deleteUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-black/50"
            onClick={() => {
              setDeleteUser(null)
              setDeleteReason('')
            }}
          />
          <div className="relative bg-white dark:bg-slate-800 rounded-xl shadow-xl max-w-md w-full border border-slate-200 dark:border-slate-700">
            <div className="px-6 py-4 border-b border-slate-100 dark:border-slate-700/50">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-red-50 dark:bg-red-900/30 flex items-center justify-center">
                  <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400" />
                </div>
                <div>
                  <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
                    Delete User — GDPR Data Purge
                  </h3>
                  <p className="text-xs text-slate-500 dark:text-slate-400">
                    This action cannot be undone
                  </p>
                </div>
              </div>
            </div>

            <div className="px-6 py-4 space-y-4">
              <p className="text-sm text-slate-600 dark:text-slate-300">
                Are you sure you want to delete{' '}
                <strong>{deleteUser.name}</strong> ({deleteUser.email})?
              </p>

              <div className="bg-red-50 dark:bg-red-900/20 rounded-lg p-3 text-sm text-red-700 dark:text-red-300 space-y-1">
                <p className="font-medium">The following data will be permanently deleted:</p>
                <ul className="list-disc list-inside text-xs space-y-0.5">
                  <li>All routing decision logs ({deleteUser.requestCount.toLocaleString()} records)</li>
                  <li>Token usage history ({formatTokens(deleteUser.tokenUsage)} tokens)</li>
                  <li>Benchmark results and metrics</li>
                  <li>Session and authentication data</li>
                </ul>
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1.5">
                  Reason for deletion <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={deleteReason}
                  onChange={(e) => setDeleteReason(e.target.value)}
                  rows={3}
                  placeholder="e.g., GDPR deletion request via HR ticket #1234"
                  className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-red-500/30 focus:border-red-500 resize-none"
                />
              </div>
            </div>

            <div className="px-6 py-3 border-t border-slate-100 dark:border-slate-700/50 flex justify-end gap-2">
              <button
                onClick={() => {
                  setDeleteUser(null)
                  setDeleteReason('')
                }}
                className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-600 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={!deleteReason.trim()}
                className="px-4 py-2 text-sm font-medium rounded-lg bg-red-600 text-white hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Delete & Purge Data
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Add Local User Dialog */}
      {showAddUser && (
        <AddLocalUserDialog
          onAdd={onAddLocalUser}
          onClose={() => setShowAddUser(false)}
        />
      )}
    </>
  )
}

// --- Add Local User Dialog ---

function AddLocalUserDialog({
  onAdd,
  onClose,
}: {
  onAdd?: (name: string, email: string, role: UserRole) => void
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<UserRole>('user')

  function handleSubmit() {
    if (name.trim() && email.trim()) {
      onAdd?.(name.trim(), email.trim(), role)
      onClose()
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-white dark:bg-slate-800 rounded-xl shadow-xl max-w-sm w-full border border-slate-200 dark:border-slate-700">
        <div className="px-6 py-4 flex items-center justify-between border-b border-slate-100 dark:border-slate-700/50">
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
            Add Local User
          </h3>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        <div className="px-6 py-4 space-y-3">
          <div>
            <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
              Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Full name"
              className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-500"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
              Email
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="user@example.com"
              className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-500"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">
              Role
            </label>
            <select
              value={role}
              onChange={(e) => setRole(e.target.value as UserRole)}
              className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/30"
            >
              <option value="user">User</option>
              <option value="admin">Admin</option>
            </select>
          </div>
        </div>
        <div className="px-6 py-3 border-t border-slate-100 dark:border-slate-700/50 flex justify-end gap-2">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-600 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={!name.trim() || !email.trim()}
            className="px-4 py-2 text-sm font-medium rounded-lg bg-indigo-600 dark:bg-indigo-500 text-white hover:bg-indigo-700 dark:hover:bg-indigo-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            Add User
          </button>
        </div>
      </div>
    </div>
  )
}
