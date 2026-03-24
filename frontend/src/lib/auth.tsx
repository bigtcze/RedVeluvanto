import { createContext, useContext, useEffect, useState } from 'react'
import type { RecordModel } from 'pocketbase'
import pb from './pocketbase'

interface AuthContextValue {
  user: RecordModel | null
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  isLoading: boolean
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<RecordModel | null>(
    pb.authStore.isValid ? (pb.authStore.record as RecordModel) : null
  )
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const verify = async () => {
      if (pb.authStore.isValid) {
        try {
          await pb.collection('users').authRefresh()
          setUser(pb.authStore.record as RecordModel)
        } catch {
          pb.authStore.clear()
          setUser(null)
        }
      } else {
        setUser(null)
      }
      setIsLoading(false)
    }

    void verify()

    const unsubscribe = pb.authStore.onChange(() => {
      setUser(pb.authStore.isValid ? (pb.authStore.record as RecordModel) : null)
    })

    return () => {
      unsubscribe()
    }
  }, [])

  const login = async (email: string, password: string) => {
    await pb.collection('users').authWithPassword(email, password)
    setUser(pb.authStore.record as RecordModel)
  }

  const logout = () => {
    pb.authStore.clear()
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, login, logout, isLoading }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used inside AuthProvider')
  }
  return ctx
}
