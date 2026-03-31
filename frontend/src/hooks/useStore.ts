import { useState, useCallback, useEffect } from 'react'
import type { Connection, DatabaseCategory } from '../types/database'
import * as api from '../api'

export function useStore() {
  const [connections, setConnections] = useState<Connection[]>([])
  const [activeConnectionId, setActiveConnectionId] = useState<string>('')
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)

  const refreshConnections = useCallback(async () => {
    try {
      const list = await api.listConnections()
      setConnections(list || [])
    } catch (e) {
      console.error('Failed to load connections:', e)
    }
  }, [])

  useEffect(() => { refreshConnections() }, [refreshConnections])

  const activeConnection = connections.find(c => c.id === activeConnectionId) ?? connections[0]
  const activeCategory: DatabaseCategory = activeConnection?.category ?? 'sql'

  const toggleSidebar = useCallback(() => setSidebarCollapsed(v => !v), [])

  return {
    connections,
    activeConnection,
    activeConnectionId,
    setActiveConnectionId,
    activeCategory,
    sidebarCollapsed,
    toggleSidebar,
    refreshConnections,
  }
}
