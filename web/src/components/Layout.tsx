import { Outlet, NavLink } from 'react-router-dom'
import { FolderGit2, Shield, Bot, Wrench, LayoutDashboard, Settings } from 'lucide-react'
import { cn } from '@/lib/utils'
import { AgentFloatingPanel } from '@/components/AgentFloatingPanel'

const navItems = [
  { to: '/dashboard', icon: LayoutDashboard, label: '概览' },
  { to: '/script', icon: FolderGit2, label: '脚本编排' },
  { to: '/mihomo', icon: Shield, label: 'Mihomo 代理' },
  { to: '/agent', icon: Bot, label: '智能 Agent' },
  { to: '/skills', icon: Wrench, label: 'MCP & Skills' },
  { to: '/config', icon: Settings, label: '配置中心' },
]

export function Layout() {
  return (
    <div className="min-h-screen flex bg-background">
      <aside className="w-60 border-r border-border bg-card/50 flex flex-col shrink-0">
        <div className="p-5 border-b border-border">
          <h1 className="font-bold text-xl tracking-tight text-foreground">Env King</h1>
          <p className="text-xs text-muted-foreground mt-0.5">团队智能助理</p>
        </div>
        <nav className="flex-1 p-3 space-y-0.5" aria-label="主导航">
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
                  isActive
                    ? 'bg-primary text-primary-foreground shadow-sm'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                )
              }
            >
              <Icon className="size-4 shrink-0" aria-hidden />
              <span>{label}</span>
            </NavLink>
          ))}
        </nav>
      </aside>
      <main className="flex-1 overflow-auto p-6 lg:p-8 min-w-0">
        <Outlet />
      </main>
      <AgentFloatingPanel />
    </div>
  )
}
