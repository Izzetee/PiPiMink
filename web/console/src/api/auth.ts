import { apiGet, apiPost, apiPut, apiDelete } from './client'
import type {
  AuthProvider,
  User,
  UserRole,
  Group,
  RoutingRule,
  AuditEntry,
  AuthMeResponse,
} from '@/types/auth'

// --- Auth flow ---

export function fetchAuthMe(): Promise<AuthMeResponse> {
  return apiGet<AuthMeResponse>('/auth/me')
}

export function logout(): Promise<{ ok: boolean }> {
  return apiPost<{ ok: boolean }>('/auth/logout')
}

// --- Auth providers ---

export function fetchAuthProviders(): Promise<AuthProvider[]> {
  return apiGet<AuthProvider[]>('/admin/auth/providers')
}

export function saveAuthProvider(provider: AuthProvider): Promise<AuthProvider> {
  return apiPut<AuthProvider>(`/admin/auth/providers/${provider.id}`, provider)
}

export function testAuthProvider(providerId: string): Promise<{ status: string; error?: string }> {
  return apiPost<{ status: string; error?: string }>(`/admin/auth/providers/${providerId}/test`)
}

// --- Users ---

export function fetchUsers(): Promise<User[]> {
  return apiGet<User[]>('/admin/auth/users')
}

export function addLocalUser(name: string, email: string, role: UserRole): Promise<User> {
  return apiPost<User>('/admin/auth/users', { name, email, role })
}

export function changeUserRole(userId: string, role: UserRole): Promise<{ status: string }> {
  return apiPut<{ status: string }>(`/admin/auth/users/${userId}/role`, { role })
}

export function deleteUser(userId: string, reason: string): Promise<{ status: string }> {
  return apiDelete<{ status: string }>(`/admin/auth/users/${userId}`, { reason })
}

// --- Groups ---

export function fetchGroups(): Promise<Group[]> {
  return apiGet<Group[]>('/admin/auth/groups')
}

export function changeGroupRole(groupId: string, role: UserRole): Promise<{ status: string }> {
  return apiPut<{ status: string }>(`/admin/auth/groups/${groupId}/role`, { role })
}

export function addRoutingRule(
  groupId: string,
  rule: Omit<RoutingRule, 'id'>,
): Promise<RoutingRule> {
  return apiPost<RoutingRule>(`/admin/auth/groups/${groupId}/rules`, rule)
}

export function removeRoutingRule(groupId: string, ruleId: string): Promise<{ status: string }> {
  return apiDelete<{ status: string }>(`/admin/auth/groups/${groupId}/rules/${ruleId}`)
}

// --- Audit log ---

export function fetchAuditLog(): Promise<AuditEntry[]> {
  return apiGet<AuditEntry[]>('/admin/auth/audit-log')
}
