import axios from "axios"
import { getConfig } from "@/config"

export const api = axios.create({
  baseURL: getConfig().apiBaseUrl,
  timeout: 30000,
  headers: { "Content-Type": "application/json" },
})

export interface PipelineJob {
  id: string
  repo: string
  branch: string
  status: "pending" | "building" | "pushing" | "success" | "failed"
  imageTag?: string
  harborRepo?: string
  startedAt?: string
  finishedAt?: string
  logs?: string
}

export interface ClashStatus {
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
