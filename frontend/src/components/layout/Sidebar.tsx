import { useState, useEffect, useCallback, useRef } from 'react'
import {
  Database, FileJson, HardDrive,
  ChevronDown, ChevronRight, Plus,
  PanelLeftClose, PanelLeftOpen,
  Settings, Folder, FolderPlus, Pencil, Trash2, MoreHorizontal,
  Power, PlugZap, Unplug,
} from 'lucide-react'
import type { Connection, DatabaseEngine } from '../../types/database'
import { NewConnectionModal } from '../modals/NewConnectionModal'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import * as api from '../../api'
import toast from 'react-hot-toast'

interface SidebarProps {
  connections: Connection[]
  activeConnectionId: string
  onSelect: (id: string) => void
  collapsed: boolean
  onToggle: () => void
  onConnectionsChanged: () => void
}

// --- Engine icons & colors ---
const engineMeta: Record<string, { icon: typeof Database; color: string; label: string }> = {
  postgresql:  { icon: Database,  color: '#336791', label: 'PG' },
  mysql:       { icon: Database,  color: '#00758f', label: 'MySQL' },
  mariadb:     { icon: Database,  color: '#003545', label: 'Maria' },
  cockroachdb: { icon: Database,  color: '#6933ff', label: 'CRDB' },
  sqlite:      { icon: Database,  color: '#003b57', label: 'SQLite' },
  sqlserver:   { icon: Database,  color: '#cc2927', label: 'MSSQL' },
  mongodb:     { icon: FileJson,  color: '#00ed64', label: 'Mongo' },
  documentdb:  { icon: FileJson,  color: '#527fff', label: 'DocDB' },
  redis:       { icon: HardDrive, color: '#d82c20', label: 'Redis' },
  valkey:      { icon: HardDrive, color: '#5b21b6', label: 'Valkey' },
  keydb:       { icon: HardDrive, color: '#ff6600', label: 'KeyDB' },
  dragonfly:   { icon: HardDrive, color: '#22c55e', label: 'Drgnfly' },
  memcached:   { icon: HardDrive, color: '#768a2e', label: 'Memcch' },
}

function EngineIcon({ engine, size = 13 }: { engine: DatabaseEngine; size?: number }) {
  const meta = engineMeta[engine] || { icon: Database, color: '#666' }
  const Icon = meta.icon
  return (
    <div className="w-[18px] h-[18px] rounded flex items-center justify-center shrink-0" style={{ backgroundColor: meta.color + '18' }}>
      <Icon size={size - 2} style={{ color: meta.color }} />
    </div>
  )
}

// --- Tree data types ---
interface FolderNode {
  type: 'folder'
  id: string
  name: string
  children: TreeNode[]
  expanded?: boolean
}

interface ConnectionNode {
  type: 'connection'
  connectionId: string
}

type TreeNode = FolderNode | ConnectionNode

// --- Tree helpers ---
function nodeId(n: TreeNode): string {
  return n.type === 'folder' ? `folder:${n.id}` : `conn:${n.connectionId}`
}

function removeFromTree(nodes: TreeNode[], dragId: string): TreeNode[] {
  return nodes
    .filter(n => nodeId(n) !== dragId)
    .map(n => n.type === 'folder' ? { ...n, children: removeFromTree(n.children, dragId) } : n)
}

function insertIntoFolder(nodes: TreeNode[], folderId: string, item: TreeNode): TreeNode[] {
  return nodes.map(n => {
    if (n.type === 'folder') {
      if (n.id === folderId) {
        // Avoid duplicates
        const filtered = n.children.filter(c => nodeId(c) !== nodeId(item))
        return { ...n, children: [...filtered, item], expanded: true }
      }
      return { ...n, children: insertIntoFolder(n.children, folderId, item) }
    }
    return n
  })
}

function updateFolder(nodes: TreeNode[], folderId: string, fn: (f: FolderNode) => FolderNode): TreeNode[] {
  return nodes.map(n => {
    if (n.type === 'folder') {
      if (n.id === folderId) return fn(n)
      return { ...n, children: updateFolder(n.children, folderId, fn) }
    }
    return n
  })
}

