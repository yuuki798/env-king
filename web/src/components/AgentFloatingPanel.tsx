import { useCallback, useEffect, useRef, useState } from "react"
import { useLocation } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { getConfig } from "@/config"
import { useAgentFormFill } from "@/contexts/AgentFormFillContext"
import type { AgentFormFillAction } from "@/api/client"
import { api } from "@/api/client"
import { Bot, Send, Minimize2, Zap } from "lucide-react"
import { cn } from "@/lib/utils"

type ChatMessage = { role: "user" | "assistant"; content: string }

type Skill = {
  id: string
  name: string
  description: string
  pages: string[]
}

function stripFormFillBlock(text: string): string {
  const idx = text.indexOf("[FORM_FILL:")
  if (idx < 0) return text.trim()
  return text.slice(0, idx).trim()
}

function parseSSE(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  onEvent: (event: string, data: string) => void
): Promise<void> {
  const decoder = new TextDecoder()
  let buffer = ""
  function pump(chunk: ReadableStreamReadResult<Uint8Array>): Promise<void> {
    if (chunk.done) return Promise.resolve()
    buffer += decoder.decode(chunk.value, { stream: true })
    const parts = buffer.split(/\n\n/)
    buffer = parts.pop() ?? ""
    for (const part of parts) {
      let event = "message"
      const dataLines: string[] = []
      for (const line of part.split(/\n/)) {
        if (line.startsWith("event:")) event = line.slice(6).trim()
        else if (line.startsWith("data:")) dataLines.push(line.slice(5))
      }
      const data = dataLines.join("\n").trim()
      onEvent(event, data)
    }
    return reader.read().then(pump)
  }
  return reader.read().then(pump)
}

// 核心对话逻辑：可复用于浮窗和 Agent 全页面
export function useAgentChat(currentPage: string) {
  const { setFormFill } = useAgentFormFill()
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState("")
  const [streaming, setStreaming] = useState(false)
  const [streamingContent, setStreamingContent] = useState("")
  const [skills, setSkills] = useState<Skill[]>([])
  const [activeSkills, setActiveSkills] = useState<Set<string>>(new Set())
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  // 只 fetch 一次（mount 时）；切页无需重新拉取，filter 实时计算
  useEffect(() => {
    api.get<Skill[]>("/agent/skills")
      .then((r) => {
        const list = Array.isArray(r.data) ? r.data : []
        setSkills(list)
      })
      .catch(() => {})
  }, [])

  // 当页面切换或 skills 加载完成时，自动激活当前页面的 skills
  useEffect(() => {
    if (skills.length === 0) return
    const auto = new Set<string>()
    skills.forEach((s) => {
      if (!s.pages || s.pages.length === 0) return
      if (s.pages.includes(currentPage)) auto.add(s.id)
    })
    setActiveSkills(auto)
  }, [currentPage, skills])

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [])

  const toggleSkill = useCallback((id: string) => {
    setActiveSkills((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  // 过滤出与当前页面相关（或全局）的 skills 供显示
  const allSkills = Array.isArray(skills) ? skills : []
  const visibleSkills = allSkills.filter(
    (s) => !s.pages || s.pages.length === 0 || s.pages.includes(currentPage)
  )

  const sendMessage = useCallback(async () => {
    const text = input.trim()
    if (!text || streaming) return
    setInput("")
    setMessages((prev) => [...prev, { role: "user", content: text }])
    setStreaming(true)
    setStreamingContent("")

    const history = messages.map((m) => ({ role: m.role, content: m.content }))
    const base = getConfig().apiBaseUrl
    const url = `${base}/agent/chat/stream`
    abortRef.current = new AbortController()
    try {
      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          message: text,
          currentPage,
          activeSkills: Array.from(activeSkills),
          history,
        }),
        signal: abortRef.current.signal,
      })
      if (!res.ok || !res.body) {
        setMessages((prev) => [
          ...prev,
          { role: "assistant", content: `请求失败: ${res.status}` },
        ])
        return
      }
      let fullContent = ""
      await parseSSE(res.body.getReader(), (event, data) => {
        if (event === "message") {
          fullContent += data
          setStreamingContent(fullContent)
        } else if (event === "form_fill") {
          try {
            const action = JSON.parse(data) as AgentFormFillAction
            setFormFill(action)
          } catch {
            // ignore
          }
        } else if (event === "error") {
          fullContent += `\n[错误] ${data}`
          setStreamingContent(fullContent)
        }
      })
      const displayContent = stripFormFillBlock(fullContent || streamingContent) || "(无回复)"
      setMessages((prev) => [...prev, { role: "assistant", content: displayContent }])
      setStreamingContent("")
    } catch (e) {
      if ((e as Error).name === "AbortError") return
      setMessages((prev) => [
        ...prev,
        { role: "assistant", content: `请求异常: ${(e as Error).message}` },
      ])
      setStreamingContent("")
    } finally {
      setStreaming(false)
      abortRef.current = null
      scrollToBottom()
    }
  }, [input, streaming, messages, currentPage, activeSkills, setFormFill, scrollToBottom])

  return {
    messages, input, setInput, streaming, streamingContent,
    skills, visibleSkills, activeSkills, toggleSkill,
    sendMessage, scrollToBottom, messagesEndRef,
  }
}

