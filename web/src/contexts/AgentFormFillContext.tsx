import { createContext, useCallback, useContext, useState } from "react"
import type { AgentFormFillAction } from "@/api/client"

type AgentFormFillContextValue = {
  formFill: AgentFormFillAction | null
  setFormFill: (action: AgentFormFillAction | null) => void
  clearFormFill: () => void
}

const AgentFormFillContext = createContext<AgentFormFillContextValue | null>(null)

export function AgentFormFillProvider({ children }: { children: React.ReactNode }) {
  const [formFill, setFormFillState] = useState<AgentFormFillAction | null>(null)
  const setFormFill = useCallback((action: AgentFormFillAction | null) => {
    setFormFillState(action)
  }, [])
  const clearFormFill = useCallback(() => setFormFillState(null), [])
  return (
    <AgentFormFillContext.Provider value={{ formFill, setFormFill, clearFormFill }}>
      {children}
    </AgentFormFillContext.Provider>
  )
}

export function useAgentFormFill() {
  const ctx = useContext(AgentFormFillContext)
  if (!ctx) {
    return {
      formFill: null,
      setFormFill: () => {},
      clearFormFill: () => {},
    }
  }
  return ctx
}
