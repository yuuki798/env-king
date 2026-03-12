import { useEffect, useState } from 'react'
import { api, type MemorySearchResult, type MemorySearchResponse, type MemoryFileResponse } from '@/api/client'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ChevronDown, ChevronUp, FileText, Loader2, Search } from 'lucide-react'
import { cn } from '@/lib/utils'

const SOURCE_MEMORY = 'memory'
const MAX_RESULTS = 10

function shortPath(path: string): string {
  const parts = path.split(/[/\\]/)
  return parts[parts.length - 1] || path
}

export function AgentMemoryPanel() {
  const [open, setOpen] = useState(false)
  const [activeTab] = useState<'memory'>('memory')
  const [searchInput, setSearchInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [disabled, setDisabled] = useState(false)
  const [reason, setReason] = useState<string | null>(null)
  const [items, setItems] = useState<MemorySearchResult[]>([])
  const [selected, setSelected] = useState<MemorySearchResult | null>(null)
  const [detail, setDetail] = useState<{ text: string; path: string } | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const source = SOURCE_MEMORY

  const fetchSearch = () => {
    setLoading(true)
    setReason(null)
    api
      .get<MemorySearchResponse>('/agent/memory/search', {
        params: { query: searchInput.trim() || undefined, source, maxResults: MAX_RESULTS },
      })
      .then((r) => {
        const data = r.data
        if (data.disabled) {
          setDisabled(true)
          setReason(data.reason ?? '记忆未启用')
          setItems([])
        } else {
          setDisabled(false)
          setItems(data.items ?? [])
        }
      })
      .catch((err) => {
        setItems([])
        setReason(err?.response?.data?.error ?? err?.message ?? '请求失败')
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    if (!open) return
    setSelected(null)
    setDetail(null)
    fetchSearch()
  }, [open, activeTab])

  const handleSearch = () => {
    fetchSearch()
  }

  const fetchFile = (item: MemorySearchResult) => {
    setSelected(item)
    setDetail(null)
    setDetailLoading(true)
    api
      .get<MemoryFileResponse>('/agent/memory/file', {
        params: {
          path: item.path,
          from: item.startLine,
          lines: Math.max(20, item.endLine - item.startLine + 1),
        },
      })
      .then((r) => {
        const data = r.data
        if (data.disabled || data.error) {
          setDetail({ text: data.reason ?? data.error ?? '无法加载', path: item.path })
        } else {
          setDetail({ text: data.text, path: data.path })
        }
      })
      .catch(() => setDetail({ text: '加载失败', path: item.path }))
      .finally(() => setDetailLoading(false))
  }

  return (
    <Card className="overflow-hidden">
      <CardHeader
        className="cursor-pointer select-none py-3 px-4 flex flex-row items-center justify-between border-b"
        onClick={() => setOpen((o) => !o)}
      >
        <span className="text-sm font-medium flex items-center gap-2">
          <FileText className="size-4 text-muted-foreground" />
          记忆
        </span>
        {open ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" />}
      </CardHeader>
      {open && (
        <CardContent className="p-0">
          <Tabs value={activeTab} onValueChange={() => {}} className="w-full">
            <div className="px-3 pt-2">
              <TabsList className="w-full grid grid-cols-1">
                <TabsTrigger value="memory">长期记忆</TabsTrigger>
              </TabsList>
            </div>
            <div className="px-3 pt-2 flex gap-2">
              <Input
                placeholder="搜索记忆（可选）"
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                className="flex-1"
              />
              <Button type="button" variant="secondary" size="icon" onClick={handleSearch} disabled={loading}>
                {loading ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4" />}
              </Button>
            </div>
            <TabsContent value="memory" className="mt-0 px-3 pb-3">
              <MemoryList
                items={items}
                loading={loading}
                disabled={disabled}
                reason={reason}
                selected={selected}
                detail={detail}
                detailLoading={detailLoading}
                onSelect={fetchFile}
              />
            </TabsContent>
          </Tabs>
        </CardContent>
      )}
    </Card>
  )
}

function MemoryList({
  items,
  loading,
  disabled,
  reason,
  selected,
  detail,
  detailLoading,
  onSelect,
}: {
  items: MemorySearchResult[]
  loading: boolean
  disabled: boolean
  reason: string | null
  selected: MemorySearchResult | null
  detail: { text: string; path: string } | null
  detailLoading: boolean
  onSelect: (item: MemorySearchResult) => void
}) {
  if (disabled && reason) {
    return (
      <div className="py-6 text-center text-sm text-muted-foreground">
        {reason}
      </div>
    )
  }
  if (loading && items.length === 0) {
    return (
      <div className="py-6 flex items-center justify-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        加载中…
      </div>
    )
  }
  if (items.length === 0) {
    return (
      <div className="py-6 text-center text-sm text-muted-foreground">
        暂无长期记忆
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-2 py-2 min-h-[120px]">
      <ul className="space-y-1 overflow-y-auto max-h-[220px]">
        {items.map((item, i) => (
          <li key={`${item.path}-${item.startLine}-${i}`}>
            <button
              type="button"
              onClick={() => onSelect(item)}
              className={cn(
                'w-full text-left rounded-md border p-2 transition-colors',
                selected?.path === item.path && selected?.startLine === item.startLine
                  ? 'bg-primary/10 border-primary/30'
                  : 'hover:bg-muted/50 border-transparent'
              )}
            >
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-xs font-medium truncate" title={item.path}>
                  {shortPath(item.path)}
                </span>
                <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
                  L{item.startLine}-{item.endLine}
                </Badge>
                {item.source && (
                  <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                    {item.source === SOURCE_MEMORY ? '长期' : '短期'}
                  </Badge>
                )}
              </div>
              <p className="text-xs text-muted-foreground mt-1 line-clamp-2">{item.snippet}</p>
            </button>
          </li>
        ))}
      </ul>
      {selected && (
        <div className="mt-2 rounded-md border bg-muted/30 p-3 min-h-[80px]">
          <div className="text-xs font-medium text-muted-foreground mb-1">{selected.path}</div>
          {detailLoading ? (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="size-4 animate-spin" />
              加载详情…
            </div>
          ) : detail ? (
            <pre className="text-xs whitespace-pre-wrap break-words font-sans max-h-[180px] overflow-y-auto">
              {detail.text}
            </pre>
          ) : null}
        </div>
      )}
    </div>
  )
}
