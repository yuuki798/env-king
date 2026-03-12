import axios from "axios"
import { getConfig } from "@/config"

export const api = axios.create({
  baseURL: getConfig().apiBaseUrl,
  timeout: 30000,
  headers: { "Content-Type": "application/json" },
})

// --- Script / Workflow ---
export interface ScriptStep {
  name: string
  kind: string
  command: string
}

export interface ScriptJobView {
  id: string
  topic: string
  status: string
  error?: string
  logs?: string
  workDir?: string
  meta?: Record<string, string>
  startedAt?: string
  finishedAt?: string
  enqueued_at?: string
}

export interface Workflow {
  id: string
  name: string
  workDir?: string
  steps: ScriptStep[]
  defaultInputs?: Record<string, string>
  createdAt?: string
  updatedAt?: string
}

export interface PresetNodeDef {
  kind: string
  label: string
  commandTemplate: string
  params: { key: string; label: string; type: string; default?: string }[]
}

export interface MihomoStatus {
  running: boolean
  proxyPort?: number
  mixedPort?: number
}

export interface AgentState {
  workspaceDir: string
  cronJobs: string[]
  lastActivity?: string
}

export interface CronJob {
  name: string
  spec: string
  script: string
}

export interface ModelEndpoint {
  name: string
  url: string
  apiKey?: string
  weight: number
}

export interface LoadBalanceConfig {
  models: ModelEndpoint[]
}

// Agent 流式对话 + 自动填表
export interface AgentFormFillAction {
  page: string
  form: string
  payload: Record<string, unknown>
}

// Agent 持久化会话
export interface ConversationMessage {
  role: string
  content: string
  createdAt?: string
}

export interface Conversation {
  id: string
  title: string
  createdAt: string
  updatedAt: string
  messages: ConversationMessage[]
}

// Agent 记忆（openclaw 双来源）
export interface MemorySearchResult {
  path: string
  startLine: number
  endLine: number
  score: number
  snippet: string
  source: "memory" | "sessions"
}

export interface MemorySearchResponse {
  items: MemorySearchResult[]
  disabled?: boolean
  reason?: string
  error?: string
}

export interface MemoryFileResponse {
  text: string
  path: string
  disabled?: boolean
  reason?: string
  error?: string
}
