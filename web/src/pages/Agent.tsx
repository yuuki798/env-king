import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { api, type CronJob } from '@/api/client'
import { Bot, MessageSquare, Clock, GitCommit, Upload, Package } from 'lucide-react'

export default function Agent() {
  const [input, setInput] = useState('')
  const [messages, setMessages] = useState<{ role: string; content: string }[]>([])
  const [state, setState] = useState<{ workspaceDir: string; lastActivity?: string } | null>(null)
  const [cronJobs, setCronJobs] = useState<CronJob[]>([])
  const [cronSpec, setCronSpec] = useState('')
  const [cronScript, setCronScript] = useState('')
  const [gitMsg, setGitMsg] = useState('')
  const [diff, setDiff] = useState('')
  const [loading, setLoading] = useState<string | null>(null)

  const refresh = () => {
    api.get('/agent/state').then((r) => setState(r.data)).catch(() => {})
    api.get('/agent/cron').then((r) => setCronJobs(r.data || [])).catch(() => {})
  }
  useEffect(() => refresh(), [])

  const send = () => {
    if (!input.trim()) return
    setMessages((m) => [...m, { role: 'user', content: input }])
    const msg = input
    setInput('')
    setLoading('chat')
    api.post('/agent/chat', { message: msg })
      .then((r) => setMessages((m) => [...m, { role: 'assistant', content: r.data?.reply || '' }]))
      .finally(() => setLoading(null))
  }

  const addCron = () => {
    if (!cronSpec.trim() || !cronScript.trim()) return
    setLoading('cron')
    api.post('/agent/cron', { spec: cronSpec, script: cronScript })
      .then(() => { setCronSpec(''); setCronScript(''); refresh() })
      .finally(() => setLoading(null))
  }

  const removeCron = (name: string) => {
    setLoading('cron')
    api.delete(`/agent/cron/${encodeURIComponent(name)}`).then(refresh).finally(() => setLoading(null))
  }

  const doGitCommit = () => {
    if (!gitMsg.trim()) return
    setLoading('git')
    api.post('/agent/git/commit', { message: gitMsg }).then(() => setGitMsg('')).finally(() => setLoading(null))
  }

  const doGitPush = () => {
    setLoading('git')
    api.post('/agent/git/push').finally(() => setLoading(null))
  }

  const doDeploy = () => {
    setLoading('deploy')
    api.post('/agent/deploy').finally(() => setLoading(null))
  }

  const doSelfModify = () => {
    if (!diff.trim()) return
    setLoading('modify')
    api.post('/agent/self-modify', { diff }).then(() => setDiff('')).finally(() => setLoading(null))
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">智能 Agent</h1>
      {state && (
        <p className="text-sm text-muted-foreground">
          工作目录: {state.workspaceDir}
          {state.lastActivity && ` · 最后活动: ${state.lastActivity}`}
        </p>
      )}

      {/* 对话 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Bot className="size-5" />
            沙盒 Agent（模仿 OpenClaw）
          </CardTitle>
          <CardContent className="pt-0 text-sm text-muted-foreground">
            目录内 root 权限，支持 cron、自我修改、git、部署
          </CardContent>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="min-h-[200px] rounded border p-4 space-y-2 overflow-auto max-h-64">
              {messages.map((m, i) => (
                <div key={i} className={m.role === 'user' ? 'text-right' : ''}>
                  <span className="text-muted-foreground text-sm">{m.role}</span>
                  <p className="mt-1">{m.content}</p>
                </div>
              ))}
              {messages.length === 0 && (
                <p className="text-muted-foreground text-sm flex items-center gap-2">
                  <MessageSquare className="size-4" /> 与 Agent 对话
                </p>
              )}
            </div>
            <div className="flex gap-2">
              <input
                className="flex-1 rounded border px-3 py-2"
                placeholder="输入指令..."
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && send()}
              />
              <Button onClick={send} disabled={loading === 'chat'}>发送</Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Cron */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Clock className="size-5" />
            Cron 定时任务
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2 flex-wrap">
            <input
              className="w-24 rounded border px-3 py-2"
              placeholder="* * * * *"
              value={cronSpec}
              onChange={(e) => setCronSpec(e.target.value)}
            />
            <input
              className="flex-1 min-w-[200px] rounded border px-3 py-2"
              placeholder="脚本内容或路径"
              value={cronScript}
              onChange={(e) => setCronScript(e.target.value)}
            />
            <Button onClick={addCron} disabled={loading === 'cron'}>添加</Button>
          </div>
          <ul className="space-y-1 text-sm">
            {cronJobs.map((j) => (
              <li key={j.name} className="flex items-center justify-between py-1 border-b">
                <span><code className="text-muted-foreground">{j.spec}</code> {j.name}</span>
                <Button variant="ghost" size="sm" onClick={() => removeCron(j.name)}>删除</Button>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>

      {/* Git & Deploy */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <GitCommit className="size-5" />
            Git & 部署
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-4">
          <div className="flex gap-2 items-center">
            <input
              className="w-48 rounded border px-3 py-2"
              placeholder="commit message"
              value={gitMsg}
              onChange={(e) => setGitMsg(e.target.value)}
            />
            <Button onClick={doGitCommit} disabled={loading === 'git'}>Commit</Button>
          </div>
          <Button variant="outline" onClick={doGitPush} disabled={loading === 'git'}>
            <Upload className="size-4 mr-2" /> Push
          </Button>
          <Button variant="outline" onClick={doDeploy} disabled={loading === 'deploy'}>
            <Package className="size-4 mr-2" /> 部署
          </Button>
        </CardContent>
      </Card>

      {/* 自我修改 */}
      <Card>
        <CardHeader>
          <CardTitle>自我修改</CardTitle>
          <p className="text-sm text-muted-foreground">输入 diff 或补丁，修改 env-king 自身代码</p>
        </CardHeader>
        <CardContent>
          <textarea
            className="w-full h-24 rounded border px-3 py-2 font-mono text-sm"
            placeholder="diff 或补丁内容..."
            value={diff}
            onChange={(e) => setDiff(e.target.value)}
          />
          <Button className="mt-2" onClick={doSelfModify} disabled={loading === 'modify'}>应用</Button>
        </CardContent>
      </Card>
    </div>
  )
}
