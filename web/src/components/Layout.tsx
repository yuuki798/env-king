import { Outlet, NavLink } from 'react-router-dom'
import { GitBranch, Shield, Bot, Wrench, LayoutDashboard } from 'lucide-react'
import { cn } from '@/lib/utils'

const navItems = [
  { to: '/dashboard', icon: LayoutDashboard, label: '概览' },
  { to: '/pipeline', icon: GitBranch, label: '构建流水线' },
  { to: '/clash', icon: Shield, label: 'Clash 代理' },
  { to: '/agent', icon: Bot, label: '智能 Agent' },
  { to: '/skills', icon: Wrench, label: 'MCP & Skills' },
]

export function Layout() {
  return (
    <div className="min-h-screen flex">
      <aside className="w-56 border-r bg-card flex flex-col">
        <div className="p-4 border-b">
          <h1 className="font-bold text-lg">Env King</h1>
          <p className="text-xs text-muted-foreground">团队智能助理</p>
        </div>
        <nav className="flex-1 p-2 space-y-1">
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors',
                  isActive
                    ? 'bg-primary text-primary-foreground'
                    : 'hover:bg-accent'
                )
              }
            >
              <Icon className="size-4" />
              {label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <main className="flex-1 overflow-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
