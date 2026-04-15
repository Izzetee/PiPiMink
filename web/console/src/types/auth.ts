// Auth & Users — TypeScript Interfaces

export type AuthProviderType = 'oauth' | 'ldap' | 'local'
export type AuthProviderStatus = 'connected' | 'disconnected' | 'not_configured'
export type UserRole = 'admin' | 'user'
export type AuthSource = 'oauth' | 'ldap' | 'local'

export type AuditAction =
  | 'provider_configured'
  | 'provider_verified'
  | 'user_created'
  | 'user_deleted'
  | 'role_changed'
  | 'group_role_changed'
  | 'group_routing_updated'

export type RoutingRuleType = 'allow_all' | 'allow_providers' | 'deny_providers' | 'allow_models' | 'deny_models'

// --- Data Interfaces ---

export interface AuthProvider {
  id: string
  type: AuthProviderType
  name: string
  status: AuthProviderStatus
  // OAuth fields
  issuerUrl?: string
  clientId?: string
  clientSecret?: string
  scopes?: string
  redirectUri?: string
  autoProvision?: boolean
  // LDAP fields
  serverUrl?: string
  bindDn?: string
  baseDn?: string
  searchFilter?: string
  groupMapping?: string
  // Common
  lastVerified: string | null
  createdAt: string | null
}

export interface User {
  id: string
  name: string
  email: string
  role: UserRole
  authSource: AuthSource
  authProviderName: string | null
  groups: string[]
  lastLogin: string
  createdAt: string
  requestCount: number
  tokenUsage: number
  avatarUrl: string | null
}

export interface RoutingRule {
  id: string
  type: RoutingRuleType
  providers?: string[]
  models?: string[]
  description: string
}

export interface Group {
  id: string
  name: string
  source: string
  memberCount: number
  role: UserRole
  routingRules: RoutingRule[]
  syncedAt: string
}

export interface AuditEntry {
  id: string
  timestamp: string
  actor: string
  action: AuditAction
  target: string
  details: string
  reason: string | null
}

// --- Auth Me response ---

export interface AuthMeResponse {
  authenticated: boolean
  oauthEnabled: boolean
  user?: {
    id: string
    name: string
    email: string
    role: UserRole
    groups?: string[]
  }
}

// --- Props Interface ---

export interface AuthAndUsersProps {
  authProviders: AuthProvider[]
  users: User[]
  groups: Group[]
  auditLog: AuditEntry[]

  onSaveProvider?: (provider: AuthProvider) => void
  onTestProvider?: (providerId: string) => void
  onChangeUserRole?: (userId: string, role: UserRole) => void
  onDeleteUser?: (userId: string, reason: string) => void
  onAddLocalUser?: (name: string, email: string, role: UserRole) => void
  onChangeGroupRole?: (groupId: string, role: UserRole) => void
  onAddRoutingRule?: (groupId: string, rule: Omit<RoutingRule, 'id'>) => void
  onRemoveRoutingRule?: (groupId: string, ruleId: string) => void
}
