const API_KEY_STORAGE_KEY = 'pipimink-api-key'

export function getApiKey(): string {
  return localStorage.getItem(API_KEY_STORAGE_KEY) ?? ''
}

export function setApiKey(key: string): void {
  localStorage.setItem(API_KEY_STORAGE_KEY, key)
}

function headers(): HeadersInit {
  const h: HeadersInit = { 'Content-Type': 'application/json' }
  const key = getApiKey()
  if (key) h['X-API-Key'] = key
  return h
}

function handleUnauthorized(res: Response): void {
  if (res.status === 401 && !window.location.pathname.startsWith('/auth/')) {
    window.location.href = '/auth/login'
  }
}

export async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: headers() })
  if (!res.ok) { handleUnauthorized(res); throw new Error(`GET ${path}: ${res.status} ${res.statusText}`) }
  return res.json() as Promise<T>
}

export async function apiPost<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method: 'POST',
    headers: headers(),
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) { handleUnauthorized(res); throw new Error(`POST ${path}: ${res.status} ${res.statusText}`) }
  return res.json() as Promise<T>
}

export async function apiPatch<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(path, {
    method: 'PATCH',
    headers: headers(),
    body: JSON.stringify(body),
  })
  if (!res.ok) { handleUnauthorized(res); throw new Error(`PATCH ${path}: ${res.status} ${res.statusText}`) }
  return res.json() as Promise<T>
}

export async function apiPut<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(path, {
    method: 'PUT',
    headers: headers(),
    body: JSON.stringify(body),
  })
  if (!res.ok) { handleUnauthorized(res); throw new Error(`PUT ${path}: ${res.status} ${res.statusText}`) }
  return res.json() as Promise<T>
}

export async function apiDelete<T = unknown>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method: 'DELETE',
    headers: headers(),
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) { handleUnauthorized(res); throw new Error(`DELETE ${path}: ${res.status} ${res.statusText}`) }
  return res.json() as Promise<T>
}
