import { renderHook, waitFor } from '@testing-library/react'
import { useSetupStatus } from './useSetupStatus'
import { vi, describe, it, expect, afterEach } from 'vitest'

vi.mock('@/api/status', () => ({
  fetchAdminStatus: vi.fn(),
}))

import { fetchAdminStatus } from '@/api/status'
const mockFetch = vi.mocked(fetchAdminStatus)

describe('useSetupStatus', () => {
  afterEach(() => vi.restoreAllMocks())

  it('needsSetup true when no key and no models', async () => {
    mockFetch.mockResolvedValue({
      adminKeyConfigured: false,
      providersConfigured: false,
      providerCount: 0,
      modelCount: 0,
      oauthEnabled: false,
    })

    const { result } = renderHook(() => useSetupStatus())
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.needsSetup).toBe(true)
  })

  it('needsSetup false when key configured', async () => {
    mockFetch.mockResolvedValue({
      adminKeyConfigured: true,
      providersConfigured: false,
      providerCount: 0,
      modelCount: 0,
      oauthEnabled: false,
    })

    const { result } = renderHook(() => useSetupStatus())
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.needsSetup).toBe(false)
  })

  it('needsSetup false when models exist', async () => {
    mockFetch.mockResolvedValue({
      adminKeyConfigured: false,
      providersConfigured: true,
      providerCount: 1,
      modelCount: 5,
      oauthEnabled: false,
    })

    const { result } = renderHook(() => useSetupStatus())
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.needsSetup).toBe(false)
  })

  it('isLoading true during fetch', () => {
    mockFetch.mockReturnValue(new Promise(() => {})) // never resolves

    const { result } = renderHook(() => useSetupStatus())
    expect(result.current.isLoading).toBe(true)
  })

  it('handles fetch error gracefully', async () => {
    mockFetch.mockRejectedValue(new Error('network'))

    const { result } = renderHook(() => useSetupStatus())
    await waitFor(() => expect(result.current.isLoading).toBe(false))

    expect(result.current.needsSetup).toBe(false)
    expect(result.current.status).toBeNull()
  })
})