function countConnections(nodes: TreeNode[]): number {
  let count = 0
  for (const n of nodes) {
    if (n.type === 'connection') count++
    else count += countConnections(n.children)
  }
  return count
}

function getTreeConnectionIds(nodes: TreeNode[]): Set<string> {
  const ids = new Set<string>()
  for (const n of nodes) {
    if (n.type === 'connection') ids.add(n.connectionId)
    else getTreeConnectionIds(n.children).forEach(id => ids.add(id))
  }
  return ids
}

function findNode(nodes: TreeNode[], dragId: string): TreeNode | null {
  for (const n of nodes) {
    if (nodeId(n) === dragId) return n
    if (n.type === 'folder') {
      const found = findNode(n.children, dragId)
      if (found) return found
    }
  }
  return null
}

// --- Inline rename ---
function InlineRename({ value, onConfirm, onCancel }: { value: string; onConfirm: (v: string) => void; onCancel: () => void }) {
  const [text, setText] = useState(value)
  const ref = useRef<HTMLInputElement>(null)
  useEffect(() => { ref.current?.focus(); ref.current?.select() }, [])
  return (
    <input ref={ref} value={text} onChange={e => setText(e.target.value)}
      onKeyDown={e => { if (e.key === 'Enter') onConfirm(text); if (e.key === 'Escape') onCancel() }}
      onBlur={() => onConfirm(text)}
      className="flex-1 min-w-0 px-1 py-0 text-[12px] bg-surface-3 border border-border-focus rounded text-text-primary outline-none font-mono" />
  )
}

