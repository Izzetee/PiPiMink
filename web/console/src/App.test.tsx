import { render, screen } from '@testing-library/react'
import { App } from './App'
import { vi, describe, it, expect, afterEach } from 'vitest'

// Mock all hooks and heavy page components
vi.mock('@/hooks/useAuth', () => ({
  useAuth: vi.fn(),
}))
vi.mock('@/hooks/useSetupStatus', () => ({
  useSetupStatus: vi.fn(),
}))
vi.mock('@/hooks/useTheme', () => ({
  useTheme: () => ({ isDark: false, toggle: vi.fn() }),
}))
vi.mock('@/pages/SetupPage', () => ({
  SetupPage: (_props: { onComplete: () => void }) => <div data-testid="setup-page">Setup</div>,
}))
vi.mock('@/pages/LoginPage', () => ({
  LoginPage: () => <div data-testid="login-page">Login</div>,
}))
vi.mock('@/pages/ModelsPage', () => ({
  ModelsPage: () => <div data-testid="models-page">Models</div>,
}))
// Stub remaining pages to avoid import overhead
vi.mock('@/pages/ProvidersPage', () => ({ ProvidersPage: () => null }))
vi.mock('@/pages/ConfigPage', () => ({ ConfigPage: () => null }))
vi.mock('@/pages/SettingsPage', () => ({ SettingsPage: () => null }))
vi.mock('@/pages/AnalyticsPage', () => ({ AnalyticsPage: () => null }))
vi.mock('@/pages/UsersPage', () => ({ UsersPage: () => null }))

import { useAuth } from '@/hooks/useAuth'
import { useSetupStatus } from '@/hooks/useSetupStatus'

const mockUseAuth = vi.mocked(useAuth)
const mockUseSetupStatus = vi.mocked(useSetupStatus)

describe('App', () => {
  afterEach(() => vi.restoreAllMocks())

  it('shows loading spinner initially', () => {
    mockUseAuth.mockReturnValue({
      user: null, isAuthenticated: false, oauthEnabled: false,
      isLoading: true, logout: vi.fn(),
    })
    mockUseSetupStatus.mockReturnValue({
      needsSetup: false, status: null, isLoading: true, refetch: vi.fn(),
    })

    render(<App />)
    // Loader2 renders an SVG with the animate-spin class
    expect(document.querySelector('.animate-spin')).toBeTruthy()
  })

  it('shows SetupPage when needsSetup', () => {
    mockUseAuth.mockReturnValue({
      user: null, isAuthenticated: false, oauthEnabled: false,
      isLoading: false, logout: vi.fn(),
    })
    mockUseSetupStatus.mockReturnValue({
      needsSetup: true,
      status: { adminKeyConfigured: false, providersConfigured: false, providerCount: 0, modelCount: 0, oauthEnabled: false },
      isLoading: false, refetch: vi.fn(),
    })

    render(<App />)
    expect(screen.getByTestId('setup-page')).toBeTruthy()
  })

  it('shows LoginPage when OAuth + unauthenticated', () => {
    mockUseAuth.mockReturnValue({
      user: null, isAuthenticated: false, oauthEnabled: true,
      isLoading: false, logout: vi.fn(),
    })
    mockUseSetupStatus.mockReturnValue({
      needsSetup: false, status: null, isLoading: false, refetch: vi.fn(),
    })

    render(<App />)
    expect(screen.getByTestId('login-page')).toBeTruthy()
  })

  it('shows AppShell when authenticated', () => {
    mockUseAuth.mockReturnValue({
      user: { id: '1', name: 'Admin', email: 'a@b.com', role: 'admin' },
      isAuthenticated: true, oauthEnabled: true,
      isLoading: false, logout: vi.fn(),
    })
    mockUseSetupStatus.mockReturnValue({
      needsSetup: false, status: null, isLoading: false, refetch: vi.fn(),
    })

    render(<App />)
    // Should render the models page (default route)
    expect(screen.getByTestId('models-page')).toBeTruthy()
  })
})
