import { useEffect, useState } from 'react'
import { api } from '@/api/client'
import { Bot, MessageSquarePlus, RefreshCw, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { useAgentChat, SkillsBar, ChatBody } from '@/components/AgentFloatingPanel'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export default function Agent() {
  const [state, setState] = useState<{ workspaceDir: string; lastActivity?: string } | null>(null)
  const chat = useAgentChat('agent')

  useEffect(() => {
    api.get('/agent/state').then((r) => setState(r.data)).catch(() => {})
  }, [])

  return (
    <div className="flex h-[calc(100vh-4rem)] max-h-[calc(100vh-4rem)] gap-4">
      {/* 左侧会话列表 */}
      <div className="w-56 shrink-0 flex flex-col rounded-xl border border-border bg-card overflow-hidden">
        <div className="shrink-0 p-2 border-b border-border flex items-center justify-between gap-1">
          <span className="text-sm font-medium">会话历史</span>
          <div className="flex gap-0.5">
            <Button type="button" variant="ghost" size="icon" className="size-8" onClick={() => { chat.loadSessions(); toast.success("已刷新") }} title="刷新">
              <RefreshCw className="size-4" />
            </Button>
            <Button type="button" variant="ghost" size="icon" className="size-8" onClick={chat.startNewSession} title="新会话">
              <MessageSquarePlus className="size-4" />
            </Button>
          </div>
        </div>
        <ul className="flex-1 overflow-y-auto p-1 space-y-0.5">
          {chat.sessions.length === 0 && (
            <li className="text-xs text-muted-foreground px-2 py-3">暂无会话，发送消息后自动保存</li>
          )}
          {chat.sessions.map((s) => (
            <li key={s.id}>
              <div className="flex items-center gap-1 group">
                <button
                  type="button"
                  onClick={() => chat.selectSession(s.id)}
                  className={cn(
                    "flex-1 min-w-0 text-left px-2 py-2 rounded-md text-sm truncate",
                    chat.sessionId === s.id ? "bg-primary/10 text-primary" : "hover:bg-muted"
                  )}
                  title={s.title || s.id}
                >
                  {s.title || "未命名会话"}
                </button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-7 opacity-0 group-hover:opacity-100 shrink-0"
                  onClick={() => chat.deleteSession(s.id)}
                  title="删除"
                >
                  <Trash2 className="size-3.5" />
                </Button>
              </div>
            </li>
          ))}
        </ul>
      </div>

      {/* 右侧主区域 */}
      <div className="flex-1 flex flex-col min-w-0">
        <div className="shrink-0 mb-4">
          <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <Bot className="size-6 text-primary" />
            智能 Agent
          </h1>
          {state && (
            <p className="text-sm text-muted-foreground mt-1">
              工作目录: {state.workspaceDir}
              {state.lastActivity && ` · 最后活动: ${state.lastActivity}`}
            </p>
          )}
        </div>

        <div className="flex-1 flex flex-col overflow-hidden rounded-xl border border-border bg-card shadow-sm">
          <SkillsBar
            visibleSkills={chat.visibleSkills}
            activeSkills={chat.activeSkills}
            toggleSkill={chat.toggleSkill}
            onSaveAsDefault={chat.saveAsDefaultSkills}
          />
          <ChatBody
            messages={chat.messages}
            streaming={chat.streaming}
            streamingContent={chat.streamingContent}
            input={chat.input}
            setInput={chat.setInput}
            sendMessage={chat.sendMessage}
            messagesEndRef={chat.messagesEndRef}
            emptyHint="你好！点亮上方技能后开始对话，我会根据对应技能帮你完成任务。例如激活「创建工作流」技能后，告诉我你想自动化什么，我会填好表单。"
          />
        </div>
      </div>
    </div>
  )
}
