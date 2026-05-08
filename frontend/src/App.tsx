import { useAuthStore } from '@/store/auth'
import { Login } from '@/pages/Login'
import { Dashboard } from '@/pages/Dashboard'

export function App() {
  const autenticado = useAuthStore((s) => !!s.token)
  const hydrated = useAuthStore((s) => s._hydrated)
  if (!hydrated) return null
  return autenticado ? <Dashboard /> : <Login />
}
