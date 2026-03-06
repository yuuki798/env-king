import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { FolderGit2, Shield, Bot, Settings, ArrowRight } from 'lucide-react'
import { api } from '@/api/client'
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'

export default function Dashboard() {
  const [mihomoOk, setMihomoOk] = useState<boolean | null>(null)
  const [scriptJobCount, setScriptJobCount] = useState<number>(0)

  useEffect(() => {
    api.get('/mihomo/status').then((r) => setMihomoOk(r.data?.running ?? false)).catch(() => setMihomoOk(null))
    api.get<unknown[]>('/script/jobs').then((r) => setScriptJobCount(r.data?.length ?? 0)).catch(() => {})
  }, [])

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">概览</h1>
        <p className="text-muted-foreground mt-1">服务状态与快捷入口</p>
      </div>
      <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="overflow-hidden transition-shadow hover:shadow-md">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Mihomo 代理</CardTitle>
            <Shield className="size-5 text-muted-foreground" aria-hidden />
          </CardHeader>
          <CardContent>
            <div className="flex items-baseline gap-2">
              {mihomoOk === null ? (
                <span className="text-2xl font-bold text-muted-foreground">—</span>
              ) : (
                <Badge variant={mihomoOk ? 'success' : 'secondary'} className="text-sm">
                  {mihomoOk ? '运行中' : '已停止'}
                </Badge>
              )}
            </div>
            <p className="text-xs text-muted-foreground mt-2">git / docker 走代理加速</p>
            <Button variant="ghost" size="sm" className="mt-2 -ml-2" asChild>
              <Link to="/mihomo">管理 <ArrowRight className="size-3 ml-1" /></Link>
            </Button>
          </CardContent>
        </Card>
        <Card className="overflow-hidden transition-shadow hover:shadow-md">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">脚本任务</CardTitle>
            <FolderGit2 className="size-5 text-muted-foreground" aria-hidden />
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{scriptJobCount}</p>
            <p className="text-xs text-muted-foreground mt-2">编排与工作流执行记录</p>
            <Button variant="ghost" size="sm" className="mt-2 -ml-2" asChild>
              <Link to="/script">查看 <ArrowRight className="size-3 ml-1" /></Link>
            </Button>
          </CardContent>
        </Card>
        <Card className="overflow-hidden transition-shadow hover:shadow-md">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">智能 Agent</CardTitle>
            <Bot className="size-5 text-muted-foreground" aria-hidden />
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">沙盒</p>
            <p className="text-xs text-muted-foreground mt-2">LLM 对话、Cron、自我修改</p>
            <Button variant="ghost" size="sm" className="mt-2 -ml-2" asChild>
              <Link to="/agent">进入 <ArrowRight className="size-3 ml-1" /></Link>
            </Button>
          </CardContent>
        </Card>
        <Card className="overflow-hidden transition-shadow hover:shadow-md">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">配置中心</CardTitle>
            <Settings className="size-5 text-muted-foreground" aria-hidden />
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">热加载</p>
            <p className="text-xs text-muted-foreground mt-2">修改配置即时生效</p>
            <Button variant="ghost" size="sm" className="mt-2 -ml-2" asChild>
              <Link to="/config">编辑 <ArrowRight className="size-3 ml-1" /></Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
