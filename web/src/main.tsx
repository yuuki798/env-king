import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import { Toaster } from 'sonner'
import { AgentFormFillProvider } from '@/contexts/AgentFormFillContext'
import { router } from './router'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AgentFormFillProvider>
      <RouterProvider router={router} />
      <Toaster richColors position="top-center" />
    </AgentFormFillProvider>
  </StrictMode>,
)
