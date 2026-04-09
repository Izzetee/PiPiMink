import { BrowserRouter, Routes, Route, Navigate, useLocation, useNavigate } from 'react-router'
import { Cpu, Network, Wrench, Settings, BarChart3, Users, Loader2 } from 'lucide-react'
import { AppShell, type NavItem } from '@/components/shell'
import { useTheme } from '@/hooks/useTheme'
import { useAuth } from '@/hooks/useAuth'
import { useSetupStatus } from '@/hooks/useSetupStatus'
import { ModelsPage } from '@/pages/ModelsPage'
import { ProvidersPage } from '@/pages/ProvidersPage'
import { ConfigPage } from '@/pages/ConfigPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { AnalyticsPage } from '@/pages/AnalyticsPage'
import { UsersPage } from '@/pages/UsersPage'
import { LoginPage } from '@/pages/LoginPage'
import { SetupPage } from '@/pages/SetupPage'

const NAV_ITEMS: Omit<NavItem, 'isActive'>[] = [
  { label: 'Models', href: '/console/models', icon: Cpu },
  { label: 'Providers', href: '/console/providers', icon: Network },
  { label: 'Config', href: '/console/config', icon: Wrench },
  { label: 'Settings', href: '/console/settings', icon: Settings },
  { label: 'Analytics', href: '/console/analytics', icon: BarChart3 },
  { label: 'Users', href: '/console/users', icon: Users },
]

function AppLayout() {
  const location = useLocation()
  const navigate = useNavigate()
  const { isDark, toggle } = useTheme()
  const { user, isAuthenticated, oauthEnabled, isLoading, logout } = useAuth()
  const { needsSetup, isLoading: setupLoading, refetch: refetchSetup } = useSetupStatus()

  if (isLoading || setupLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-900">
        <Loader2 className="w-8 h-8 text-indigo-500 animate-spin" />
      </div>
    )
  }

  // Show setup wizard on first run (no admin key, no models)
  if (needsSetup) {
    return <SetupPage onComplete={refetchSetup} />
  }

  // If OAuth is enabled and user is not authenticated, show login page
  if (oauthEnabled && !isAuthenticated) {
    return <LoginPage oauthEnabled={oauthEnabled} />
  }

  const shellRole = user?.role === 'admin' ? 'admin' as const : 'viewer' as const
  const currentUser = user
    ? { name: user.name, role: shellRole }
    : { name: 'Admin', role: 'admin' as const }

  const navigationItems: NavItem[] = NAV_ITEMS.map((item) => ({
    ...item,
    isActive: location.pathname.startsWith(item.href),
  }))

  return (
    <AppShell
      navigationItems={navigationItems}
      user={currentUser}
      onNavigate={(href) => navigate(href)}
      onLogout={logout}
      onToggleTheme={toggle}
      isDark={isDark}
    >
      <Routes>
        <Route path="models" element={<ModelsPage />} />
        <Route path="providers" element={<ProvidersPage />} />
        <Route path="config" element={<ConfigPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="analytics" element={<AnalyticsPage />} />
        <Route path="users" element={<UsersPage />} />
        <Route path="*" element={<Navigate to="/console/models" replace />} />
      </Routes>
    </AppShell>
  )
}

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/console/*" element={<AppLayout />} />
        <Route path="*" element={<Navigate to="/console/models" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
