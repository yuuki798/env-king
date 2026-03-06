import { useEffect, useRef, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { api, type ScriptJobView, type Workflow, type ScriptStep } from '@/api/client'
import { useAgentFormFill } from '@/contexts/AgentFormFillContext'
import {
  Play,
  RefreshCw,
  ListTodo,
  FolderGit2,
  Loader2,
  ChevronDown,
  ChevronRight,
  Trash2,
  Plus,
  X,
  Pencil,
  Copy,
} from 'lucide-react'
import { cn } from '@/lib/utils'

const statusVariant: Record<string, 'default' | 'secondary' | 'success' | 'destructive' | 'warning' | 'outline'> = {
  pending: 'secondary',
  running: 'default',
  succeeded: 'success',
  success: 'success',
  failed: 'destructive',
  canceled: 'outline',
}

export default function Script() {
  const [jobs, setJobs] = useState<ScriptJobView[]>([])
  const [workflows, setWorkflows] = useState<Workflow[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null)

  // 新建工作流表单
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [createName, setCreateName] = useState('')
  const [createWorkDir, setCreateWorkDir] = useState('workdir')
  const [createSteps, setCreateSteps] = useState<ScriptStep[]>([
    { name: '', kind: 'shell', command: '' },
  ])
  const [createDefaultInputs, setCreateDefaultInputs] = useState<{ key: string; value: string }[]>([
    { key: '', value: '' },
  ])
  const [createSubmitting, setCreateSubmitting] = useState(false)
  const [initPresetsLoading, setInitPresetsLoading] = useState(false)

  // 编辑工作流
  const [editingWorkflow, setEditingWorkflow] = useState<Workflow | null>(null)
  const [editName, setEditName] = useState('')
  const [editWorkDir, setEditWorkDir] = useState('')
  const [editSteps, setEditSteps] = useState<ScriptStep[]>([])
  const [editDefaultInputs, setEditDefaultInputs] = useState<{ key: string; value: string }[]>([])
  const [editSubmitting, setEditSubmitting] = useState(false)

  // 运行工作流：直接执行（参数已在工作流 defaultInputs 中填好）
  const [runningWorkflowId, setRunningWorkflowId] = useState<string | null>(null)

  // 任务执行窗口：执行后保留并轮询显示该任务的 shell 流
  const [activeJob, setActiveJob] = useState<ScriptJobView | null>(null)
  const [activeTab, setActiveTab] = useState<'jobs' | 'workflows'>('jobs')
  const logPreRef = useRef<HTMLPreElement>(null)
  const { formFill, clearFormFill } = useAgentFormFill()

  // Agent 自动填表：收到 script.createWorkflow 时填入新建工作流表单并打开
  useEffect(() => {
    if (!formFill || formFill.page !== 'script' || formFill.form !== 'createWorkflow') return
    const p = formFill.payload as {
      name?: string
      workDir?: string
      steps?: ScriptStep[]
      defaultInputs?: Record<string, string>
    }
    if (p.name != null) setCreateName(String(p.name))
    if (p.workDir != null) setCreateWorkDir(String(p.workDir))
    if (Array.isArray(p.steps) && p.steps.length > 0) {
      setCreateSteps(p.steps.map((s) => ({
        name: s?.name ?? '',
        kind: (s as ScriptStep)?.kind ?? 'shell',
        command: s?.command ?? '',
      })))
    }
    if (p.defaultInputs && typeof p.defaultInputs === 'object') {
      const entries = Object.entries(p.defaultInputs).filter(([k]) => k)
      setCreateDefaultInputs(
        entries.length
          ? entries.map(([key, value]) => ({ key, value: String(value ?? '') }))
          : [{ key: '', value: '' }]
      )
    }
    setActiveTab('workflows')
    setShowCreateForm(true)
    setSelectedJobId(null)
    clearFormFill()
  }, [formFill, clearFormFill])

  const fetchJobs = () => {
    api.get<ScriptJobView[]>('/script/jobs').then((r) => setJobs(r.data || [])).catch(() => [])
  }
  const fetchWorkflows = () => {
    api.get<Workflow[]>('/script/workflows').then((r) => setWorkflows(r.data || [])).catch(() => [])
  }

  useEffect(() => {
    setLoading(true)
    Promise.all([api.get('/script/jobs'), api.get('/script/workflows')])
      .then(([jr, wr]) => {
        setJobs(jr.data || [])
        setWorkflows(wr.data || [])
      })
      .finally(() => setLoading(false))
  }, [])

  // 任务窗口：轮询当前任务状态与日志，直到结束
  useEffect(() => {
    if (!activeJob?.id) return
    const status = activeJob.status
    if (status !== 'pending' && status !== 'running') return

    const timer = setInterval(() => {
      api.get<ScriptJobView>(`/script/jobs/${activeJob.id}`).then((r) => {
        const j = r.data
        if (j) setActiveJob(j)
      })
    }, 1500)
    return () => clearInterval(timer)
  }, [activeJob?.id, activeJob?.status])

  // 任务窗口日志自动滚到底部
  useEffect(() => {
    if (activeJob?.logs && logPreRef.current) {
      logPreRef.current.scrollTop = logPreRef.current.scrollHeight
    }
  }, [activeJob?.logs])

  const runWorkflowDirect = (wf: Workflow) => {
    setRunningWorkflowId(wf.id)
    api
      .post<ScriptJobView>(`/script/workflows/${wf.id}/run`, {})
      .then((res) => {
        const job = res.data
        if (job?.id) setActiveJob(job)
        fetchJobs()
      })
      .finally(() => setRunningWorkflowId(null))
  }

  const submitCreateWorkflow = () => {
    const name = createName.trim()
    if (!name) return
    const steps = createSteps.filter((s) => s.name.trim() || s.command.trim())
    if (steps.length === 0) return
    const defaultInputs: Record<string, string> = {}
    createDefaultInputs.forEach(({ key, value }) => {
      const k = key.trim()
      if (k) defaultInputs[k] = value.trim()
    })
    setCreateSubmitting(true)
    api
      .post('/script/workflows', {
        name,
        workDir: createWorkDir.trim() || undefined,
        steps,
        defaultInputs: Object.keys(defaultInputs).length ? defaultInputs : undefined,
      })
      .then(() => {
        fetchWorkflows()
        setShowCreateForm(false)
        setCreateName('')
        setCreateWorkDir('')
        setCreateSteps([{ name: '', kind: 'shell', command: '' }])
        setCreateDefaultInputs([{ key: '', value: '' }])
      })
      .finally(() => setCreateSubmitting(false))
  }

  const openEditWorkflow = (wf: Workflow) => {
    setEditingWorkflow(wf)
    setEditName(wf.name ?? '')
    setEditWorkDir(wf.workDir ?? 'workdir')
    setEditSteps(
      wf.steps?.length
        ? wf.steps.map((s) => ({ name: s.name ?? '', kind: s.kind ?? 'shell', command: s.command ?? '' }))
        : [{ name: '', kind: 'shell', command: '' }]
    )
    const def = wf.defaultInputs
    setEditDefaultInputs(
      def && Object.keys(def).length > 0
        ? Object.entries(def).map(([key, value]) => ({ key, value: value ?? '' }))
        : [{ key: '', value: '' }]
    )
  }

  const submitEditWorkflow = () => {
    if (!editingWorkflow) return
    const name = editName.trim()
    if (!name) return
    const steps = editSteps.filter((s) => s.name.trim() || s.command.trim())
    if (steps.length === 0) return
    const defaultInputs: Record<string, string> = {}
    editDefaultInputs.forEach(({ key, value }) => {
      const k = key.trim()
      if (k) defaultInputs[k] = value.trim()
    })
    setEditSubmitting(true)
    api
      .put(`/script/workflows/${editingWorkflow.id}`, {
        name,
        workDir: editWorkDir.trim() || undefined,
        steps,
        defaultInputs: Object.keys(defaultInputs).length ? defaultInputs : undefined,
      })
      .then(() => {
        fetchWorkflows()
        setEditingWorkflow(null)
      })
      .finally(() => setEditSubmitting(false))
  }

  const openCopyWorkflow = (wf: Workflow) => {
    setShowCreateForm(true)
    setCreateName((wf.name || '未命名') + ' (副本)')
    setCreateWorkDir(wf.workDir?.trim() || 'workdir')
    setCreateSteps(
      wf.steps?.length
        ? wf.steps.map((s) => ({ name: s.name ?? '', kind: s.kind ?? 'shell', command: s.command ?? '' }))
        : [{ name: '', kind: 'shell', command: '' }]
    )
    const def = wf.defaultInputs
    setCreateDefaultInputs(
      def && Object.keys(def).length > 0
        ? Object.entries(def).map(([key, value]) => ({ key, value: value ?? '' }))
        : [{ key: '', value: '' }]
    )
  }

  const deleteWorkflow = (id: string) => {
    if (!confirm('确定删除该工作流？')) return
    api.delete(`/script/workflows/${id}`).then(fetchWorkflows)
  }

  const initPresets = () => {
    setInitPresetsLoading(true)
    api
      .post('/script/workflows/init')
      .then(() => fetchWorkflows())
      .catch(() => {})
      .finally(() => setInitPresetsLoading(false))
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold tracking-tight">脚本编排</h1>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            fetchJobs()
            fetchWorkflows()
          }}
          disabled={loading}
        >
          <RefreshCw className={cn('size-4', loading && 'animate-spin')} />
          刷新
        </Button>
      </div>

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as 'jobs' | 'workflows')} className="space-y-4">
        <TabsList className="grid w-full grid-cols-2 lg:w-auto lg:inline-grid">
          <TabsTrigger value="jobs" className="gap-2">
            <ListTodo className="size-4" />
            任务列表
          </TabsTrigger>
          <TabsTrigger value="workflows" className="gap-2">
            <FolderGit2 className="size-4" />
            工作流
          </TabsTrigger>
        </TabsList>

        <TabsContent value="jobs" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">执行记录</CardTitle>
              <p className="text-sm text-muted-foreground">工作流运行记录，点击可展开日志</p>
            </CardHeader>
            <CardContent>
              {jobs.length === 0 && !loading && (
                <p className="text-sm text-muted-foreground py-8 text-center">暂无任务</p>
              )}
              <ul className="space-y-1">
                {jobs.map((j) => (
                  <li
                    key={j.id}
                    className={cn(
                      'rounded-lg border p-3 transition-colors cursor-pointer',
                      selectedJobId === j.id ? 'bg-accent/50 border-primary/30' : 'hover:bg-accent/30'
                    )}
                    onClick={() => setSelectedJobId(selectedJobId === j.id ? null : j.id)}
                  >
                    <div className="flex items-center justify-between gap-2 flex-wrap">
                      <div className="flex items-center gap-2 min-w-0">
                        {selectedJobId === j.id ? (
                          <ChevronDown className="size-4 shrink-0" />
                        ) : (
                          <ChevronRight className="size-4 shrink-0" />
                        )}
                        <span className="font-mono text-xs text-muted-foreground truncate">{j.id}</span>
                        {j.meta?.workflow && (
                          <span className="text-xs text-muted-foreground truncate">{j.meta.workflow}</span>
                        )}
                      </div>
                      <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        {(j.enqueued_at || j.startedAt || j.finishedAt) && (
                          <span>
                            {new Date(j.enqueued_at || j.startedAt || j.finishedAt || '').toLocaleString()}
                          </span>
                        )}
                        <Badge variant={statusVariant[j.status] ?? 'outline'}>{j.status}</Badge>
                      </div>
                    </div>
                    {selectedJobId === j.id && (
                      <div className="mt-3 pt-3 border-t">
                        {j.workDir && (
                          <p className="text-xs text-muted-foreground mb-1">工作目录: {j.workDir}</p>
                        )}
                        <pre className="text-xs bg-muted rounded p-3 max-h-64 overflow-auto whitespace-pre-wrap">
                          {j.logs || '(无输出)'}
                        </pre>
                        {j.error && (
                          <p className="text-xs text-destructive mt-1">错误: {j.error}</p>
                        )}
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="workflows" className="space-y-4">
          {/* 新建工作流表单 */}
          {showCreateForm ? (
            <Card>
              <CardHeader className="flex flex-row items-center justify-between">
                <CardTitle className="text-lg">新建工作流</CardTitle>
                <Button variant="ghost" size="icon" onClick={() => setShowCreateForm(false)} aria-label="关闭">
                  <X className="size-4" />
                </Button>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>名称</Label>
                  <Input
                    placeholder="如：Clone + Build + Push"
                    value={createName}
                    onChange={(e) => setCreateName(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>工作目录（可选）</Label>
                  <Input
                    placeholder="留空则每次运行使用临时目录"
                    value={createWorkDir}
                    onChange={(e) => setCreateWorkDir(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>步骤（命令中可用 &#123;&#123;变量名&#125;&#125; 占位）</Label>
                  {createSteps.map((step, i) => (
                    <div key={i} className="flex gap-2 items-start flex-wrap rounded border p-3 bg-muted/30">
                      <Input
                        placeholder="步骤名"
                        className="w-36"
                        value={step.name}
                        onChange={(e) => {
                          const n = [...createSteps]
                          n[i] = { ...n[i], name: e.target.value }
                          setCreateSteps(n)
                        }}
                      />
                      <Input
                        placeholder="命令，如 git clone {{repo}} -b {{branch}} ."
                        className="flex-1 min-w-[200px]"
                        value={step.command}
                        onChange={(e) => {
                          const n = [...createSteps]
                          n[i] = { ...n[i], command: e.target.value }
                          setCreateSteps(n)
                        }}
                      />
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setCreateSteps((s) => s.filter((_, idx) => idx !== i))}
                        aria-label="删除步骤"
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      setCreateSteps((s) => [...s, { name: '', kind: 'shell', command: '' }])
                    }
                  >
                    <Plus className="size-4" />
                    添加步骤
                  </Button>
                </div>
                <div className="space-y-2">
                  <Label>默认参数（键值对，创建时填好，运行直接使用）</Label>
                  {createDefaultInputs.map((row, i) => (
                    <div key={i} className="flex gap-2 items-center">
                      <Input
                        placeholder="参数名"
                        className="w-32"
                        value={row.key}
                        onChange={(e) => {
                          const n = [...createDefaultInputs]
                          n[i] = { ...n[i], key: e.target.value }
                          setCreateDefaultInputs(n)
                        }}
                      />
                      <Input
                        placeholder="默认值"
                        className="flex-1 min-w-[120px]"
                        value={row.value}
                        onChange={(e) => {
                          const n = [...createDefaultInputs]
                          n[i] = { ...n[i], value: e.target.value }
                          setCreateDefaultInputs(n)
                        }}
                      />
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() =>
                          setCreateDefaultInputs((s) => s.filter((_, idx) => idx !== i))
                        }
                        aria-label="删除"
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCreateDefaultInputs((s) => [...s, { key: '', value: '' }])}
                  >
                    <Plus className="size-4" />
                    添加参数
                  </Button>
                </div>
                <div className="flex gap-2">
                  <Button onClick={submitCreateWorkflow} disabled={createSubmitting}>
                    {createSubmitting ? (
                      <Loader2 className="size-4 animate-spin" />
                    ) : (
                      <Plus className="size-4" />
                    )}
                    创建
                  </Button>
                  <Button variant="outline" onClick={() => setShowCreateForm(false)}>
                    取消
                  </Button>
                </div>
              </CardContent>
            </Card>
          ) : (
            <div className="flex gap-2">
              <Button onClick={() => setShowCreateForm(true)}>
                <Plus className="size-4" />
                新建工作流
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={initPresets}
                disabled={initPresetsLoading}
              >
                {initPresetsLoading ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : null}
                初始化预设
              </Button>
            </div>
          )}

          {/* 工作流列表 */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">工作流列表</CardTitle>
              <p className="text-sm text-muted-foreground">参数已在工作流中配置，点击「运行」直接执行</p>
            </CardHeader>
            <CardContent>
              {workflows.length === 0 && !loading && (
                <p className="text-sm text-muted-foreground py-8 text-center">
                  暂无工作流，请点击「新建工作流」或「初始化预设」
                </p>
              )}
              <ul className="space-y-3">
                {workflows.map((wf) => (
                  <li
                    key={wf.id}
                    className="rounded-lg border p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                  >
                    <div className="min-w-0">
                      <p className="font-medium truncate">{wf.name}</p>
                      <p className="text-xs text-muted-foreground mt-1">
                        {wf.steps?.length ?? 0} 步
                        {wf.workDir ? ` · ${wf.workDir}` : ' · 运行时可指定工作目录'}
                      </p>
                      {wf.steps?.length ? (
                        <ul className="text-xs text-muted-foreground mt-2 space-y-0.5">
                          {wf.steps.slice(0, 3).map((s, i) => (
                            <li key={i}>
                              {s.name || '(未命名)'}: {s.command.slice(0, 50)}
                              {s.command.length > 50 ? '…' : ''}
                            </li>
                          ))}
                          {(wf.steps?.length ?? 0) > 3 && (
                            <li>… 共 {wf.steps?.length} 步</li>
                          )}
                        </ul>
                      ) : null}
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <Button
                        size="sm"
                        onClick={() => runWorkflowDirect(wf)}
                        disabled={runningWorkflowId === wf.id}
                      >
                        {runningWorkflowId === wf.id ? (
                          <Loader2 className="size-4 animate-spin" />
                        ) : (
                          <Play className="size-4" />
                        )}
                        运行
                      </Button>
                      <Button variant="outline" size="sm" onClick={() => openEditWorkflow(wf)}>
                        <Pencil className="size-4" />
                        编辑
                      </Button>
                      <Button variant="outline" size="sm" onClick={() => openCopyWorkflow(wf)}>
                        <Copy className="size-4" />
                        复制
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => deleteWorkflow(wf.id)}
                        className="text-destructive hover:text-destructive"
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 编辑工作流弹窗 */}
      {editingWorkflow && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50"
          onClick={() => !editSubmitting && setEditingWorkflow(null)}
          role="dialog"
          aria-modal="true"
          aria-labelledby="edit-workflow-title"
        >
          <Card
            className="w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col shadow-lg"
            onClick={(e) => e.stopPropagation()}
          >
            <CardHeader className="flex flex-row items-center justify-between shrink-0">
              <CardTitle id="edit-workflow-title">编辑工作流：{editingWorkflow.name}</CardTitle>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => !editSubmitting && setEditingWorkflow(null)}
                aria-label="关闭"
              >
                <X className="size-4" />
              </Button>
            </CardHeader>
            <CardContent className="space-y-4 overflow-y-auto">
              <div className="space-y-2">
                <Label>名称</Label>
                <Input
                  placeholder="如：Clone + Build + Push"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>工作目录（可选）</Label>
                <Input
                  placeholder="留空则每次运行使用临时目录"
                  value={editWorkDir}
                  onChange={(e) => setEditWorkDir(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label>步骤（命令中可用 &#123;&#123;变量名&#125;&#125; 占位）</Label>
                {editSteps.map((step, i) => (
                  <div key={i} className="flex gap-2 items-start flex-wrap rounded border p-3 bg-muted/30">
                    <Input
                      placeholder="步骤名"
                      className="w-36"
                      value={step.name}
                      onChange={(e) => {
                        const n = [...editSteps]
                        n[i] = { ...n[i], name: e.target.value }
                        setEditSteps(n)
                      }}
                    />
                    <Input
                      placeholder="命令"
                      className="flex-1 min-w-[200px]"
                      value={step.command}
                      onChange={(e) => {
                        const n = [...editSteps]
                        n[i] = { ...n[i], command: e.target.value }
                        setEditSteps(n)
                      }}
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => setEditSteps((s) => s.filter((_, idx) => idx !== i))}
                      aria-label="删除步骤"
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                ))}
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setEditSteps((s) => [...s, { name: '', kind: 'shell', command: '' }])}
                >
                  <Plus className="size-4" />
                  添加步骤
                </Button>
              </div>
              <div className="space-y-2">
                <Label>默认参数（键值对，创建时填好，运行直接使用）</Label>
                {editDefaultInputs.map((row, i) => (
                  <div key={i} className="flex gap-2 items-center">
                    <Input
                      placeholder="参数名"
                      className="w-32"
                      value={row.key}
                      onChange={(e) => {
                        const n = [...editDefaultInputs]
                        n[i] = { ...n[i], key: e.target.value }
                        setEditDefaultInputs(n)
                      }}
                    />
                    <Input
                      placeholder="默认值"
                      className="flex-1 min-w-[120px]"
                      value={row.value}
                      onChange={(e) => {
                        const n = [...editDefaultInputs]
                        n[i] = { ...n[i], value: e.target.value }
                        setEditDefaultInputs(n)
                      }}
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => setEditDefaultInputs((s) => s.filter((_, idx) => idx !== i))}
                      aria-label="删除"
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                ))}
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setEditDefaultInputs((s) => [...s, { key: '', value: '' }])}
                >
                  <Plus className="size-4" />
                  添加参数
                </Button>
              </div>
              <div className="flex gap-2 pt-2">
                <Button onClick={submitEditWorkflow} disabled={editSubmitting}>
                  {editSubmitting ? <Loader2 className="size-4 animate-spin" /> : null}
                  保存
                </Button>
                <Button
                  variant="outline"
                  onClick={() => !editSubmitting && setEditingWorkflow(null)}
                  disabled={editSubmitting}
                >
                  取消
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* 任务执行窗口：实时显示 shell 流 */}
      {activeJob && (
        <Card className="border-primary/30 bg-zinc-900/95">
          <CardHeader className="flex flex-row items-center justify-between py-3">
            <div className="flex items-center gap-2 flex-wrap min-w-0">
              <CardTitle className="text-base font-mono text-green-400">
                任务 {activeJob.id.slice(0, 8)}…
              </CardTitle>
              <Badge variant={statusVariant[activeJob.status] ?? 'outline'}>{activeJob.status}</Badge>
              {activeJob.meta?.workflow && (
                <span className="text-xs text-muted-foreground truncate">{activeJob.meta.workflow}</span>
              )}
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setActiveJob(null)}
              aria-label="关闭任务窗口"
            >
              <X className="size-4" />
              关闭
            </Button>
          </CardHeader>
          <CardContent className="pt-0 space-y-2">
            {activeJob.workDir && (
              <p className="text-xs text-muted-foreground">工作目录: {activeJob.workDir}</p>
            )}
            <pre
              ref={logPreRef}
              className="text-xs font-mono text-green-300/90 bg-black/50 rounded-lg p-4 max-h-[320px] overflow-auto whitespace-pre-wrap border border-zinc-700"
            >
              {activeJob.logs || (activeJob.status === 'pending' ? '等待执行…' : '(无输出)')}
            </pre>
            {activeJob.error && (
              <p className="text-xs text-destructive">错误: {activeJob.error}</p>
            )}
          </CardContent>
        </Card>
      )}

    </div>
  )
}
