import { useEffect, useState } from 'react'
import { api } from '@/api/client'
import { Bot } from 'lucide-react'
import { useAgentChat, SkillsBar, ChatBody } from '@/components/AgentFloatingPanel'

export default function Agent() {
  const [state, setState] = useState<{ workspaceDir: string; lastActivity?: string } | null>(null)
  const chat = useAgentChat('agent')

  useEffect(() => {
    api.get('/agent/state').then((r) => setState(r.data)).catch(() => {})
  }, [])

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)] max-h-[calc(100vh-4rem)]">
      {/* 标题 */}
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

      {/* 对话区域 */}
      <div className="flex-1 flex flex-col overflow-hidden rounded-xl border border-border bg-card shadow-sm">
        {/* Skills 选择条 */}
        <SkillsBar
          visibleSkills={chat.visibleSkills}
          activeSkills={chat.activeSkills}
          toggleSkill={chat.toggleSkill}
        />

        {/* 消息 + 输入 */}
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
  )
}
