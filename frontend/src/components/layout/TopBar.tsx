import { useState } from 'react'
import { Search, RefreshCw, MoreHorizontal, Power, Pencil, Trash2, Copy } from 'lucide-react'
import type { Connection, DatabaseCategory } from '../../types/database'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import * as api from '../../api'
import toast from 'react-hot-toast'

interface TopBarProps {
  connection: Connection
  category: DatabaseCategory
  onConnectionsChanged: () => void
}

const engineLabel: Record<string, string> = {
  postgresql: 'PostgreSQL',
  mysql: 'MySQL',
  mongodb: 'MongoDB',
  documentdb: 'DocumentDB',
  redis: 'Redis',
  memcached: 'Memcached',
}

export function TopBar({ connection, onConnectionsChanged }: TopBarProps) {
  const [showMenu, setShowMenu] = useState(false)
  const [showDisconnect, setShowDisconnect] = useState(false)
  const [showDelete, setShowDelete] = useState(false)

  const handleDisconnect = async () => {
    try {
      await api.disconnectFromDatabase(connection.id)
      onConnectionsChanged()
      toast.success('Disconnected')
    } catch (err: any) {
      toast.error(err.message || 'Failed to disconnect')
    }
  }

  const handleDelete = async () => {
    try {
      await api.deleteConnection(connection.id)
      onConnectionsChanged()
      toast.success('Deleted')
    } catch (err: any) {
      toast.error(err.message || 'Failed to delete')
    }
  }

  return (
    <>
      <header className="h-11 bg-surface-1 border-b border-border-dim flex items-center px-4 gap-3 shrink-0">
        {/* Connection info */}
        <div className="flex items-center gap-2.5 min-w-0">
          <span className={`w-1.5 h-1.5 rounded-full ${
            connection.status === 'connected' ? 'bg-status-ok' :
            connection.status === 'error' ? 'bg-status-error' : 'bg-status-idle'
          }`} />
          <span className="text-[13px] font-medium text-text-primary truncate">{connection.name}</span>
          <span className="text-[11px] text-text-muted">{engineLabel[connection.engine]}</span>
          <span className="text-[11px] text-text-muted font-mono">{connection.host}:{connection.port}</span>
        </div>

        {/* Spacer */}
        <div className="flex-1" />

        {/* Search */}
        <div className="flex items-center gap-1.5 px-2.5 py-1 text-[12px] text-text-muted bg-surface-2 border border-border-dim rounded-md cursor-pointer hover:border-border-base transition-colors">
          <Search size={12} />
          <span>Search</span>
          <kbd className="ml-3 text-[10px] font-mono opacity-50">{'\u2318'}K</kbd>
        </div>

        {/* Actions */}
        <button onClick={() => toast.success('Refreshed')} className="p-1 rounded-md text-text-muted hover:text-text-secondary transition-colors">
          <RefreshCw size={14} />
        </button>

        <div className="relative">
          <button onClick={() => setShowMenu(!showMenu)} className="p-1 rounded-md text-text-muted hover:text-text-secondary transition-colors">
            <MoreHorizontal size={14} />
          </button>
          {showMenu && (
            <>
              <div className="fixed inset-0 z-40" onClick={() => setShowMenu(false)} />
              <div className="absolute right-0 top-full mt-1.5 w-44 bg-surface-2 border border-border-base rounded-lg shadow-2xl shadow-black/40 py-1 z-50">
                <button className="w-full flex items-center gap-2 px-3 py-1.5 text-[13px] text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors" onClick={() => { setShowMenu(false); toast.success('Copied') }}>
                  <Copy size={13} /> Copy URI
                </button>
                <button className="w-full flex items-center gap-2 px-3 py-1.5 text-[13px] text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors" onClick={() => { setShowMenu(false) }}>
                  <Pencil size={13} /> Edit
                </button>
                <div className="my-1 border-t border-border-dim" />
                <button className="w-full flex items-center gap-2 px-3 py-1.5 text-[13px] text-yellow-400/80 hover:text-yellow-400 hover:bg-yellow-500/5 transition-colors" onClick={() => { setShowMenu(false); setShowDisconnect(true) }}>
                  <Power size={13} /> Disconnect
                </button>
                <button className="w-full flex items-center gap-2 px-3 py-1.5 text-[13px] text-red-400/80 hover:text-red-400 hover:bg-red-500/5 transition-colors" onClick={() => { setShowMenu(false); setShowDelete(true) }}>
                  <Trash2 size={13} /> Delete
                </button>
              </div>
            </>
          )}
        </div>
      </header>

      <ConfirmDialog open={showDisconnect} onClose={() => setShowDisconnect(false)} onConfirm={handleDisconnect} title="Disconnect" message={`Disconnect from ${connection.name}? Active queries will be terminated.`} confirmLabel="Disconnect" variant="warning" />
      <ConfirmDialog open={showDelete} onClose={() => setShowDelete(false)} onConfirm={handleDelete} title="Delete connection" message={`Permanently delete "${connection.name}"?`} confirmLabel="Delete" variant="danger" />
    </>
  )
}
