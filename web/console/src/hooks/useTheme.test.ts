import { renderHook, act } from '@testing-library/react'
import { useTheme } from './useTheme'
import { vi, describe, it, expect, beforeEach } from 'vitest'

describe('useTheme', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.classList.remove('dark')
    // jsdom doesn't implement matchMedia — provide a stub
    window.matchMedia = vi.fn().mockReturnValue({ matches: false } as MediaQueryList)
  })

  it('reads dark from localStorage', () => {
    localStorage.setItem('pipimink-theme', 'dark')
    const { result } = renderHook(() => useTheme())
    expect(result.current.isDark).toBe(true)
  })

  it('reads light from localStorage', () => {
    localStorage.setItem('pipimink-theme', 'light')
    const { result } = renderHook(() => useTheme())
    expect(result.current.isDark).toBe(false)
  })

  it('toggle flips state', () => {
    localStorage.setItem('pipimink-theme', 'light')
    const { result } = renderHook(() => useTheme())
    expect(result.current.isDark).toBe(false)

    act(() => result.current.toggle())
    expect(result.current.isDark).toBe(true)

    act(() => result.current.toggle())
    expect(result.current.isDark).toBe(false)
  })

  it('toggle updates localStorage', () => {
    localStorage.setItem('pipimink-theme', 'light')
    const { result } = renderHook(() => useTheme())

    act(() => result.current.toggle())
    expect(localStorage.getItem('pipimink-theme')).toBe('dark')
  })
})
