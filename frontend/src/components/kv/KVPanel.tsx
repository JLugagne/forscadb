import { useState, useMemo, useEffect } from 'react'
import {
  Search, Plus, Trash2, Copy, Clock, Pencil,
  Activity, Gauge, Zap, Server, BarChart3, Timer, HardDrive,
  ChevronRight, ChevronDown, Folder, FolderOpen,
} from 'lucide-react'
import type { KVEntry, KVStats } from '../../types/database'
import { Modal } from '../shared/Modal'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import { ResizeHandle } from '../shared/ResizeHandle'
import { useResize } from '../../hooks/useResize'
import * as api from '../../api'
import toast from 'react-hot-toast'

function formatTTL(ttl: number | null) {
  if (ttl === null) return '\u221e'
  if (ttl < 60) return `${ttl}s`
  if (ttl < 3600) return `${Math.floor(ttl / 60)}m${ttl % 60 ? ` ${ttl % 60}s` : ''}`
  return `${Math.floor(ttl / 3600)}h${Math.floor((ttl % 3600) / 60)}m`
}

const typeBadge: Record<string, string> = {
  string: 'text-emerald-400 bg-emerald-500/8',
  list: 'text-blue-400 bg-blue-500/8',
  set: 'text-pink-400 bg-pink-500/8',
  zset: 'text-purple-400 bg-purple-500/8',
  hash: 'text-amber-400 bg-amber-500/8',
  stream: 'text-orange-400 bg-orange-500/8',
}

const typeLabel: Record<string, string> = {
  string: 'str', list: 'list', set: 'set', zset: 'zset', hash: 'hash', stream: 'stream',
}

// --- Tree data structure ---
interface TreeNode {
  segment: string      // this segment name (e.g. "session", "a1b2c3d4")
  fullPath: string     // full path up to here (e.g. "session", "session:a1b2c3d4")
  entry?: KVEntry      // leaf node has the actual entry
  children: TreeNode[]
}

function buildTree(entries: KVEntry[], separator = ':'): TreeNode[] {
  const root: TreeNode[] = []

  for (const entry of entries) {
    const parts = entry.key.split(separator)
    let current = root

    for (let i = 0; i < parts.length; i++) {
      const segment = parts[i]
      const fullPath = parts.slice(0, i + 1).join(separator)
      const isLeaf = i === parts.length - 1

      let node = current.find(n => n.segment === segment && !n.entry === !isLeaf)
      // For leaves, always create a new node (keys are unique)
      if (!node && !isLeaf) {
        node = current.find(n => n.segment === segment && n.children.length > 0)
      }

      if (!node) {
        node = {
          segment,
          fullPath,
          entry: isLeaf ? entry : undefined,
          children: [],
        }
        current.push(node)
      }

      if (!isLeaf) {
        current = node.children
      }
    }
  }

  return root
}

function countLeaves(nodes: TreeNode[]): number {
  let count = 0
  for (const n of nodes) {
    if (n.entry) count++
    else count += countLeaves(n.children)
  }
  return count
}

// --- Tree node component ---
function TreeNodeView({ node, depth, selectedKey, onSelect, expanded, onToggle }: {
  node: TreeNode
  depth: number
  selectedKey: string | null
  onSelect: (entry: KVEntry) => void
  expanded: Set<string>
  onToggle: (path: string) => void
}) {
  const isFolder = !node.entry && node.children.length > 0
  const isOpen = expanded.has(node.fullPath)
  const isSelected = node.entry && selectedKey === node.entry.key

  if (isFolder) {
    const leafCount = countLeaves(node.children)
    return (
      <div>
        <button
          onClick={() => onToggle(node.fullPath)}
          className="w-full flex items-center gap-1.5 py-[3px] text-left transition-colors hover:bg-surface-hover/40 group"
          style={{ paddingLeft: depth * 14 + 8 }}
        >
          {isOpen
            ? <ChevronDown size={11} className="text-text-muted shrink-0" />
            : <ChevronRight size={11} className="text-text-muted shrink-0" />
          }
          {isOpen
            ? <FolderOpen size={12} className="text-accent-kv/60 shrink-0" />
            : <Folder size={12} className="text-text-muted shrink-0" />
          }
          <span className="text-[12px] font-mono text-text-secondary truncate">{node.segment}</span>
          <span className="ml-auto pr-2 text-[9px] text-text-muted tabular-nums shrink-0">{leafCount}</span>
        </button>
        {isOpen && node.children.map(child => (
          <TreeNodeView
            key={child.fullPath + (child.entry ? ':leaf' : '')}
            node={child}
            depth={depth + 1}
            selectedKey={selectedKey}
            onSelect={onSelect}
            expanded={expanded}
            onToggle={onToggle}
          />
        ))}
      </div>
    )
  }

  // Leaf node
  const entry = node.entry!
  return (
    <button
      onClick={() => onSelect(entry)}
      className={`w-full flex items-center gap-1.5 py-[3px] text-left transition-colors
        ${isSelected
          ? 'bg-accent-kv/8 text-text-primary'
          : 'text-text-secondary hover:text-text-primary hover:bg-surface-hover'
        }`}
      style={{ paddingLeft: depth * 14 + 8 }}
    >
      <span className={`text-[8px] font-bold uppercase tracking-wider px-1 py-px rounded shrink-0 ${typeBadge[entry.type] || ''}`}>
        {typeLabel[entry.type]}
      </span>
      <span className="font-mono text-[11px] truncate flex-1">{node.segment}</span>
      <span className="text-[9px] text-text-muted font-mono shrink-0 pr-2">{formatTTL(entry.ttl)}</span>
    </button>
  )
}

