import { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/api/client'
import { Shield, Power, RefreshCw, Loader2, RotateCw, Save, Plus, Trash2, Link2, Globe } from 'lucide-react'

type MihomoStatus = {
  running: boolean
  proxyPort?: number
  mixedPort?: number
  note?: string
}

type SystemProxyStatus = {
  enabled: boolean
  host: string
  port: number
}

type Subscription = {
  id: string
  name: string
  url: string
}

// mihomo /proxies 返回格式：{ proxies: { [name]: { type, now?, all? } } }
type ProxiesResponse = {
  proxies?: Record<
    string,
    { type: string; now?: string; all?: string[]; name?: string }
  >
}

export default function Mihomo() {
  const [status, setStatus] = useState<MihomoStatus>({ running: false })
  const [loading, setLoading] = useState(false)
  const [configYaml, setConfigYaml] = useState('')
  const [configLoading, setConfigLoading] = useState(false)
  const [configSaving, setConfigSaving] = useState(false)
  const [restarting, setRestarting] = useState(false)
  const [configMessage, setConfigMessage] = useState<string | null>(null)

  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [subName, setSubName] = useState('')
  const [subUrl, setSubUrl] = useState('')
  const [subLoading, setSubLoading] = useState(false)
  const [applyLoading, setApplyLoading] = useState(false)

  const [proxiesData, setProxiesData] = useState<ProxiesResponse | null>(null)
  const [proxiesLoading, setProxiesLoading] = useState(false)
  const [proxySelectLoading, setProxySelectLoading] = useState<string | null>(null)

  const [sysProxy, setSysProxy] = useState<SystemProxyStatus>({ enabled: false, host: '127.0.0.1', port: 17890 })
  const [sysProxyLoading, setSysProxyLoading] = useState(false)

  const [mode, setMode] = useState<string>('')
  const [modeLoading, setModeLoading] = useState(false)

  const fetchStatus = () => {
    api
      .get('/mihomo/status')
      .then((r) => setStatus(r.data || { running: false }))
      .catch(() => setStatus({ running: false }))
  }

  const fetchConfig = () => {
    setConfigLoading(true)
    setConfigMessage(null)
    api
      .get<string>('/mihomo/config', { responseType: 'text' })
      .then((r) => setConfigYaml(typeof r.data === 'string' ? r.data : ''))
      .catch(() => setConfigMessage('无法加载配置'))
      .finally(() => setConfigLoading(false))
  }

  const fetchSysProxy = () => {
    api.get<SystemProxyStatus>('/mihomo/system-proxy')
      .then((r) => setSysProxy(r.data || { enabled: false, host: '127.0.0.1', port: 17890 }))
      .catch(() => {})
  }

  const toggleSysProxy = () => {
    setSysProxyLoading(true)
    api.post(sysProxy.enabled ? '/mihomo/system-proxy/off' : '/mihomo/system-proxy/on')
      .then((r) => setSysProxy(r.data?.status || { ...sysProxy, enabled: !sysProxy.enabled }))
      .catch((e) => setConfigMessage(e?.response?.data?.error || '系统代理设置失败'))
      .finally(() => setSysProxyLoading(false))
  }

  const fetchSubscriptions = () => {
    api
      .get<Subscription[]>('/mihomo/subscriptions')
      .then((r) => setSubscriptions(Array.isArray(r.data) ? r.data : []))
      .catch(() => setSubscriptions([]))
  }

  const fetchProxies = () => {
    if (!status.running) return
    setProxiesLoading(true)
    api
      .get<ProxiesResponse>('/mihomo/proxies')
      .then((r) => setProxiesData(r.data || null))
      .catch(() => setProxiesData(null))
      .finally(() => setProxiesLoading(false))
  }

  const fetchMode = () => {
    if (!status.running) return
    api
      .get<{ mode: string }>('/mihomo/mode')
      .then((r) => setMode(r.data?.mode || ''))
      .catch(() => {})
  }

  const switchMode = (newMode: string) => {
    setModeLoading(true)
    api
      .put('/mihomo/mode', { mode: newMode })
      .then(() => setMode(newMode))
      .catch((e) => setConfigMessage(e?.response?.data?.error || '模式切换失败'))
      .finally(() => setModeLoading(false))
  }

  useEffect(() => {
    fetchStatus()
    fetchConfig()
    fetchSubscriptions()
    fetchSysProxy()
  }, [])

  useEffect(() => {
    if (status.running) {
      fetchProxies()
      fetchMode()
    } else {
      setProxiesData(null)
      setMode('')
    }
  }, [status.running])

  const toggle = () => {
    setLoading(true)
    api
      .post(status.running ? '/mihomo/stop' : '/mihomo/start')
      .then(() => {
        fetchStatus()
        if (!status.running) setTimeout(fetchProxies, 500)
      })
      .finally(() => setLoading(false))
  }

  const saveConfig = () => {
    setConfigSaving(true)
    setConfigMessage(null)
    api
      .put('/mihomo/config', configYaml, { headers: { 'Content-Type': 'application/x-yaml' } })
      .then(() => {
        setConfigMessage('已保存，请点击「重启」使配置生效')
      })
      .catch((e) => setConfigMessage(e?.response?.data?.error || '保存失败'))
      .finally(() => setConfigSaving(false))
  }

  const restart = () => {
    setRestarting(true)
    setConfigMessage(null)
    api
      .post('/mihomo/restart')
      .then(() => {
        fetchStatus()
        setConfigMessage('已重启')
        setTimeout(fetchProxies, 500)
      })
      .catch((e) => setConfigMessage(e?.response?.data?.error || '重启失败'))
      .finally(() => setRestarting(false))
  }

  const addSubscription = () => {
    if (!subName.trim() || !subUrl.trim()) return
    setSubLoading(true)
    api
      .post<Subscription>('/mihomo/subscriptions', { name: subName.trim(), url: subUrl.trim() })
        .then((r) => {
        setSubscriptions((prev) => [...(Array.isArray(prev) ? prev : []), r.data])
        setSubName('')
        setSubUrl('')
      })
      .finally(() => setSubLoading(false))
  }

  const deleteSubscription = (id: string) => {
    api.delete(`/mihomo/subscriptions/${id}`).then(() => fetchSubscriptions())
  }

  const applySubscriptions = () => {
    setApplyLoading(true)
    api
      .post('/mihomo/subscriptions/apply')
      .then(() => {
        setConfigMessage('已应用订阅到 config，请刷新配置或重启使生效')
        fetchConfig()
      })
      .catch((e) => setConfigMessage(e?.response?.data?.error || '应用失败'))
      .finally(() => setApplyLoading(false))
  }

  const setProxy = (groupName: string, selectedName: string) => {
    setProxySelectLoading(groupName)
    api
      .put(`/mihomo/proxies/${encodeURIComponent(groupName)}`, { name: selectedName })
      .then(() => fetchProxies())
      .finally(() => setProxySelectLoading(null))
  }

  const proxyHint = `http://127.0.0.1:${sysProxy.port}`

  const selectorGroups =
    proxiesData?.proxies &&
    Object.entries(proxiesData.proxies).filter(
      ([_, p]) => p && (p.type === 'Selector' || p.type === 'select')
    )

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Mihomo 代理</h1>
        <p className="text-muted-foreground mt-1">代理加速 git / docker，订阅与节点选择（类 Clash Verge）</p>
      </div>
      <Card className="overflow-hidden">
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div>
            <CardTitle className="flex items-center gap-2 text-lg">
              <Shield className="size-5 text-primary" aria-hidden />
              Mihomo
            </CardTitle>
            <p className="text-sm text-muted-foreground mt-2">
              嵌入 mihomo 内核，git clone / docker pull 可通过代理加速
            </p>
          </div>
          <Button variant="outline" size="icon" onClick={fetchStatus} aria-label="刷新状态">
            <RefreshCw className="size-4" />
          </Button>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="flex flex-wrap items-center gap-3">
            <Button onClick={toggle} disabled={loading} size="lg">
              {loading ? <Loader2 className="size-4 animate-spin" /> : <Power className="size-4" />}
              {status.running ? '停止内核' : '启动内核'}
            </Button>
            <Button
              variant="outline"
              size="lg"
              onClick={restart}
              disabled={restarting || !status.running}
              aria-label="重启代理"
            >
              {restarting ? <Loader2 className="size-4 animate-spin" /> : <RotateCw className="size-4" />}
              重启
            </Button>
            <Badge variant={status.running ? 'success' : 'secondary'} className="text-sm">
              内核 {status.running ? '运行中' : '已停止'}
            </Badge>
          </div>
          {/* 系统代理开关 */}
          <div className="flex items-center gap-3 rounded-lg border px-4 py-3">
            <Globe className="size-5 shrink-0 text-muted-foreground" />
            <div className="flex-1">
              <p className="text-sm font-medium">系统代理</p>
              <p className="text-xs text-muted-foreground">
                {sysProxy.enabled
                  ? `已接管：${sysProxy.host}:${sysProxy.port}`
                  : `未开启 — 浏览器/git 不走代理`}
              </p>
            </div>
            <Button
              variant={sysProxy.enabled ? 'default' : 'outline'}
              size="sm"
              onClick={toggleSysProxy}
              disabled={sysProxyLoading || !status.running}
            >
              {sysProxyLoading
                ? <Loader2 className="size-4 animate-spin" />
                : sysProxy.enabled ? '关闭系统代理' : '开启系统代理'}
            </Button>
          </div>
          <div className="rounded-lg bg-muted/50 p-4 text-sm">
            <p className="font-medium text-foreground mb-2">建议环境变量</p>
            <code className="block text-muted-foreground break-all">HTTP_PROXY={proxyHint}</code>
            <code className="block text-muted-foreground break-all mt-1">HTTPS_PROXY={proxyHint}</code>
            {status.note && <p className="mt-2 text-muted-foreground">{status.note}</p>}
          </div>
        </CardContent>
      </Card>

      {/* 订阅链接 */}
      <Card className="overflow-hidden">
        <CardHeader className="flex flex-row items-center justify-between gap-4">
          <CardTitle className="flex items-center gap-2 text-lg">
            <Link2 className="size-5" />
            订阅链接
          </CardTitle>
          <Button
            size="sm"
            onClick={applySubscriptions}
            disabled={applyLoading || !Array.isArray(subscriptions) || subscriptions.length === 0}
          >
            {applyLoading ? <Loader2 className="size-4 animate-spin" /> : '应用订阅到配置'}
          </Button>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            添加订阅 URL 后点击「应用订阅到配置」，将写入 proxy-providers 与「订阅」选择组；重启后 mihomo 会拉取节点。
          </p>
          <div className="flex flex-wrap gap-2">
            <Input
              className="w-40"
              placeholder="名称"
              value={subName}
              onChange={(e) => setSubName(e.target.value)}
            />
            <Input
              className="flex-1 min-w-[200px]"
              placeholder="订阅 URL（支持 proxy-provider 的 http 链接）"
              value={subUrl}
              onChange={(e) => setSubUrl(e.target.value)}
            />
            <Button onClick={addSubscription} disabled={subLoading || !subName.trim() || !subUrl.trim()}>
              {subLoading ? <Loader2 className="size-4 animate-spin" /> : <Plus className="size-4" />}
              添加
            </Button>
          </div>
          <ul className="space-y-2">
            {(Array.isArray(subscriptions) ? subscriptions : []).map((sub) => (
              <li
                key={sub.id}
                className="flex items-center justify-between rounded border bg-muted/30 px-3 py-2 text-sm"
              >
                <span className="font-medium">{sub.name}</span>
                <code className="max-w-[50%] truncate text-muted-foreground">{sub.url}</code>
                <Button variant="ghost" size="sm" onClick={() => deleteSubscription(sub.id)} aria-label="删除">
                  <Trash2 className="size-4 text-destructive" />
                </Button>
              </li>
            ))}
            {(!Array.isArray(subscriptions) || subscriptions.length === 0) && (
              <li className="text-sm text-muted-foreground">暂无订阅，请添加后点击「应用订阅到配置」</li>
            )}
          </ul>
        </CardContent>
      </Card>

      {/* 节点选择（需运行中且 config 开启 external-controller） */}
      {status.running && (
        <Card className="overflow-hidden">
          <CardHeader className="flex flex-row items-center justify-between gap-4">
            <CardTitle className="text-lg">节点选择</CardTitle>
            <Button variant="outline" size="sm" onClick={() => { fetchProxies(); fetchMode() }} disabled={proxiesLoading}>
              {proxiesLoading ? <Loader2 className="size-4 animate-spin" /> : '刷新'}
            </Button>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* 模式切换 */}
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium shrink-0">运行模式</span>
              <div className="flex rounded-lg border overflow-hidden">
                {(['rule', 'global', 'direct'] as const).map((m) => (
                  <button
                    key={m}
                    onClick={() => switchMode(m)}
                    disabled={modeLoading}
                    className={[
                      'px-4 py-1.5 text-sm transition-colors',
                      mode === m
                        ? 'bg-primary text-primary-foreground font-medium'
                        : 'bg-background text-muted-foreground hover:bg-muted',
                    ].join(' ')}
                  >
                    {m === 'rule' ? '规则' : m === 'global' ? '全局' : '直连'}
                  </button>
                ))}
              </div>
              {modeLoading && <Loader2 className="size-4 animate-spin text-muted-foreground" />}
              <span className="text-xs text-muted-foreground">
                {mode === 'global' ? '全部流量走选定节点' : mode === 'direct' ? '全部直连，不走代理' : '按规则分流'}
              </span>
            </div>

            {/* Selector 组节点选择（rule / global 模式有效） */}
            {selectorGroups && selectorGroups.length > 0 ? (
              <div className="space-y-3">
                {selectorGroups.map(([groupName, proxy]) => (
                  <div key={groupName} className="flex items-center gap-3">
                    <Label className="w-24 shrink-0">{groupName}</Label>
                    <select
                      className="max-w-xs rounded-md border bg-background px-3 py-2 text-sm"
                      value={proxy.now || ''}
                      onChange={(e) => setProxy(groupName, e.target.value)}
                      disabled={!!proxySelectLoading}
                    >
                      {(proxy.all || []).map((name) => (
                        <option key={name} value={name}>
                          {name}
                        </option>
                      ))}
                    </select>
                    {proxySelectLoading === groupName && <Loader2 className="size-4 animate-spin" />}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                {proxiesLoading ? '加载中…' : '无可选节点组'}
              </p>
            )}
          </CardContent>
        </Card>
      )}

      <Card className="overflow-hidden">
        <CardHeader className="flex flex-row items-center justify-between gap-4">
          <CardTitle className="text-lg">配置 (config.yaml)</CardTitle>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={fetchConfig} disabled={configLoading}>
              {configLoading ? <Loader2 className="size-4 animate-spin" /> : '刷新'}
            </Button>
            <Button size="sm" onClick={saveConfig} disabled={configSaving}>
              {configSaving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
              保存
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          {configMessage && (
            <p className="text-sm text-muted-foreground">{configMessage}</p>
          )}
          <textarea
            className="w-full min-h-[280px] rounded-lg border bg-muted/30 p-3 font-mono text-sm"
            value={configYaml}
            onChange={(e) => setConfigYaml(e.target.value)}
            placeholder="YAML 配置..."
            spellCheck={false}
          />
        </CardContent>
      </Card>
    </div>
  )
}
