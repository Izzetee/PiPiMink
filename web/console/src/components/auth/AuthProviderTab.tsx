import { useState } from 'react'
import type { AuthProvider } from '@/types/auth'
import {
  CheckCircle2,
  XCircle,
  CircleDashed,
  Wifi,
  Eye,
  EyeOff,
  Loader2,
  Clock,
} from 'lucide-react'

interface AuthProviderTabProps {
  providers: AuthProvider[]
  onSave?: (provider: AuthProvider) => void
  onTest?: (providerId: string) => void
}

export function AuthProviderTab({
  providers,
  onSave,
  onTest,
}: AuthProviderTabProps) {
  return (
    <div className="space-y-6">
      {providers.map((provider) => (
        <ProviderCard
          key={provider.id}
          provider={provider}
          onSave={onSave}
          onTest={onTest}
        />
      ))}
    </div>
  )
}

// --- Provider Card ---

interface ProviderCardProps {
  provider: AuthProvider
  onSave?: (provider: AuthProvider) => void
  onTest?: (providerId: string) => void
}

function ProviderCard({ provider, onSave, onTest }: ProviderCardProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [isTesting, setIsTesting] = useState(false)
  const isLdap = provider.type === 'ldap'
  const isComingSoon = isLdap

  const statusConfig = {
    connected: {
      icon: CheckCircle2,
      label: 'Connected',
      color:
        'text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-900/30',
    },
    disconnected: {
      icon: XCircle,
      label: 'Disconnected',
      color:
        'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/30',
    },
    not_configured: {
      icon: CircleDashed,
      label: 'Not Configured',
      color:
        'text-slate-500 dark:text-slate-400 bg-slate-100 dark:bg-slate-700/50',
    },
  }

  const status = statusConfig[provider.status]
  const StatusIcon = status.icon

  function handleTest() {
    setIsTesting(true)
    onTest?.(provider.id)
    setTimeout(() => setIsTesting(false), 2000)
  }

  return (
    <div
      className={`bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm overflow-hidden ${
        isComingSoon ? 'opacity-75' : ''
      }`}
    >
      {/* Card header */}
      <div className="px-4 sm:px-6 py-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-100 dark:border-slate-700/50">
        <div className="flex items-center gap-3">
          <div
            className={`w-10 h-10 rounded-lg flex items-center justify-center ${
              provider.type === 'oauth'
                ? 'bg-indigo-50 dark:bg-indigo-900/30'
                : 'bg-slate-100 dark:bg-slate-700/50'
            }`}
          >
            <Wifi
              className={`w-5 h-5 ${
                provider.type === 'oauth'
                  ? 'text-indigo-600 dark:text-indigo-400'
                  : 'text-slate-500 dark:text-slate-400'
              }`}
              strokeWidth={1.5}
            />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-200">
                {provider.name}
              </h3>
              <span className="text-xs font-mono uppercase text-slate-400 dark:text-slate-500">
                {provider.type}
              </span>
              {isComingSoon && (
                <span className="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-400">
                  Coming Soon
                </span>
              )}
            </div>
            {provider.lastVerified && (
              <p className="text-xs text-slate-400 dark:text-slate-500 flex items-center gap-1 mt-0.5">
                <Clock className="w-3 h-3" />
                Last verified{' '}
                {new Date(provider.lastVerified).toLocaleDateString('en-US', {
                  month: 'short',
                  day: 'numeric',
                  year: 'numeric',
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </p>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span
            className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${status.color}`}
          >
            <StatusIcon className="w-3.5 h-3.5" />
            {status.label}
          </span>
        </div>
      </div>

      {/* Card body */}
      <div className="px-4 sm:px-6 py-4">
        {provider.type === 'oauth' ? (
          <OAuthFields
            provider={provider}
            isEditing={isEditing}
            disabled={isComingSoon}
          />
        ) : (
          <LdapFields
            provider={provider}
            isEditing={isEditing}
            disabled={isComingSoon}
          />
        )}
      </div>

      {/* Card footer */}
      <div className="px-4 sm:px-6 py-3 bg-slate-50 dark:bg-slate-800/50 border-t border-slate-100 dark:border-slate-700/50 flex flex-col sm:flex-row items-stretch sm:items-center justify-end gap-2">
        {!isComingSoon && (
          <>
            <button
              onClick={handleTest}
              disabled={isTesting || provider.status === 'not_configured'}
              className="inline-flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-600 text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {isTesting ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Wifi className="w-4 h-4" />
              )}
              {isTesting ? 'Testing...' : 'Test Connection'}
            </button>
            <button
              onClick={() => {
                if (isEditing) {
                  onSave?.(provider)
                }
                setIsEditing(!isEditing)
              }}
              className="inline-flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-indigo-600 dark:bg-indigo-500 text-white hover:bg-indigo-700 dark:hover:bg-indigo-600 transition-colors"
            >
              {isEditing ? 'Save Changes' : 'Edit Configuration'}
            </button>
          </>
        )}
      </div>
    </div>
  )
}

// --- OAuth Fields ---

function OAuthFields({
  provider,
  isEditing,
  disabled,
}: {
  provider: AuthProvider
  isEditing: boolean
  disabled: boolean
}) {
  const [showSecret, setShowSecret] = useState(false)

  const fields = [
    { label: 'Issuer URL', value: provider.issuerUrl, mono: true },
    { label: 'Client ID', value: provider.clientId, mono: true },
    {
      label: 'Client Secret',
      value: provider.clientSecret,
      mono: true,
      secret: true,
    },
    { label: 'Scopes', value: provider.scopes, mono: true },
    { label: 'Redirect URI', value: provider.redirectUri, mono: true },
    {
      label: 'Auto-Provision Users',
      value: provider.autoProvision ? 'Enabled' : 'Disabled',
      toggle: true,
    },
  ]

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-3">
      {fields.map((field) => (
        <div key={field.label} className={field.label === 'Issuer URL' || field.label === 'Redirect URI' ? 'sm:col-span-2' : ''}>
          <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
            {field.label}
          </label>
          {isEditing && !disabled ? (
            field.toggle ? (
              <div className="flex items-center gap-2">
                <button
                  className={`relative w-9 h-5 rounded-full transition-colors ${
                    provider.autoProvision
                      ? 'bg-indigo-600 dark:bg-indigo-500'
                      : 'bg-slate-300 dark:bg-slate-600'
                  }`}
                >
                  <span
                    className={`absolute top-0.5 left-0.5 w-4 h-4 rounded-full bg-white shadow transition-transform ${
                      provider.autoProvision ? 'translate-x-4' : ''
                    }`}
                  />
                </button>
                <span className="text-sm text-slate-600 dark:text-slate-300">
                  {provider.autoProvision ? 'Enabled' : 'Disabled'}
                </span>
              </div>
            ) : (
              <div className="relative">
                <input
                  type={field.secret && !showSecret ? 'password' : 'text'}
                  defaultValue={field.value ?? ''}
                  className="w-full px-3 py-2 text-sm font-mono rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-500"
                />
                {field.secret && (
                  <button
                    onClick={() => setShowSecret(!showSecret)}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                  >
                    {showSecret ? (
                      <EyeOff className="w-4 h-4" />
                    ) : (
                      <Eye className="w-4 h-4" />
                    )}
                  </button>
                )}
              </div>
            )
          ) : (
            <div
              className={`text-sm ${
                field.mono ? 'font-mono' : ''
              } text-slate-700 dark:text-slate-300 ${
                disabled ? 'text-slate-400 dark:text-slate-500' : ''
              }`}
            >
              {field.toggle ? (
                <span
                  className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${
                    provider.autoProvision
                      ? 'bg-emerald-50 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-400'
                      : 'bg-slate-100 dark:bg-slate-700/50 text-slate-500 dark:text-slate-400'
                  }`}
                >
                  {field.value}
                </span>
              ) : field.secret ? (
                <span className="text-slate-400 dark:text-slate-500">
                  {field.value}
                </span>
              ) : (
                <span className="break-all">
                  {field.value || (
                    <span className="text-slate-300 dark:text-slate-600 italic">
                      Not set
                    </span>
                  )}
                </span>
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

// --- LDAP Fields ---

function LdapFields({
  provider,
  isEditing,
  disabled,
}: {
  provider: AuthProvider
  isEditing: boolean
  disabled: boolean
}) {
  const fields = [
    { label: 'Server URL', value: provider.serverUrl },
    { label: 'Bind DN', value: provider.bindDn },
    { label: 'Base DN', value: provider.baseDn },
    { label: 'Search Filter', value: provider.searchFilter },
    { label: 'Group Mapping', value: provider.groupMapping },
  ]

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-3">
      {fields.map((field) => (
        <div key={field.label} className={field.label === 'Server URL' ? 'sm:col-span-2' : ''}>
          <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
            {field.label}
          </label>
          {isEditing && !disabled ? (
            <input
              type="text"
              defaultValue={field.value ?? ''}
              disabled={disabled}
              placeholder={disabled ? 'Coming soon...' : `Enter ${field.label.toLowerCase()}`}
              className="w-full px-3 py-2 text-sm font-mono rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-800 dark:text-slate-200 disabled:bg-slate-100 dark:disabled:bg-slate-800 disabled:text-slate-400 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-500"
            />
          ) : (
            <span
              className={`text-sm font-mono ${
                disabled
                  ? 'text-slate-300 dark:text-slate-600 italic'
                  : 'text-slate-700 dark:text-slate-300'
              }`}
            >
              {field.value || (
                <span className="text-slate-300 dark:text-slate-600 italic">
                  Not configured
                </span>
              )}
            </span>
          )}
        </div>
      ))}
    </div>
  )
}