// --- Folder node ---
function FolderNodeView({ folder, depth, connections, activeConnectionId, onSelect, tree, onCommit, onConnectionsChanged, onOpenNewConnection }: {
  folder: FolderNode
  depth: number
  connections: Connection[]
  activeConnectionId: string
  onSelect: (id: string) => void
  tree: TreeNode[]
  onCommit: (updated: TreeNode[]) => void
  onConnectionsChanged: () => void
  onOpenNewConnection: () => void
}) {
  const [renaming, setRenaming] = useState(false)
  const [showCtx, setShowCtx] = useState(false)
  const [dropOver, setDropOver] = useState(false)
  const ctxRef = useRef<HTMLDivElement>(null)
  const isOpen = folder.expanded !== false
  const connCount = countConnections(folder.children)

  useEffect(() => {
    if (!showCtx) return
    const handler = (e: MouseEvent) => {
      if (ctxRef.current && !ctxRef.current.contains(e.target as Node)) setShowCtx(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [showCtx])

  const toggleExpand = () => onCommit(updateFolder(tree, folder.id, f => ({ ...f, expanded: !isOpen })))

  const handleDragStart = (e: React.DragEvent) => {
    e.dataTransfer.setData('text/plain', nodeId(folder))
    e.dataTransfer.effectAllowed = 'move'
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    setDropOver(true)
  }

  const handleDragLeave = () => setDropOver(false)

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDropOver(false)
    const dragId = e.dataTransfer.getData('text/plain')
    if (!dragId || dragId === nodeId(folder)) return
    // Don't allow dropping a folder into itself
    const draggedNode = findNode(tree, dragId)
    if (!draggedNode) {
      // Might be from ungrouped — parse connectionId
      if (dragId.startsWith('conn:')) {
        const connId = dragId.slice(5)
        const item: ConnectionNode = { type: 'connection', connectionId: connId }
        const updated = insertIntoFolder(tree, folder.id, item)
        onCommit(updated)
      }
      return
    }
    const cleaned = removeFromTree(tree, dragId)
    const updated = insertIntoFolder(cleaned, folder.id, draggedNode)
    onCommit(updated)
  }

  return (
    <div>
      <div
        draggable
        onDragStart={handleDragStart}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        className={`w-full flex items-center gap-1.5 py-[4px] text-left transition-colors group
          ${dropOver ? 'bg-accent-sql/10 outline outline-1 outline-accent-sql/30 rounded-md' : 'hover:bg-surface-hover/40'}`}
        style={{ paddingLeft: depth * 14 + 8 }}
        onContextMenu={e => { e.preventDefault(); setShowCtx(true) }}
      >
        <button onClick={toggleExpand} className="shrink-0">
          {isOpen ? <ChevronDown size={11} className="text-text-muted" /> : <ChevronRight size={11} className="text-text-muted" />}
        </button>
        <Folder size={13} className={isOpen ? 'text-amber-400/60 shrink-0' : 'text-text-muted shrink-0'} />
        {renaming ? (
          <InlineRename value={folder.name} onConfirm={name => {
            if (name.trim()) onCommit(updateFolder(tree, folder.id, f => ({ ...f, name: name.trim() })))
            setRenaming(false)
          }} onCancel={() => setRenaming(false)} />
        ) : (
          <>
            <span className="text-[12px] text-text-secondary truncate flex-1" onDoubleClick={() => setRenaming(true)}>{folder.name}</span>
            <span className="text-[9px] text-text-muted tabular-nums shrink-0">{connCount || ''}</span>
            <div className="relative shrink-0 mr-1" ref={ctxRef}>
              <button onClick={e => { e.stopPropagation(); setShowCtx(!showCtx) }}
                className="p-0.5 rounded text-text-muted hover:text-text-primary opacity-0 group-hover:opacity-100 transition-all">
                <MoreHorizontal size={12} />
              </button>
              {showCtx && (
                <div className="absolute right-0 top-full mt-1 w-36 bg-surface-2 border border-border-base rounded-lg shadow-2xl shadow-black/50 py-1 z-50 animate-scale-in">
                  <button onClick={() => { onOpenNewConnection(); setShowCtx(false) }} className="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors">
                    <Plus size={12} /> Add connection
                  </button>
                  <button onClick={() => {
                    const nf: FolderNode = { type: 'folder', id: crypto.randomUUID(), name: 'New folder', children: [], expanded: true }
                    onCommit(updateFolder(tree, folder.id, f => ({ ...f, children: [...f.children, nf], expanded: true })))
                    setShowCtx(false)
                  }} className="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors">
                    <FolderPlus size={12} /> Add subfolder
                  </button>
                  <div className="my-1 border-t border-border-dim" />
                  <button onClick={() => { setRenaming(true); setShowCtx(false) }} className="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors">
                    <Pencil size={12} /> Rename
                  </button>
                  <div className="my-1 border-t border-border-dim" />
                  <button onClick={() => {
                    onCommit(removeFromTree(tree, nodeId(folder)))
                    setShowCtx(false)
                  }} className="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-red-400/80 hover:text-red-400 hover:bg-red-500/5 transition-colors">
                    <Trash2 size={12} /> Delete
                  </button>
                </div>
              )}
            </div>
          </>
        )}
      </div>
      {isOpen && folder.children.map((child, i) => (
        <SidebarTreeNode key={child.type === 'folder' ? child.id : child.connectionId + i} node={child} depth={depth + 1}
          connections={connections} activeConnectionId={activeConnectionId} onSelect={onSelect} tree={tree} onCommit={onCommit}
          onConnectionsChanged={onConnectionsChanged} onOpenNewConnection={onOpenNewConnection} />
      ))}
    </div>
  )
}

// --- Connection context menu ---
function ConnectionContextMenu({ conn, onClose, onConnectionsChanged, tree, onCommit }: {
  conn: Connection
  onClose: () => void
  onConnectionsChanged: () => void
  tree: TreeNode[]
  onCommit: (updated: TreeNode[]) => void
}) {
  const [renaming, setRenaming] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [showDisconnectConfirm, setShowDisconnectConfirm] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) onClose()
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [onClose])

  const handleConnect = async () => {
    onClose()
    try {
      await api.connectToDatabase(conn.id)
      onConnectionsChanged()
      toast.success('Connected')
    } catch (err: any) {
      toast.error(err.message || 'Failed to connect')
    }
  }

  const handleDisconnect = async () => {
    try {
      await api.disconnectFromDatabase(conn.id)
      onConnectionsChanged()
      toast.success('Disconnected')
    } catch (err: any) {
      toast.error(err.message || 'Failed to disconnect')
    }
  }

  const handleDelete = async () => {
    try {
      // Remove from tree first
      onCommit(removeFromTree(tree, `conn:${conn.id}`))
      await api.deleteConnection(conn.id)
      onConnectionsChanged()
      toast.success('Deleted')
    } catch (err: any) {
      toast.error(err.message || 'Failed to delete')
    }
  }

  const handleRename = async (name: string) => {
    if (!name.trim()) { setRenaming(false); return }
    try {
      await api.updateConnection({ ...conn, name: name.trim() })
      onConnectionsChanged()
    } catch (err: any) {
      toast.error(err.message || 'Failed to rename')
    }
    setRenaming(false)
    onClose()
  }

  if (renaming) {
    return (
      <div ref={menuRef} className="absolute right-0 top-full mt-1 w-44 bg-surface-2 border border-border-base rounded-lg shadow-2xl shadow-black/50 p-2 z-50 animate-scale-in">
        <InlineRename value={conn.name} onConfirm={handleRename} onCancel={onClose} />
      </div>
    )
  }

  return (
    <>
      <div ref={menuRef} className="absolute right-0 top-full mt-1 w-44 bg-surface-2 border border-border-base rounded-lg shadow-2xl shadow-black/50 py-1 z-50 animate-scale-in">
        <button onClick={() => setRenaming(true)} className="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors">
          <Pencil size={12} /> Rename
        </button>
        <div className="my-1 border-t border-border-dim" />
        {conn.status === 'connected' ? (
          <button onClick={() => { onClose(); setShowDisconnectConfirm(true) }} className="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-yellow-400/80 hover:text-yellow-400 hover:bg-yellow-500/5 transition-colors">
            <Unplug size={12} /> Disconnect
          </button>
        ) : (
          <>
            <button onClick={handleConnect} className="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-secondary hover:text-text-primary hover:bg-surface-hover transition-colors">
              <PlugZap size={12} /> {conn.status === 'error' ? 'Reconnect' : 'Connect'}
            </button>
          </>
        )}
        <div className="my-1 border-t border-border-dim" />
        <button onClick={() => { onClose(); setShowDeleteConfirm(true) }} className="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-red-400/80 hover:text-red-400 hover:bg-red-500/5 transition-colors">
          <Trash2 size={12} /> Delete
        </button>
      </div>
      <ConfirmDialog open={showDisconnectConfirm} onClose={() => setShowDisconnectConfirm(false)} onConfirm={handleDisconnect}
        title="Disconnect" message={`Disconnect from ${conn.name}? Active queries will be terminated.`} confirmLabel="Disconnect" variant="warning" />
      <ConfirmDialog open={showDeleteConfirm} onClose={() => setShowDeleteConfirm(false)} onConfirm={handleDelete}
        title="Delete connection" message={`Permanently delete "${conn.name}"?`} confirmLabel="Delete" variant="danger" />
    </>
  )
}

