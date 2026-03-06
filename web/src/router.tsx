import { createBrowserRouter, Navigate } from 'react-router-dom'
import { Layout } from '@/components/Layout'
import Dashboard from '@/pages/Dashboard'
import Script from '@/pages/Script'
import Mihomo from '@/pages/Mihomo'
import Agent from '@/pages/Agent'
import Skills from '@/pages/Skills'
import Config from '@/pages/Config'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: <Dashboard /> },
      { path: 'script', element: <Script /> },
      { path: 'mihomo', element: <Mihomo /> },
      { path: 'agent', element: <Agent /> },
      { path: 'skills', element: <Skills /> },
      { path: 'config', element: <Config /> },
    ],
  },
])
