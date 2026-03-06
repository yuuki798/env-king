import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { api } from '@/api/client'
import { Settings, Loader2, Save } from 'lucide-react'

export default function Config() {
  const [raw, setRaw] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchConfig = () => {
    setLoading(true)
    setError(null)
    api
      .get<Record<string, unknown>>('/config')
      .then((r) => {
        setRaw(JSON.stringify(r.data || {}, null, 2))
      })
      .catch((e) => setError(e?.response?.data?.error || e.message || '获取配置失败'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchConfig()
  }, [])

  const save = () => {
    let data: Record<string, unknown>
    try {
      data = JSON.parse(raw)
    } catch (e) {
      setError('JSON 格式无效')
      return
    }
    setSaving(true)
    setError(null)
    api
      .put('/config', data)
      .then(() => fetchConfig())
      .catch((e) => setError(e?.response?.data?.error || e.message || '保存失败'))
      .finally(() => setSaving(false))
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
          <Settings className="size-7" />
          配置中心
        </h1>
        <Button onClick={save} disabled={saving || loading}>
          {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
          保存并热加载
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">当前配置</CardTitle>
          <p className="text-sm text-muted-foreground">
            修改后点击保存，将立即写入配置文件并生效（viper 热加载）。支持嵌套 JSON，与 config.dev.yaml / config.prod.yaml 结构一致。
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <div className="rounded-md bg-destructive/10 text-destructive text-sm px-3 py-2">
              {error}
            </div>
          )}
          {loading ? (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              加载中...
            </div>
          ) : (
            <>
              <div className="space-y-2">
                <Label>YAML 对应结构（JSON 编辑）</Label>
                <Textarea
                  className="font-mono text-sm min-h-[320px]"
                  value={raw}
                  onChange={(e) => setRaw(e.target.value)}
                  placeholder='{"github":{"token":""},"script":{"work_dir":"/tmp/env-king-script"}}'
                />
              </div>
              <Button onClick={save} disabled={saving}>
                {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
                保存并应用
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
