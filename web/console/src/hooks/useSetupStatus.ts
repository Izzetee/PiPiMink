import { useState, useEffect, useCallback } from 'react'
import { fetchAdminStatus } from '@/api/status'
import type { AdminStatus } from '@/api/status'

interface SetupStatus {
  needsSetup: boolean
  status: AdminStatus | null
  isLoading: boolean
  refetch: () => void
}

export function useSetupStatus(): SetupStatus {
  const [status, setStatus] = useState<AdminStatus | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  const load = useCallback(() => {
    setIsLoading(true)
    fetchAdminStatus()
      .then(setStatus)
      .catch(() => setStatus(null))
      .finally(() => setIsLoading(false))
  }, [])

  useEffect(() => { load() }, [load])

  const needsSetup = status !== null && !status.adminKeyConfigured && status.modelCount === 0

  return { needsSetup, status, isLoading, refetch: load }
}
