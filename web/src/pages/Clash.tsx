import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { api } from '@/api/client'
import { Shield, Power, RefreshCw } from 'lucide-react'

type ClashStatus = {
  running: boolean
  proxyPort?: number
  mixedPort?: number
  note?: string
}

export default function Clash() {
  const [status, setStatus] = useState<ClashStatus>({ running: false })
  const [loading, setLoading] = useState(false)

  const fetchStatus = () => {
    api
      .get('/clash/status')
      .then((r) => setStatus(r.data || { running: false }))
      .catch(() => setStatus({ running: false }))
  }

  useEffect(() => fetchStatus(), [])

  const toggle = () => {
    setLoading(true)
    api
      .post(status.running ? '/clash/stop' : '/clash/start')
      .then(fetchStatus)
      .finally(() => setLoading(false))
  }

  const proxyHint = status.proxyPort
    ? `http://127.0.0.1:${status.proxyPort}`
    : 'http://127.0.0.1:7890'

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Clash 代理</h1>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Shield className="size-5" />
              Clash for Linux
            </CardTitle>
            <p className="pt-2 text-sm text-muted-foreground">
              集成 clash-for-linux-install，git clone / docker pull 可通过代理加速
            </p>
          </div>
          <Button variant="outline" size="sm" onClick={fetchStatus}>
            <RefreshCw className="size-4" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-4">
            <Button onClick={toggle} disabled={loading}>
              <Power className="size-4 mr-2" />
              {status.running ? '停止' : '启动'}
            </Button>
            <span className={status.running ? 'text-green-600' : 'text-muted-foreground'}>
              {status.running ? '运行中' : '已停止'}
            </span>
          </div>
          <div className="text-sm text-muted-foreground space-y-1">
            <p>建议设置：</p>
            <p><code>HTTP_PROXY={proxyHint}</code></p>
            <p><code>HTTPS_PROXY={proxyHint}</code></p>
            {status.note && <p>{status.note}</p>}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
