import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { api, type LoadBalanceConfig, type ModelEndpoint } from '@/api/client'
import { Wrench, Plus, Trash2, Scale, MessageSquare } from 'lucide-react'

export default function Skills() {
  const [mcp, setMcp] = useState<string[]>([])
  const [skills, setSkills] = useState<string[]>([])
  const [mcpSource, setMcpSource] = useState('')
  const [skillSource, setSkillSource] = useState('')
  const [loadCfg, setLoadCfg] = useState<LoadBalanceConfig>({ models: [] })
  const [newModel, setNewModel] = useState<Partial<ModelEndpoint>>({ name: '', url: '', weight: 1 })
  const [loading, setLoading] = useState<string | null>(null)

  // 飞书开关状态
  const [feishuConfigured, setFeishuConfigured] = useState(false)
  const [feishuEnabled, setFeishuEnabled] = useState(false)
  const [feishuLoading, setFeishuLoading] = useState(false)

  const refresh = () => {
    api.get('/skills/mcp').then((r) => setMcp(r.data || [])).catch(() => {})
    api.get('/skills/list').then((r) => setSkills(r.data || [])).catch(() => {})
    api.get('/skills/load-balance').then((r) => setLoadCfg(r.data || { models: [] })).catch(() => {})
    api.get('/feishu/status').then((r) => {
      setFeishuConfigured(!!r.data?.configured)
      setFeishuEnabled(!!r.data?.enabled)
    }).catch(() => {})
  }
  useEffect(() => refresh(), [])

  const toggleFeishu = () => {
    setFeishuLoading(true)
    const action = feishuEnabled ? 'disable' : 'enable'
    api.post(`/feishu/${action}`, {})
      .then(() => setFeishuEnabled(!feishuEnabled))
      .catch(() => {})
      .finally(() => setFeishuLoading(false))
  }

  const installMcp = () => {
    if (!mcpSource.trim()) return
    setLoading('mcp')
    api.post('/skills/mcp', { source: mcpSource })
      .then(() => { setMcpSource(''); refresh() })
      .finally(() => setLoading(null))
  }

  const uninstallMcp = (name: string) => {
    setLoading('mcp')
    api.delete(`/skills/mcp/${encodeURIComponent(name)}`).then(refresh).finally(() => setLoading(null))
  }

  const installSkill = () => {
    if (!skillSource.trim()) return
    setLoading('skill')
    const cmd = skillSource.trim()
    const body = cmd.includes('npx') || cmd.includes('skills add') ? { command: cmd } : { source: cmd }
    api.post('/skills/install', body)
      .then(() => { setSkillSource(''); refresh() })
      .finally(() => setLoading(null))
  }

  const uninstallSkill = (name: string) => {
    setLoading('skill')
    api.delete(`/skills/skill/${encodeURIComponent(name)}`).then(refresh).finally(() => setLoading(null))
  }

  const addModel = () => {
    if (!newModel.name || !newModel.url) return
    const next = {
      models: [...loadCfg.models, { ...newModel, weight: newModel.weight ?? 1 } as ModelEndpoint],
    }
    setLoading('lb')
    api.put('/skills/load-balance', next)
      .then(() => { setNewModel({ name: '', url: '', weight: 1 }); refresh() })
      .finally(() => setLoading(null))
  }

  const removeModel = (idx: number) => {
    const next = { models: loadCfg.models.filter((_, i) => i !== idx) }
    setLoading('lb')
    api.put('/skills/load-balance', next).then(refresh).finally(() => setLoading(null))
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">MCP & Skills</h1>

      {/* 安装 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Wrench className="size-5" />
            安装
          </CardTitle>
          <CardContent className="pt-0 text-sm text-muted-foreground">
            npm 包名或 URL，如 @modelcontextprotocol/server-filesystem
          </CardContent>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2 opacity-60">
            <Input className="flex-1" placeholder="安装 MCP (暂未开放)" value={mcpSource} onChange={(e) => setMcpSource(e.target.value)} disabled />
            <Button onClick={installMcp} disabled>
              <Plus className="size-4 mr-2" /> 安装 MCP
            </Button>
          </div>
          <div className="flex gap-2">
            <Input
              className="flex-1"
              placeholder="npx skills add https://github.com/vercel-labs/skills --skill find-skills"
              value={skillSource}
              onChange={(e) => setSkillSource(e.target.value)}
            />
            <Button onClick={installSkill} disabled={loading === 'skill'}>
              <Plus className="size-4 mr-2" /> 安装 Skill
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* 已安装 */}
      <Card>
        <CardHeader>
          <CardTitle>已安装</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-6 md:grid-cols-2">
            <div>
              <h3 className="font-medium mb-2">MCP</h3>
              <ul className="space-y-1 text-sm">
                {mcp.length === 0 ? (
                  <li className="text-muted-foreground">暂无</li>
                ) : (
                  mcp.map((s) => (
                    <li key={s} className="flex items-center justify-between py-1 border-b">
                      {s}
                      <Button variant="ghost" size="sm" onClick={() => uninstallMcp(s)}>
                        <Trash2 className="size-4" />
                      </Button>
                    </li>
                  ))
                )}
              </ul>
            </div>
            <div>
              <h3 className="font-medium mb-2">Skills</h3>
              <ul className="space-y-1 text-sm">
                {skills.length === 0 ? (
                  <li className="text-muted-foreground">暂无</li>
                ) : (
                  skills.map((s) => (
                    <li key={s} className="flex items-center justify-between py-1 border-b">
                      {s}
                      <Button variant="ghost" size="sm" onClick={() => uninstallSkill(s)}>
                        <Trash2 className="size-4" />
                      </Button>
                    </li>
                  ))
                )}
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 飞书机器人 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MessageSquare className="size-5" />
            飞书机器人
          </CardTitle>
          <CardContent className="pt-0 text-sm text-muted-foreground">
            控制飞书群消息监听开关（需在配置中心填写 app_id / app_secret）
          </CardContent>
        </CardHeader>
        <CardContent>
          {!feishuConfigured ? (
            <p className="text-sm text-muted-foreground">飞书未配置，请先在配置中心填写 feishu.app_id 和 feishu.app_secret。</p>
          ) : (
            <div className="flex items-center gap-4">
              <span className="text-sm">
                当前状态：
                <span className={feishuEnabled ? 'text-green-600 font-medium' : 'text-muted-foreground'}>
                  {feishuEnabled ? '监听中' : '已关闭'}
                </span>
              </span>
              <Button
                variant={feishuEnabled ? 'destructive' : 'default'}
                size="sm"
                onClick={toggleFeishu}
                disabled={feishuLoading}
              >
                {feishuEnabled ? '关闭监听' : '开启监听'}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 负载均衡 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Scale className="size-5" />
            Token / 模型负载均衡
          </CardTitle>
          <CardContent className="pt-0 text-sm text-muted-foreground">
            配置多模型端点及权重
          </CardContent>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2 flex-wrap items-end">
            <Input className="w-32" placeholder="名称" value={newModel.name} onChange={(e) => setNewModel((m) => ({ ...m, name: e.target.value }))} />
            <Input className="flex-1 min-w-[180px]" placeholder="API URL" value={newModel.url} onChange={(e) => setNewModel((m) => ({ ...m, url: e.target.value }))} />
            <Input className="w-20" type="number" placeholder="权重" value={newModel.weight ?? 1} onChange={(e) => setNewModel((m) => ({ ...m, weight: parseInt(e.target.value) || 1 }))} />
            <Button onClick={addModel} disabled={loading === 'lb'}>添加</Button>
          </div>
          <ul className="space-y-1 text-sm">
            {loadCfg.models?.map((m, i) => (
              <li key={i} className="flex items-center justify-between py-2 border-b">
                <span>{m.name} · {m.url} (权重 {m.weight})</span>
                <Button variant="ghost" size="sm" onClick={() => removeModel(i)}>
                  <Trash2 className="size-4" />
                </Button>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>
    </div>
  )
}