// SkillsBar：技能选择条
export function SkillsBar({ visibleSkills, activeSkills, toggleSkill }: {
  visibleSkills: Skill[]
  activeSkills: Set<string>
  toggleSkill: (id: string) => void
}) {
  if (visibleSkills.length === 0) return null
  return (
    <div className="flex flex-wrap gap-1.5 px-3 py-2 border-b border-border bg-muted/30">
      {visibleSkills.map((s) => (
        <button
          key={s.id}
          onClick={() => toggleSkill(s.id)}
          title={s.description}
          className={cn(
            "flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs transition-colors",
            activeSkills.has(s.id)
              ? "border-primary bg-primary text-primary-foreground"
              : "border-border bg-background text-muted-foreground hover:border-primary hover:text-foreground"
          )}
        >
          <Zap className="size-3" />
          {s.name}
        </button>
      ))}
    </div>
  )
}

// ChatBody：消息列表 + 输入框（通用）
export function ChatBody({
  messages, streaming, streamingContent, input, setInput, sendMessage, messagesEndRef,
  emptyHint,
}: {
  messages: ChatMessage[]
  streaming: boolean
  streamingContent: string
  input: string
  setInput: (v: string) => void
  sendMessage: () => void
  messagesEndRef: React.RefObject<HTMLDivElement>
  emptyHint?: string
}) {
  return (
    <>
      <div className="flex-1 overflow-y-auto p-3 space-y-3">
        {messages.length === 0 && !streaming && (
          <p className="text-sm text-muted-foreground">
            {emptyHint ?? "你好，点亮上方技能后与我对话，我会根据技能能力帮你完成任务。"}
          </p>
        )}
        {messages.map((m, i) => (
          <div
            key={i}
            className={cn(
              "rounded-lg px-3 py-2 text-sm max-w-[95%]",
              m.role === "user"
                ? "ml-auto bg-primary text-primary-foreground"
                : "mr-auto bg-muted"
            )}
          >
            {m.content}
          </div>
        ))}
        {streaming && streamingContent && (
          <div className="mr-auto rounded-lg bg-muted px-3 py-2 text-sm max-w-[95%] whitespace-pre-wrap">
            {stripFormFillBlock(streamingContent)}
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>
      <div className="border-t border-border p-2">
        <div className="flex gap-2">
          <Textarea
            placeholder="输入消息…"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault()
                sendMessage()
              }
            }}
            rows={2}
            className="min-h-0 resize-none"
            disabled={streaming}
          />
          <Button
            size="icon"
            className="shrink-0 h-auto"
            onClick={sendMessage}
            disabled={streaming || !input.trim()}
            aria-label="发送"
          >
            <Send className="size-4" />
          </Button>
        </div>
      </div>
    </>
  )
}

// AgentFloatingPanel：右下角浮窗，在 /agent 页面不显示
export function AgentFloatingPanel() {
  const location = useLocation()
  const [open, setOpen] = useState(false)

  // Agent 页面本身已经是完整 agent 界面，不需要浮窗
  if (location.pathname === "/agent") return null

  const currentPage = location.pathname.replace(/^\//, "") || "dashboard"

  return <FloatingPanelInner open={open} setOpen={setOpen} currentPage={currentPage} />
}

function FloatingPanelInner({
  open,
  setOpen,
  currentPage,
}: {
  open: boolean
  setOpen: (v: boolean) => void
  currentPage: string
}) {
  const chat = useAgentChat(currentPage)

  return (
    <>
      <div
        className={cn(
          "fixed bottom-6 right-6 z-50 flex flex-col rounded-xl border border-border bg-card shadow-lg transition-all duration-200",
          open ? "h-[480px] w-[400px]" : "h-0 w-0 overflow-hidden border-0 opacity-0"
        )}
      >
        <div className="flex items-center justify-between border-b border-border px-3 py-2 shrink-0">
          <span className="flex items-center gap-2 text-sm font-medium">
            <Bot className="size-4" />
            智能助理
          </span>
          <Button
            variant="ghost"
            size="icon"
            className="size-8"
            onClick={() => setOpen(false)}
            aria-label="收起"
          >
            <Minimize2 className="size-4" />
          </Button>
        </div>
        <SkillsBar
          visibleSkills={chat.visibleSkills}
          activeSkills={chat.activeSkills}
          toggleSkill={chat.toggleSkill}
        />
        <ChatBody
          messages={chat.messages}
          streaming={chat.streaming}
          streamingContent={chat.streamingContent}
          input={chat.input}
          setInput={chat.setInput}
          sendMessage={chat.sendMessage}
          messagesEndRef={chat.messagesEndRef}
        />
      </div>
      {!open && (
        <Button
          size="icon"
          className="fixed bottom-6 right-6 z-50 size-12 rounded-full shadow-lg"
          onClick={() => setOpen(true)}
          aria-label="打开助理"
        >
          <Bot className="size-5" />
        </Button>
      )}
    </>
  )
}