// --- Connection node ---
function ConnectionNodeView({ node, depth, connections, activeConnectionId, onSelect, tree, onCommit, onConnectionsChanged }: {
  node: ConnectionNode
  depth: number
  connections: Connection[]
  activeConnectionId: string
  onSelect: (id: string) => void
  tree: TreeNode[]
  onCommit: (updated: TreeNode[]) => void
  onConnectionsChanged: () => void
}) {
  const [showCtx, setShowCtx] = useState(false)
  const conn = connections.find(c => c.id === node.connectionId)
  if (!conn) return null
  const active = conn.id === activeConnectionId

  const handleDragStart = (e: React.DragEvent) => {
    e.dataTransfer.setData('text/plain', nodeId(node))
    e.dataTransfer.effectAllowed = 'move'
  }

  return (
    <div
      draggable
      onDragStart={handleDragStart}
      onContextMenu={e => { e.preventDefault(); setShowCtx(true) }}
      onClick={() => onSelect(conn.id)}
      className={`w-full flex items-center gap-1.5 py-[5px] rounded-md text-[12px] transition-all cursor-grab active:cursor-grabbing group
        ${active ? 'bg-surface-3 text-text-primary' : 'text-text-secondary hover:text-text-primary hover:bg-surface-hover'}`}
      style={{ paddingLeft: depth * 14 + 8 }}
    >
      <EngineIcon engine={conn.engine} size={13} />
      <span className={`w-[5px] h-[5px] rounded-full shrink-0 ${
        conn.status === 'connected' ? 'bg-status-ok' :
        conn.status === 'error' ? 'bg-status-error' : 'bg-status-idle'
      }`} />
      <span className="truncate flex-1">{conn.name}</span>
      <div className="relative shrink-0 mr-1">
        <button onClick={e => { e.stopPropagation(); setShowCtx(!showCtx) }}
          className="p-0.5 rounded text-text-muted hover:text-text-primary opacity-0 group-hover:opacity-100 transition-all">
          <MoreHorizontal size={12} />
        </button>
        {showCtx && <ConnectionContextMenu conn={conn} onClose={() => setShowCtx(false)}
          onConnectionsChanged={onConnectionsChanged} tree={tree} onCommit={onCommit} />}
      </div>
    </div>
  )
}

