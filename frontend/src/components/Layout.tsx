import { NavLink, Outlet, Link } from 'react-router'
import { LayoutDashboard, Inbox, Key, Settings, LogOut, Info } from 'lucide-react'
import { useAuth } from '@/lib/auth'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/inbox', icon: Inbox, label: 'Inbox' },
  { to: '/keywords', icon: Key, label: 'Keywords' },
  { to: '/settings', icon: Settings, label: 'Settings' },
]

export default function Layout() {
  const { user, logout } = useAuth()

  return (
    <div className="flex h-screen bg-background text-foreground">
      <aside className="hidden md:flex md:w-64 md:flex-col md:shrink-0 border-r border-border bg-sidebar">
        <div className="flex h-16 items-center px-5 border-b border-border">
          <span className="text-lg font-bold tracking-tight text-sidebar-foreground">
            Red<span className="text-red-500">Veluvanto</span>
          </span>
        </div>

        <div className="flex flex-1 flex-col gap-1 p-3 overflow-y-auto">
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-sidebar-accent text-sidebar-accent-foreground'
                    : 'text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground'
                )
              }
            >
              <Icon className="size-4 shrink-0" />
              {label}
            </NavLink>
          ))}
        </div>

        <div className="border-t border-border p-3">
          <div className="flex items-center gap-3 px-3 py-2 mb-1">
            <div className="size-7 rounded-full bg-primary flex items-center justify-center shrink-0">
              <span className="text-xs font-bold text-primary-foreground">
                {(user?.name as string | undefined)?.[0]?.toUpperCase() ??
                  (user?.email as string | undefined)?.[0]?.toUpperCase() ?? '?'}
              </span>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-sidebar-foreground truncate">
                {(user?.name as string | undefined) ?? (user?.email as string | undefined) ?? 'User'}
              </p>
            </div>
          </div>
          <Link
            to="/about"
            className="flex items-center gap-3 px-3 py-2 text-xs text-sidebar-foreground/50 hover:text-sidebar-foreground transition-colors"
          >
            <Info className="size-3.5 shrink-0" />
            About
          </Link>
          <Button
            variant="ghost"
            size="sm"
            onClick={logout}
            className="w-full justify-start gap-3 text-sidebar-foreground/70 hover:text-destructive"
          >
            <LogOut className="size-4 shrink-0" />
            Sign out
          </Button>
        </div>
      </aside>

      <main className="flex-1 overflow-auto pb-16 md:pb-0">
        <Outlet />
      </main>

      <nav className="fixed bottom-0 inset-x-0 z-50 flex h-16 items-center border-t border-border bg-sidebar md:hidden">
        {navItems.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              cn(
                'flex flex-1 flex-col items-center justify-center gap-1 py-2 text-xs font-medium transition-colors',
                isActive
                  ? 'text-sidebar-primary'
                  : 'text-sidebar-foreground/60 hover:text-sidebar-foreground'
              )
            }
          >
            <Icon className="size-5 shrink-0" />
            {label}
          </NavLink>
        ))}
        <button
          type="button"
          onClick={logout}
          className="flex flex-1 flex-col items-center justify-center gap-1 py-2 text-xs font-medium text-sidebar-foreground/60 hover:text-destructive transition-colors"
        >
          <LogOut className="size-5 shrink-0" />
          Sign out
        </button>
      </nav>
    </div>
  )
}
