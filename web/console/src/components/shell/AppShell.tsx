import { useState } from 'react'
import { Menu, X } from 'lucide-react'
import { MainNav, type NavItem } from './MainNav'
import { UserMenu } from './UserMenu'

export interface AppShellUser {
  name: string
  email?: string
  role?: 'admin' | 'operator' | 'viewer'
  avatarUrl?: string
}

interface AppShellProps {
  children: React.ReactNode
  navigationItems: NavItem[]
  user?: AppShellUser
  onNavigate?: (href: string) => void
  onLogout?: () => void
  onToggleTheme?: () => void
  isDark?: boolean
}

export function AppShell({
  children,
  navigationItems,
  user,
  onNavigate,
  onLogout,
  onToggleTheme,
  isDark,
}: AppShellProps) {
  const [mobileOpen, setMobileOpen] = useState(false)
  const [sidebarHovered, setSidebarHovered] = useState(false)

  return (
    <div
      className="h-screen flex bg-slate-50 dark:bg-slate-900"
      style={{ fontFamily: "'Inter', system-ui, sans-serif" }}
    >
      {/* Desktop sidebar */}
      <aside
        className={`
          hidden md:flex flex-col h-full shrink-0
          bg-white dark:bg-slate-800
          border-r border-slate-200 dark:border-slate-700
          transition-all duration-200 ease-out
          ${sidebarHovered ? 'w-56' : 'w-16'}
        `}
        onMouseEnter={() => setSidebarHovered(true)}
        onMouseLeave={() => setSidebarHovered(false)}
      >
        {/* Logo */}
        <div className="flex items-center gap-2.5 px-4 py-4 border-b border-slate-100 dark:border-slate-700/50 min-h-[56px]">
          <img
            src="/console/pipimink_square_small_split.png"
            alt="PiPiMink"
            className="w-8 h-8 shrink-0"
          />
          <div
            className={`overflow-hidden transition-all duration-200 ${
              sidebarHovered ? 'opacity-100 w-auto' : 'opacity-0 w-0'
            }`}
          >
            <span className="font-semibold text-sm text-slate-800 dark:text-slate-200 whitespace-nowrap">
              PiPiMink
            </span>
            <span className="block text-[10px] text-slate-400 dark:text-slate-500 whitespace-nowrap leading-tight">
              Console
            </span>
          </div>
        </div>

        {/* Navigation */}
        <div className="flex-1 py-3 overflow-y-auto">
          <MainNav
            items={navigationItems}
            expanded={sidebarHovered}
            onNavigate={onNavigate}
          />
        </div>

        {/* User menu */}
        <div className="border-t border-slate-100 dark:border-slate-700/50 py-2">
          <UserMenu
            user={user}
            expanded={sidebarHovered}
            onLogout={onLogout}
            onToggleTheme={onToggleTheme}
            isDark={isDark}
          />
        </div>
      </aside>

      {/* Mobile overlay */}
      {mobileOpen && (
        <div
          className="fixed inset-0 bg-black/40 z-40 md:hidden"
          onClick={() => setMobileOpen(false)}
        />
      )}

      {/* Mobile drawer */}
      <aside
        className={`
          fixed inset-y-0 left-0 z-50 w-64 flex flex-col
          bg-white dark:bg-slate-800
          border-r border-slate-200 dark:border-slate-700
          transform transition-transform duration-200 ease-out md:hidden
          ${mobileOpen ? 'translate-x-0' : '-translate-x-full'}
        `}
      >
        {/* Logo */}
        <div className="flex items-center gap-2.5 px-4 py-4 border-b border-slate-100 dark:border-slate-700/50">
          <img
            src="/console/pipimink_square_small_split.png"
            alt="PiPiMink"
            className="w-8 h-8 shrink-0"
          />
          <div>
            <span className="font-semibold text-sm text-slate-800 dark:text-slate-200">
              PiPiMink
            </span>
            <span className="block text-[10px] text-slate-400 dark:text-slate-500 leading-tight">
              Console
            </span>
          </div>
        </div>

        {/* Navigation */}
        <div className="flex-1 py-3 overflow-y-auto">
          <MainNav
            items={navigationItems}
            expanded
            onNavigate={(href) => {
              setMobileOpen(false)
              onNavigate?.(href)
            }}
          />
        </div>

        {/* User menu */}
        <div className="border-t border-slate-100 dark:border-slate-700/50 py-2">
          <UserMenu
            user={user}
            expanded
            onLogout={onLogout}
            onToggleTheme={onToggleTheme}
            isDark={isDark}
          />
        </div>
      </aside>

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Mobile header */}
        <header className="md:hidden flex items-center gap-3 px-4 py-3 border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800">
          <button
            onClick={() => setMobileOpen(!mobileOpen)}
            className="p-1.5 rounded-md text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            {mobileOpen ? (
              <X className="w-5 h-5" strokeWidth={1.5} />
            ) : (
              <Menu className="w-5 h-5" strokeWidth={1.5} />
            )}
          </button>
          <div className="flex items-center gap-2">
            <img
              src="/console/pipimink_square_small_split.png"
              alt="PiPiMink"
              className="w-6 h-6"
            />
            <span className="font-semibold text-sm text-slate-800 dark:text-slate-200">
              PiPiMink
            </span>
          </div>
        </header>

        {/* Page content */}
        <main className="flex-1 overflow-auto">{children}</main>
      </div>
    </div>
  )
}
