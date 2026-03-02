import { createBrowserRouter, Navigate } from 'react-router-dom'
import { Layout } from '@/components/Layout'
import Dashboard from '@/pages/Dashboard'
import Pipeline from '@/pages/Pipeline'
import Clash from '@/pages/Clash'
import Agent from '@/pages/Agent'
import Skills from '@/pages/Skills'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: <Dashboard /> },
      { path: 'pipeline', element: <Pipeline /> },
      { path: 'clash', element: <Clash /> },
      { path: 'agent', element: <Agent /> },
      { path: 'skills', element: <Skills /> },
    ],
  },
])