// --- Stats card ---
function StatCard({ icon: Icon, label, value, sub, color }: { icon: typeof Activity; label: string; value: string; sub?: string; color: string }) {
  return (
    <div className="p-3 rounded-lg bg-surface-2/50 border border-border-dim">
      <div className="flex items-center gap-1.5 mb-1">
        <Icon size={12} className={color} />
        <span className="text-[10px] text-text-muted uppercase tracking-wider">{label}</span>
      </div>
      <div className="text-base font-semibold text-text-primary font-mono">{value}</div>
      {sub && <div className="text-[10px] text-text-muted mt-0.5">{sub}</div>}
    </div>
  )
}

// --- Main panel ---
interface KVPanelProps {
  connectionId: string
}

export function KVPanel({ connectionId }: KVPanelProps) {
  const [entries, setEntries] = useState<KVEntry[]>([])
  const [stats, setStats] = useState<KVStats | null>(null)
  const [selected, setSelected] = useState<KVEntry | null>(null)
  const [search, setSearch] = useState('')
  const [view, setView] = useState<'keys' | 'stats'>('keys')
  const [showAdd, setShowAdd] = useState(false)
  const [showEdit, setShowEdit] = useState<KVEntry | null>(null)
  const [deleteKey, setDeleteKey] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const keyList = useResize({ direction: 'horizontal', initialSize: 300, minSize: 180, maxSize: 500 })

  // Add key form state
  const [addKey, setAddKey] = useState('')
  const [addValue, setAddValue] = useState('')
  const [addTTL, setAddTTL] = useState('')

  // Edit key form state
  const [editValue, setEditValue] = useState('')
  const [editTTL, setEditTTL] = useState('')

  // Load data on mount
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    Promise.all([
      api.getKeys(connectionId, '*', 500),
      api.getKVStats(connectionId),
    ]).then(([k, s]) => {
      if (cancelled) return
      setEntries(k || [])
      setStats(s || null)
      if (k?.length) setSelected(k[0])
      setLoading(false)
    }).catch(err => { if (!cancelled) { console.error(err); setLoading(false) } })
    return () => { cancelled = true }
  }, [connectionId])

  // Init edit values when showEdit changes
  useEffect(() => {
    if (showEdit) {
      setEditValue(prettyValue(showEdit.value))
      setEditTTL(showEdit.ttl?.toString() ?? '')
    }
  }, [showEdit])

  // Expanded folder paths
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  // Auto-expand top-level folders when entries load
  useEffect(() => {
    const topLevel = new Set<string>()
    for (const e of entries) {
      const first = e.key.split(':')[0]
      topLevel.add(first)
    }
    setExpanded(topLevel)
  }, [entries])

  const toggleExpand = (path: string) => {
    setExpanded(prev => {
      const next = new Set(prev)
      next.has(path) ? next.delete(path) : next.add(path)
      return next
    })
  }

  const filtered = useMemo(() =>
    entries.filter(e => e.key.toLowerCase().includes(search.toLowerCase())),
    [entries, search]
  )

  const tree = useMemo(() => buildTree(filtered), [filtered])

  const prettyValue = (v: string) => {
    try { return JSON.stringify(JSON.parse(v), null, 2) } catch { return v }
  }

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center text-[13px] text-text-muted">
        Loading...
      </div>
    )
  }

  return (
    <div className="flex h-full">
      {/* LEFT: Key tree (resizable) */}
      <div className="shrink-0 border-r border-border-dim flex flex-col bg-surface-1" style={{ width: keyList.size }}>
        <div className="h-9 flex items-center gap-2 px-3 border-b border-border-dim shrink-0">
          <div className="flex bg-surface-2 rounded-md p-0.5 border border-border-dim">
            <button onClick={() => setView('keys')} className={`px-2 py-0.5 text-[11px] font-medium rounded transition-colors ${view === 'keys' ? 'bg-surface-3 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}>Keys</button>
            <button onClick={() => setView('stats')} className={`px-2 py-0.5 text-[11px] font-medium rounded transition-colors ${view === 'stats' ? 'bg-surface-3 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}>Stats</button>
          </div>
          <span className="ml-auto text-[10px] text-text-muted tabular-nums font-mono">{stats?.totalKeys?.toLocaleString() ?? entries.length}</span>
          <button onClick={() => setShowAdd(true)} className="p-0.5 rounded text-text-muted hover:text-accent-kv transition-colors"><Plus size={13} /></button>
        </div>

        {view === 'keys' && (
          <>
            <div className="px-2 py-1.5 border-b border-border-dim">
              <div className="relative">
                <Search size={12} className="absolute left-2 top-1/2 -translate-y-1/2 text-text-muted" />
                <input
                  value={search}
                  onChange={e => setSearch(e.target.value)}
                  placeholder="Filter keys..."
                  className="w-full pl-7 pr-2 py-1 text-[12px] font-mono bg-surface-2 border border-border-dim rounded-md
                    text-text-primary placeholder:text-text-muted outline-none focus:border-border-base transition-colors"
                />
              </div>
            </div>
            <div className="flex-1 overflow-y-auto py-0.5">
              {tree.map(node => (
                <TreeNodeView
                  key={node.fullPath + (node.entry ? ':leaf' : '')}
                  node={node}
                  depth={0}
                  selectedKey={selected?.key ?? null}
                  onSelect={setSelected}
                  expanded={expanded}
                  onToggle={toggleExpand}
                />
              ))}
              {entries.length === 0 && (
                <div className="flex items-center justify-center h-32 text-[12px] text-text-muted">No keys found</div>
              )}
            </div>
          </>
        )}

        {view === 'stats' && stats && (
          <div className="flex-1 overflow-y-auto p-2 space-y-2">
            <StatCard icon={HardDrive} label="Keys" value={stats.totalKeys.toLocaleString()} color="text-accent-kv" />
            <StatCard icon={Server} label="Memory" value={stats.memoryUsed} sub={`Peak: ${stats.memoryPeak}`} color="text-blue-400" />
            <StatCard icon={Zap} label="Ops/sec" value={stats.opsPerSec.toLocaleString()} color="text-amber-400" />
            <StatCard icon={Gauge} label="Hit Rate" value={`${stats.hitRate}%`} sub={`${stats.keyspaceHits.toLocaleString()} hits`} color="text-emerald-400" />
            <StatCard icon={Activity} label="Clients" value={String(stats.connectedClients)} color="text-purple-400" />
            <StatCard icon={Timer} label="Uptime" value={`${stats.uptimeDays}d`} color="text-cyan-400" />
            <StatCard icon={BarChart3} label="Misses" value={stats.keyspaceMisses.toLocaleString()} color="text-red-400" />
            <div className="p-3 rounded-lg bg-surface-2/50 border border-border-dim">
              <div className="flex justify-between mb-1.5">
                <span className="text-[10px] text-text-muted uppercase tracking-wider">Cache hit rate</span>
                <span className="text-[12px] font-mono font-semibold text-text-primary">{stats.hitRate}%</span>
              </div>
              <div className="h-1.5 bg-surface-3 rounded-full overflow-hidden">
                <div
                  className="h-full rounded-full bg-gradient-to-r from-accent-kv to-purple-400 transition-all duration-700 ease-out"
                  style={{ width: `${stats.hitRate}%` }}
                />
              </div>
            </div>
          </div>
        )}

        {view === 'stats' && !stats && (
          <div className="flex-1 flex items-center justify-center text-[12px] text-text-muted">No stats available</div>
        )}
      </div>

      <ResizeHandle direction="horizontal" onMouseDown={keyList.onMouseDown} />

      {/* RIGHT: Value viewer */}
      <div className="flex-1 flex flex-col min-w-0">
        {selected ? (
          <>
            <div className="h-9 flex items-center gap-3 px-4 border-b border-border-dim bg-surface-1 shrink-0">
              <span className={`text-[9px] font-bold uppercase tracking-wider px-1.5 py-px rounded ${typeBadge[selected.type] || ''}`}>
                {selected.type}
              </span>
              <span className="text-[13px] font-medium text-text-primary font-mono truncate">{selected.key}</span>
              <div className="ml-auto flex items-center gap-2">
                <span className="flex items-center gap-1 text-[11px] text-text-muted"><Clock size={11} /> {formatTTL(selected.ttl)}</span>
                <span className="text-[11px] text-text-muted font-mono">{selected.size}</span>
                <span className="text-[11px] text-text-muted">{selected.encoding}</span>
                <div className="w-px h-4 bg-border-dim ml-1" />
                <button onClick={() => { navigator.clipboard.writeText(selected.value); toast.success('Copied') }} className="p-1 rounded-md text-text-muted hover:text-text-secondary transition-colors"><Copy size={13} /></button>
                <button onClick={() => setShowEdit(selected)} className="p-1 rounded-md text-text-muted hover:text-text-secondary transition-colors"><Pencil size={13} /></button>
                <button onClick={() => setDeleteKey(selected.key)} className="p-1 rounded-md text-text-muted hover:text-red-400 transition-colors"><Trash2 size={13} /></button>
              </div>
            </div>
            <div className="flex-1 overflow-auto p-4">
              <pre className="font-mono text-[13px] text-text-secondary leading-relaxed whitespace-pre-wrap break-all">
                {prettyValue(selected.value)}
              </pre>
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-[13px] text-text-muted">
            Select a key to view its value
          </div>
        )}
      </div>

      {/* Modals */}
      <Modal open={showAdd} onClose={() => setShowAdd(false)} title="Set Key">
        <div className="space-y-3">
          <div>
            <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Key</label>
            <input type="text" placeholder="cache:my_key" value={addKey} onChange={e => setAddKey(e.target.value)} className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary placeholder:text-text-muted outline-none focus:border-border-focus transition-colors font-mono" />
          </div>
          <div>
            <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">TTL (sec)</label>
            <input type="number" placeholder="0 = no expiry" value={addTTL} onChange={e => setAddTTL(e.target.value)} className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary placeholder:text-text-muted outline-none focus:border-border-focus transition-colors font-mono" />
          </div>
          <div>
            <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Value</label>
            <textarea rows={5} value={addValue} onChange={e => setAddValue(e.target.value)} className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary outline-none focus:border-border-focus transition-colors font-mono resize-none" />
          </div>
          <div className="flex gap-2 justify-end pt-1">
            <button onClick={() => setShowAdd(false)} className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-surface-3 text-text-secondary hover:text-text-primary transition-colors">Cancel</button>
            <button
              onClick={async () => {
                try {
                  const ttl = addTTL ? parseInt(addTTL) : null
                  await api.setKVEntry(connectionId, addKey, addValue, ttl)
                  toast.success('Key set')
                  setShowAdd(false)
                  setAddKey(''); setAddValue(''); setAddTTL('')
                  const k = await api.getKeys(connectionId, '*', 500)
                  setEntries(k || [])
                } catch (err: any) {
                  toast.error(err.message || 'Failed to set key')
                }
              }}
              className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-accent-kv text-white hover:bg-accent-kv/90 transition-colors"
            >
              Set
            </button>
          </div>
        </div>
      </Modal>

      <Modal open={showEdit !== null} onClose={() => setShowEdit(null)} title="Edit Key" subtitle={showEdit?.key}>
        {showEdit && (
          <div className="space-y-3">
            <div>
              <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Value</label>
              <textarea rows={8} value={editValue} onChange={e => setEditValue(e.target.value)} className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary outline-none focus:border-border-focus transition-colors font-mono resize-none" spellCheck={false} />
            </div>
            <div>
              <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">TTL (sec)</label>
              <input type="number" value={editTTL} onChange={e => setEditTTL(e.target.value)} placeholder="no expiry" className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary placeholder:text-text-muted outline-none focus:border-border-focus transition-colors font-mono" />
            </div>
            <div className="flex gap-2 justify-end pt-1">
              <button onClick={() => setShowEdit(null)} className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-surface-3 text-text-secondary hover:text-text-primary transition-colors">Cancel</button>
              <button
                onClick={async () => {
                  try {
                    const ttl = editTTL ? parseInt(editTTL) : null
                    await api.setKVEntry(connectionId, showEdit!.key, editValue, ttl)
                    toast.success('Updated')
                    setShowEdit(null)
                    const k = await api.getKeys(connectionId, '*', 500)
                    setEntries(k || [])
                    if (selected?.key === showEdit!.key) {
                      const entry = await api.getKVEntry(connectionId, showEdit!.key)
                      setSelected(entry)
                    }
                  } catch (err: any) {
                    toast.error(err.message || 'Failed to update key')
                  }
                }}
                className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-accent-kv text-white hover:bg-accent-kv/90 transition-colors"
              >
                Save
              </button>
            </div>
          </div>
        )}
      </Modal>

      <ConfirmDialog
        open={deleteKey !== null}
        onClose={() => setDeleteKey(null)}
        onConfirm={async () => {
          try {
            await api.deleteKVEntry(connectionId, deleteKey!)
            toast.success('Deleted')
            const deletedKey = deleteKey
            setDeleteKey(null)
            const k = await api.getKeys(connectionId, '*', 500)
            setEntries(k || [])
            if (selected?.key === deletedKey) setSelected(null)
          } catch (err: any) {
            toast.error(err.message || 'Failed to delete key')
          }
        }}
        title="Delete Key"
        message={`Delete "${deleteKey}"?`}
        confirmLabel="Delete"
      />
    </div>
  )
}
