import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { api, type PipelineJob } from '@/api/client'
import { Plus, RefreshCw } from 'lucide-react'

export default function Pipeline() {
  const [jobs, setJobs] = useState<PipelineJob[]>([])
  const [loading, setLoading] = useState(false)
  const [repo, setRepo] = useState('')
  const [branch, setBranch] = useState('main')
  const [submitting, setSubmitting] = useState(false)

  const fetchJobs = () => {
    setLoading(true)
    api
      .get('/pipeline/jobs')
      .then((r) => setJobs(r.data || []))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchJobs()
  }, [])

  const triggerBuild = () => {
    if (!repo.trim()) return
    setSubmitting(true)
    api
      .post('/pipeline/trigger', { repo: repo.trim(), branch: branch.trim() || 'main' })
      .then(() => {
        setRepo('')
        fetchJobs()
      })
      .finally(() => setSubmitting(false))
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">构建流水线</h1>
      <Card>
        <CardHeader>
          <CardTitle>新建构建</CardTitle>
          <p className="text-sm text-muted-foreground">
            从 GitHub 拉取私有仓库，构建 Docker 镜像并推送到 Harbor
          </p>
        </CardHeader>
        <CardContent className="flex gap-4 flex-wrap">
          <input
            className="flex-1 min-w-[260px] rounded border px-3 py-2"
            placeholder="owner/repo 或 https://github.com/owner/repo"
            value={repo}
            onChange={(e) => setRepo(e.target.value)}
          />
          <input
            className="w-36 rounded border px-3 py-2"
            placeholder="分支"
            value={branch}
            onChange={(e) => setBranch(e.target.value)}
          />
          <Button onClick={triggerBuild} disabled={submitting}>
            <Plus className="size-4 mr-2" />
            {submitting ? '提交中...' : '触发构建'}
          </Button>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>构建历史</CardTitle>
          <Button variant="outline" size="sm" onClick={fetchJobs} disabled={loading}>
            <RefreshCw className="size-4" />
          </Button>
        </CardHeader>
        <CardContent>
          {jobs.length === 0 && !loading && (
            <p className="text-muted-foreground text-sm">暂无构建记录</p>
          )}
          {jobs.map((j) => (
            <div key={j.id} className="py-3 border-b last:border-0 space-y-1">
              <div className="flex items-center justify-between">
                <span className="font-medium">{j.repo} @ {j.branch}</span>
                <span className="text-sm text-muted-foreground">{j.status}</span>
              </div>
              <p className="text-xs text-muted-foreground">Job: {j.id}</p>
              {j.harborRepo && (
                <p className="text-xs text-muted-foreground">镜像: {j.harborRepo}</p>
              )}
              {j.startedAt && (
                <p className="text-xs text-muted-foreground">开始: {j.startedAt}</p>
              )}
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
