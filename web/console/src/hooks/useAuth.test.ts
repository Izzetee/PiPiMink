import { renderHook, waitFor, act } from '@testing-library/react'
import { useAuth } from './useAuth'
import { vi, describe, it, expect, afterEach } from 'vitest'

vi.mock('@/api/auth', () => ({
  fetchAuthMe: vi.fn(),
  logout: vi.fn(),
}))

import { fetchAuthMe, logout } from '@/api/auth'
const mockFetchMe = vi.mocked(fetchAuthMe)
const mockLogout = vi.mocked(logout)

describe('useAuth', () => {
  afterEach(() => vi.restoreAllMocks())

  it('starts in loading state', () => {
    mockFetchMe.mockReturnValue(new Promise(() => {}))
    const { result } = renderHook(() => useAuth())
    expect(result.current.isLoading).toBe(true)
  })

  it('authenticated when API returns user', async () => {
    mockFetchMe.mockResolvedValue({
      authenticated: true,
      oauthEnabled: true,
      user: { id: '1', name: 'Jane', email: 'jane@co.com', role: 'admin' as const },
    })

    const { result } = renderHook(() => useAuth())
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.isAuthenticated).toBe(true)
    expect(result.current.user?.name).toBe('Jane')
    expect(result.current.oauthEnabled).toBe(true)
  })

  it('unauthenticated on error', async () => {
    mockFetchMe.mockRejectedValue(new Error('fail'))

    const { result } = renderHook(() => useAuth())
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.user).toBeNull()
  })

  it('logout clears state', async () => {
    mockFetchMe.mockResolvedValue({ authenticated: true, oauthEnabled: false })
    mockLogout.mockResolvedValue({ ok: true })

    const { result } = renderHook(() => useAuth())
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    await act(async () => { await result.current.logout() })

    expect(result.current.user).toBeNull()
    expect(result.current.isAuthenticated).toBe(false)
  })

  it('logout redirects when OAuth enabled', async () => {
    mockFetchMe.mockResolvedValue({ authenticated: true, oauthEnabled: true })
    mockLogout.mockResolvedValue({ ok: true })

    // Mock window.location
    const originalLocation = window.location
    Object.defineProperty(window, 'location', {
      value: { ...originalLocation, href: '' },
      writable: true,
    })

    const { result } = renderHook(() => useAuth())
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    await act(async () => { await result.current.logout() })

    expect(window.location.href).toBe('/auth/login')

    Object.defineProperty(window, 'location', { value: originalLocation, writable: true })
  })
})