// --- Dispatcher ---
function SidebarTreeNode(props: {
  node: TreeNode; depth: number; connections: Connection[]; activeConnectionId: string
  onSelect: (id: string) => void; tree: TreeNode[]; onCommit: (t: TreeNode[]) => void
  onConnectionsChanged: () => void; onOpenNewConnection: () => void
}) {
  if (props.node.type === 'folder') return <FolderNodeView folder={props.node} {...props} />
  return <ConnectionNodeView {...props} node={props.node} />
}

// --- Ungrouped connection (with context menu) ---
function UngroupedConnectionView({ conn, active, onSelect, onConnectionsChanged, tree, onCommit }: {
  conn: Connection; active: boolean; onSelect: (id: string) => void
  onConnectionsChanged: () => void; tree: TreeNode[]; onCommit: (t: TreeNode[]) => void
}) {
  const [showCtx, setShowCtx] = useState(false)
  return (
    <div
      draggable
      onDragStart={e => {
        e.dataTransfer.setData('text/plain', `conn:${conn.id}`)
        e.dataTransfer.effectAllowed = 'move'
      }}
      onContextMenu={e => { e.preventDefault(); setShowCtx(true) }}
      onClick={() => onSelect(conn.id)}
      className={`w-full flex items-center gap-1.5 px-2 py-[5px] rounded-md text-[12px] transition-all cursor-grab active:cursor-grabbing group
        ${active ? 'bg-surface-3 text-text-primary' : 'text-text-secondary hover:text-text-primary hover:bg-surface-hover'}`}
    >
      <EngineIcon engine={conn.engine} size={13} />
      <span className={`w-[5px] h-[5px] rounded-full shrink-0 ${
        conn.status === 'connected' ? 'bg-status-ok' :
        conn.status === 'error' ? 'bg-status-error' : 'bg-status-idle'
      }`} />
      <span className="truncate flex-1">{conn.name}</span>
      <div className="relative shrink-0 mr-1">
        <button onClick={e => { e.stopPropagation(); setShowCtx(!showCtx) }}
          className="p-0.5 rounded text-text-muted hover:text-text-primary opacity-0 group-hover:opacity-100 transition-all">
          <MoreHorizontal size={12} />
        </button>
        {showCtx && <ConnectionContextMenu conn={conn} onClose={() => setShowCtx(false)}
          onConnectionsChanged={onConnectionsChanged} tree={tree} onCommit={onCommit} />}
      </div>
    </div>
  )
}

// --- Root drop zone (drop onto root level) ---
function RootDropZone({ tree, onCommit }: { tree: TreeNode[]; onCommit: (t: TreeNode[]) => void }) {
  const [dropOver, setDropOver] = useState(false)

  return (
    <div
      onDragOver={e => { e.preventDefault(); e.dataTransfer.dropEffect = 'move'; setDropOver(true) }}
      onDragLeave={() => setDropOver(false)}
      onDrop={e => {
        e.preventDefault()
        setDropOver(false)
        const dragId = e.dataTransfer.getData('text/plain')
        if (!dragId) return
        const node = findNode(tree, dragId)
        if (!node) return
        const cleaned = removeFromTree(tree, dragId)
        onCommit([...cleaned, node])
      }}
      className={`h-4 mx-2 my-1 rounded transition-colors ${dropOver ? 'bg-accent-sql/10 border border-dashed border-accent-sql/30' : ''}`}
    />
  )
}

