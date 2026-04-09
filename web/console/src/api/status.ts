export interface AdminStatus {
  adminKeyConfigured: boolean
  providersConfigured: boolean
  providerCount: number
  modelCount: number
  oauthEnabled: boolean
}

export async function fetchAdminStatus(): Promise<AdminStatus> {
  const res = await fetch('/admin/status')
  if (!res.ok) throw new Error(`GET /admin/status: ${res.status}`)
  return res.json() as Promise<AdminStatus>
}
