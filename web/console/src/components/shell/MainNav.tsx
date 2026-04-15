import { type LucideIcon } from 'lucide-react'

export interface NavItem {
  label: string
  href: string
  icon: LucideIcon
  isActive?: boolean
  badge?: string | number
}

interface MainNavProps {
  items: NavItem[]
  expanded: boolean
  onNavigate?: (href: string) => void
}

export function MainNav({ items, expanded, onNavigate }: MainNavProps) {
  return (
    <nav className="flex flex-col gap-1 px-2">
      {items.map((item) => {
        const Icon = item.icon
        return (
          <button
            key={item.href}
            onClick={() => onNavigate?.(item.href)}
            title={!expanded ? item.label : undefined}
            className={`
              group relative flex items-center gap-3 rounded-lg px-3 py-2.5
              transition-all duration-200 outline-none
              ${item.isActive
                ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-950/50 dark:text-indigo-300'
                : 'text-slate-500 hover:bg-slate-100 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-100'
              }
            `}
          >
            {/* Active indicator bar */}
            {item.isActive && (
              <span className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 rounded-r-full bg-indigo-500 dark:bg-indigo-400" />
            )}

            <Icon
              className={`w-5 h-5 shrink-0 transition-colors duration-200 ${
                item.isActive
                  ? 'text-indigo-600 dark:text-indigo-400'
                  : 'text-slate-400 group-hover:text-slate-600 dark:text-slate-500 dark:group-hover:text-slate-300'
              }`}
              strokeWidth={1.75}
            />

            <span
              className={`
                text-sm font-medium whitespace-nowrap transition-all duration-200
                ${expanded ? 'opacity-100 w-auto' : 'opacity-0 w-0 overflow-hidden'}
              `}
            >
              {item.label}
            </span>

            {/* Badge */}
            {item.badge !== undefined && expanded && (
              <span className="ml-auto text-xs font-medium px-1.5 py-0.5 rounded-full bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300">
                {item.badge}
              </span>
            )}
          </button>
        )
      })}
    </nav>
  )
}
