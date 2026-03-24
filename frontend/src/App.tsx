import { BrowserRouter, Routes, Route } from 'react-router'
import { AuthProvider } from '@/lib/auth'
import ProtectedRoute from '@/components/ProtectedRoute'
import Layout from '@/components/Layout'
import Login from '@/pages/Login'
import Dashboard from '@/pages/Dashboard'
import Inbox from '@/pages/Inbox'
import ThreadDetail from '@/pages/ThreadDetail'
import Keywords from '@/pages/Keywords'
import Settings from '@/pages/Settings'
import PersonaBuilder from '@/pages/PersonaBuilder'
import About from '@/pages/About'

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route element={<ProtectedRoute />}>
            <Route element={<Layout />}>
              <Route path="/" element={<Dashboard />} />
              <Route path="/inbox" element={<Inbox />} />
              <Route path="/threads/:id" element={<ThreadDetail />} />
              <Route path="/keywords" element={<Keywords />} />
              <Route path="/settings" element={<Settings />} />
              <Route path="/personas/new" element={<PersonaBuilder />} />
              <Route path="/personas/:id" element={<PersonaBuilder />} />
              <Route path="/about" element={<About />} />
            </Route>
          </Route>
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}
