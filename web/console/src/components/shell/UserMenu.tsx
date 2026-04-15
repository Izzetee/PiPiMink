import { useState, useRef, useEffect } from 'react'
import { LogOut, Moon, Sun, ChevronUp } from 'lucide-react'

interface UserMenuProps {
  user?: {
    name: string
    avatarUrl?: string
    role?: 'admin' | 'operator' | 'viewer'
  }
  expanded: boolean
  onLogout?: () => void
  onToggleTheme?: () => void
  isDark?: boolean
}

const roleColors = {
  admin: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/50 dark:text-indigo-300',
  operator: 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300',
  viewer: 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400',
}

export function UserMenu({ user, expanded, onLogout, onToggleTheme, isDark }: UserMenuProps) {
  const [open, setOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  // Close on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) {
      document.addEventListener('mousedown', handleClick)
      return () => document.removeEventListener('mousedown', handleClick)
    }
  }, [open])

  if (!user) return null

  const initials = user.name
    .split(' ')
    .map((n) => n[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)

  return (
    <div ref={menuRef} className="relative px-2">
      <button
        onClick={() => setOpen(!open)}
        title={!expanded ? user.name : undefined}
        className={`
          w-full flex items-center gap-3 rounded-lg px-3 py-2.5
          text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800
          transition-all duration-200 outline-none
        `}
      >
        {/* Avatar */}
        {user.avatarUrl ? (
          <img
            src={user.avatarUrl}
            alt={user.name}
            className="w-7 h-7 rounded-full shrink-0 object-cover"
          />
        ) : (
          <span className="w-7 h-7 rounded-full shrink-0 bg-indigo-100 dark:bg-indigo-900/50 text-indigo-700 dark:text-indigo-300 text-xs font-semibold flex items-center justify-center">
            {initials}
          </span>
        )}

        <span
          className={`
            text-sm font-medium whitespace-nowrap transition-all duration-200 text-left truncate
            ${expanded ? 'opacity-100 w-auto flex-1' : 'opacity-0 w-0 overflow-hidden'}
          `}
        >
          {user.name}
        </span>

        {expanded && (
          <ChevronUp
            className={`w-4 h-4 shrink-0 text-slate-400 transition-transform duration-200 ${
              open ? 'rotate-0' : 'rotate-180'
            }`}
            strokeWidth={1.75}
          />
        )}
      </button>

      {/* Dropdown */}
      {open && (
        <div className="absolute bottom-full left-2 right-2 mb-1 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg py-1 z-50 animate-fade-in min-w-[180px]">
          {/* User info */}
          <div className="px-3 py-2 border-b border-slate-100 dark:border-slate-800">
            <p className="text-sm font-medium text-slate-900 dark:text-slate-100">{user.name}</p>
            {user.role && (
              <span
                className={`inline-block mt-1 text-xs font-medium px-2 py-0.5 rounded-full capitalize ${
                  roleColors[user.role]
                }`}
              >
                {user.role}
              </span>
            )}
          </div>

          {/* Theme toggle */}
          <button
            onClick={() => {
              onToggleTheme?.()
              setOpen(false)
            }}
            className="w-full flex items-center gap-2.5 px-3 py-2 text-sm text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
          >
            {isDark ? (
              <Sun className="w-4 h-4" strokeWidth={1.75} />
            ) : (
              <Moon className="w-4 h-4" strokeWidth={1.75} />
            )}
            {isDark ? 'Light mode' : 'Dark mode'}
          </button>

          {/* Logout */}
          <button
            onClick={() => {
              onLogout?.()
              setOpen(false)
            }}
            className="w-full flex items-center gap-2.5 px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
          >
            <LogOut className="w-4 h-4" strokeWidth={1.75} />
            Log out
          </button>
        </div>
      )}
    </div>
  )
}
