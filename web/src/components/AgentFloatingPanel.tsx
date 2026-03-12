import { useCallback, useEffect, useRef, useState } from "react"
import { useLocation } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { getConfig } from "@/config"
import { useAgentFormFill } from "@/contexts/AgentFormFillContext"
import type { AgentFormFillAction, Conversation } from "@/api/client"
import { api } from "@/api/client"
import { Bot, Send, Minimize2, Zap } from "lucide-react"
import ReactMarkdown from "react-markdown"
import remarkBreaks from "remark-breaks"
import remarkGfm from "remark-gfm"
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

// 预处理：将字面量 \n 转为真实换行，确保 Markdown 正确解析
function normalizeNewlines(s: string): string {
  return s.replace(/\\n/g, "\n")
}

// Markdown 渲染用的自定义组件（提取到外部避免 JSX 解析歧义）
const markdownComponents = {
  pre: ({ children }: { children?: React.ReactNode }) => (
    <pre className="!my-2 overflow-x-auto rounded-md bg-muted/80 p-2 text-xs text-foreground">{children}</pre>
  ),
  code: ({ className, children, ...props }: { className?: string; children?: React.ReactNode }) =>
    className ? (
      <code className={cn(className, "text-foreground")} {...props}>{children}</code>
    ) : (
      <code className="rounded bg-muted/80 px-1 py-0.5 text-xs text-foreground" {...props}>{children}</code>
    ),
  // 标题使用深色、加粗，避免灰色
  h1: ({ children }: { children?: React.ReactNode }) => (
    <h1 className="text-lg font-bold text-foreground mt-3 mb-1 first:mt-0">{children}</h1>
  ),
  h2: ({ children }: { children?: React.ReactNode }) => (
    <h2 className="text-base font-semibold text-foreground mt-3 mb-1 first:mt-0">{children}</h2>
  ),
  h3: ({ children }: { children?: React.ReactNode }) => (
    <h3 className="text-sm font-semibold text-foreground mt-2 mb-1 first:mt-0">{children}</h3>
  ),
  p: ({ children }: { children?: React.ReactNode }) => (
    <p className="text-foreground my-1.5 [&:first-child]:mt-0 [&:last-child]:mb-0">{children}</p>
  ),
  li: ({ children }: { children?: React.ReactNode }) => (
    <li className="text-foreground">{children}</li>
  ),
  strong: ({ children }: { children?: React.ReactNode }) => (
    <strong className="font-semibold text-primary">{children}</strong>
  ),
}

// MarkdownContent：将 LLM 输出即时渲染为 Markdown（流式输出时也会实时更新）
// remark-breaks: 单换行转为 <br>；remark-gfm: 表格、删除线等
function MarkdownContent({ content }: { content: string }) {
  const normalized = normalizeNewlines(content)
  if (!normalized.trim()) return null
  return (
    <div className="prose prose-sm max-w-none dark:prose-invert prose-headings:text-foreground prose-p:text-foreground prose-li:text-foreground [&>*:first-child]:mt-0 [&>*:last-child]:mb-0">
      <ReactMarkdown remarkPlugins={[remarkBreaks, remarkGfm]} components={markdownComponents}>
        {normalized}
      </ReactMarkdown>
    </div>
  )
}

// 核心对话逻辑：可复用于浮窗和 Agent 全页面（含会话持久化）
export function useAgentChat(currentPage: string) {
  const { setFormFill } = useAgentFormFill()
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState("")
  const [streaming, setStreaming] = useState(false)
  const [streamingContent, setStreamingContent] = useState("")
  const [skills, setSkills] = useState<Skill[]>([])
  const [activeSkills, setActiveSkills] = useState<Set<string>>(new Set())
  const [sessionId, setSessionId] = useState<string>("")
  const [sessions, setSessions] = useState<Conversation[]>([])
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)

  const loadSessions = useCallback(() => {
    api.get<Conversation[]>("/agent/sessions")
      .then((r) => setSessions(Array.isArray(r.data) ? r.data : []))
      .catch(() => setSessions([]))
  }, [])

  const selectSession = useCallback((id: string) => {
    if (!id) return
    api.get<Conversation>(`/agent/sessions/${id}`)
      .then((r) => {
        const msgs = (r.data?.messages ?? []).map((m) => ({
          role: m.role as "user" | "assistant",
          content: m.content,
        }))
        setMessages(msgs)
        setSessionId(id)
      })
      .catch(() => {})
  }, [])

  const startNewSession = useCallback(() => {
    setSessionId("")
    setMessages([])
    import("sonner").then(({ toast }) => toast.success("已开始新会话"))
  }, [])

  const deleteSession = useCallback((id: string) => {
    api.delete(`/agent/sessions/${id}`).then(() => {
      loadSessions()
      setSessionId((prev) => {
        if (prev === id) {
          setMessages([])
          return ""
        }
        return prev
      })
    }).catch(() => {})
  }, [loadSessions])

  // 只 fetch 一次（mount 时）；切页无需重新拉取，filter 实时计算
  useEffect(() => {
    api.get<Skill[]>("/agent/skills")
      .then((r) => {
        const list = Array.isArray(r.data) ? r.data : []
        setSkills(list)
      })
      .catch(() => {})
  }, [])

  useEffect(() => {
    loadSessions()
  }, [loadSessions])

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
          sessionId: sessionId || undefined,
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
        } else if (event === "session") {
          setSessionId(data)
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
      loadSessions()
    }
  }, [input, streaming, messages, currentPage, activeSkills, sessionId, setFormFill, scrollToBottom, loadSessions])

  return {
    messages, input, setInput, streaming, streamingContent,
    skills, visibleSkills, activeSkills, toggleSkill,
    sessionId, sessions, loadSessions, selectSession, startNewSession, deleteSession,
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
  messagesEndRef: React.RefObject<HTMLDivElement | null>
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
            {m.role === "user" ? (
              <span className="whitespace-pre-wrap">{m.content}</span>
            ) : (
              <MarkdownContent content={m.content} />
            )}
          </div>
        ))}
        {streaming && streamingContent && (
          <div className="mr-auto rounded-lg bg-muted px-3 py-2 text-sm max-w-[95%]">
            <MarkdownContent content={stripFormFillBlock(streamingContent)} />
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