// === Main Sidebar ===
export function Sidebar({ connections, activeConnectionId, onSelect, collapsed, onToggle, onConnectionsChanged }: SidebarProps) {
  const [showNew, setShowNew] = useState(false)
  const [tree, setTree] = useState<TreeNode[]>([])
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    api.getSidebarTree().then(raw => {
      try {
        const parsed = JSON.parse(raw)
        if (Array.isArray(parsed)) setTree(parsed)
      } catch { /* ignore */ }
      setLoaded(true)
    }).catch(() => setLoaded(true))
  }, [])

  const commit = useCallback((t: TreeNode[]) => {
    setTree(t)
    api.saveSidebarTree(JSON.stringify(t)).catch(() => {})
  }, [])

  const addRootFolder = () => {
    const nf: FolderNode = { type: 'folder', id: crypto.randomUUID(), name: 'New folder', children: [], expanded: true }
    commit([...tree, nf])
  }

  const treeConnIds = getTreeConnectionIds(tree)
  const ungrouped = connections.filter(c => !treeConnIds.has(c.id))

  return (
    <>
      <aside
        className="h-screen bg-surface-1 border-r border-border-dim flex flex-col shrink-0 transition-[width] duration-200 ease-out"
        style={{ width: collapsed ? 48 : 240 }}
      >
        <div className="flex items-center justify-between h-12 px-3 border-b border-border-dim shrink-0">
          {!collapsed && <span className="font-display text-base italic text-text-primary tracking-tight pl-1">DataForge</span>}
          <button onClick={onToggle} className="p-1 rounded-md text-text-muted hover:text-text-secondary transition-colors">
            {collapsed ? <PanelLeftOpen size={16} /> : <PanelLeftClose size={16} />}
          </button>
        </div>

        {!collapsed && (
          <div className="px-2 pt-2 flex gap-1">
            <button onClick={() => setShowNew(true)}
              className="flex-1 flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-[12px] text-text-muted hover:text-text-secondary hover:bg-surface-hover transition-colors">
              <Plus size={13} /> Connection
            </button>
            <button onClick={addRootFolder}
              className="flex items-center gap-1 px-2 py-1.5 rounded-md text-[12px] text-text-muted hover:text-text-secondary hover:bg-surface-hover transition-colors"
              title="Add folder">
              <FolderPlus size={13} />
            </button>
          </div>
        )}

        <nav className="flex-1 overflow-y-auto px-1 py-2">
          {!collapsed && loaded && (
            <>
              {tree.map((node, i) => (
                <SidebarTreeNode key={node.type === 'folder' ? node.id : (node.connectionId + i)} node={node} depth={0}
                  connections={connections} activeConnectionId={activeConnectionId} onSelect={onSelect} tree={tree} onCommit={commit}
                  onConnectionsChanged={onConnectionsChanged} onOpenNewConnection={() => setShowNew(true)} />
              ))}

              {tree.length > 0 && <RootDropZone tree={tree} onCommit={commit} />}

              {ungrouped.length > 0 && (
                <div className={tree.length > 0 ? 'mt-1 pt-2 border-t border-border-dim' : ''}>
                  {tree.length > 0 && (
                    <div className="px-2 pb-1">
                      <span className="text-[10px] font-semibold text-text-muted uppercase tracking-widest">Ungrouped</span>
                    </div>
                  )}
                  {ungrouped.map(conn => (
                    <UngroupedConnectionView key={conn.id} conn={conn} active={conn.id === activeConnectionId}
                      onSelect={onSelect} onConnectionsChanged={onConnectionsChanged} tree={tree} onCommit={commit} />
                  ))}
                </div>
              )}

              {connections.length === 0 && tree.length === 0 && (
                <div className="flex flex-col items-center justify-center h-full text-center py-8 px-4">
                  <Database size={20} className="text-text-muted mb-2" />
                  <p className="text-[12px] text-text-muted">No connections yet</p>
                </div>
              )}
            </>
          )}
        </nav>

        {!collapsed && (
          <div className="px-2 py-2 border-t border-border-dim">
            <button className="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-md text-[13px] text-text-muted hover:text-text-secondary hover:bg-surface-hover transition-colors">
              <Settings size={14} />
              Settings
            </button>
          </div>
        )}
      </aside>

      <NewConnectionModal open={showNew} onClose={() => setShowNew(false)} onConnectionsChanged={onConnectionsChanged} />
    </>
  )
}
