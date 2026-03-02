import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { GitBranch, Shield, Bot } from 'lucide-react'
import { api } from '@/api/client'
import { useEffect, useState } from 'react'

export default function Dashboard() {
  const [clashOk, setClashOk] = useState<boolean | null>(null)
  const [pipelineCount, setPipelineCount] = useState<number>(0)

  useEffect(() => {
    api.get('/clash/status').then((r) => setClashOk(r.data?.running ?? false)).catch(() => setClashOk(null))
    api.get('/pipeline/jobs').then((r) => setPipelineCount(r.data?.length ?? 0)).catch(() => {})
  }, [])

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">概览</h1>
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Clash 代理</CardTitle>
            <Shield className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">
              {clashOk === null ? '-' : clashOk ? '运行中' : '未运行'}
            </p>
            <p className="text-xs text-muted-foreground">git clone 走代理加速</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">构建任务</CardTitle>
            <GitBranch className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{pipelineCount}</p>
            <p className="text-xs text-muted-foreground">GitHub → Docker → Harbor</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">智能 Agent</CardTitle>
            <Bot className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">待接入</p>
            <p className="text-xs text-muted-foreground">LLM 沙盒 + 自我修改</p>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
