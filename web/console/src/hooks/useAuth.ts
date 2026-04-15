import { useState, useEffect, useCallback } from 'react'
import { fetchAuthMe, logout as apiLogout } from '@/api/auth'
import type { AuthMeResponse } from '@/types/auth'

interface AuthState {
  user: AuthMeResponse['user'] | null
  isAuthenticated: boolean
  oauthEnabled: boolean
  isLoading: boolean
  logout: () => Promise<void>
}

export function useAuth(): AuthState {
  const [user, setUser] = useState<AuthMeResponse['user'] | null>(null)
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [oauthEnabled, setOauthEnabled] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    fetchAuthMe()
      .then((data) => {
        setIsAuthenticated(data.authenticated)
        setOauthEnabled(data.oauthEnabled)
        setUser(data.user ?? null)
      })
      .catch(() => {
        setIsAuthenticated(false)
        setUser(null)
      })
      .finally(() => setIsLoading(false))
  }, [])

  const logout = useCallback(async () => {
    try {
      await apiLogout()
    } catch {
      // ignore logout errors
    }
    setUser(null)
    setIsAuthenticated(false)
    if (oauthEnabled) {
      window.location.href = '/auth/login'
    }
  }, [oauthEnabled])

  return { user, isAuthenticated, oauthEnabled, isLoading, logout }
}
