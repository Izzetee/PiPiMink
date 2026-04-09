import { apiGet, apiPost, getApiKey, setApiKey } from './client'
import { vi, describe, it, expect, beforeEach } from 'vitest'

describe('client', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  it('getApiKey/setApiKey use localStorage', () => {
    expect(getApiKey()).toBe('')
    setApiKey('my-key')
    expect(getApiKey()).toBe('my-key')
  })

  it('apiGet sends GET with headers', async () => {
    setApiKey('test-key')
    const mockFetch = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), { status: 200 }),
    )

    await apiGet('/test')

    expect(mockFetch).toHaveBeenCalledWith('/test', {
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': 'test-key',
      },
    })
  })

  it('apiPost sends body as JSON', async () => {
    const mockFetch = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), { status: 200 }),
    )

    await apiPost('/test', { foo: 'bar' })

    expect(mockFetch).toHaveBeenCalledWith('/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ foo: 'bar' }),
    })
  })

  it('401 triggers redirect', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response('Unauthorized', { status: 401 }),
    )

    const originalLocation = window.location
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, href: '/console/', pathname: '/console/' },
      writable: true,
    })

    await expect(apiGet('/test')).rejects.toThrow('401')
    expect(window.location.href).toBe('/auth/login')

    Object.defineProperty(window, 'location', { value: originalLocation, writable: true })
  })
})
