import { useState, useEffect } from 'react'
import {
  FileStack, Plus, Trash2, Copy, Pencil,
  ChevronRight, ChevronDown, FileJson,
} from 'lucide-react'
import type { NoSQLCollection } from '../../types/database'
import { Modal } from '../shared/Modal'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import { JsonEditor } from '../shared/JsonEditor'
import { ResizeHandle } from '../shared/ResizeHandle'
import { useResize } from '../../hooks/useResize'
import * as api from '../../api'
import toast from 'react-hot-toast'

function fmt(n: number) {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

// --- JSON tree viewer ---
function JsonNode({ value, depth = 0 }: { value: unknown; depth?: number }) {
  const [open, setOpen] = useState(depth < 2)

  if (value === null) return <span className="text-text-muted italic">null</span>
  if (typeof value === 'boolean') return <span className={value ? 'text-emerald-400' : 'text-red-400'}>{String(value)}</span>
  if (typeof value === 'number') return <span className="text-blue-400">{value}</span>
  if (typeof value === 'string') return <span className="text-amber-300">"{value}"</span>

  if (Array.isArray(value)) {
    if (!value.length) return <span className="text-text-muted">[]</span>
    return (
      <span>
        <button onClick={() => setOpen(!open)} className="text-text-muted hover:text-text-secondary">
          {open ? <ChevronDown size={11} className="inline" /> : <ChevronRight size={11} className="inline" />}
          {!open && <span className="text-[11px]"> [{value.length}]</span>}
        </button>
        {open && <div className="ml-3 border-l border-border-dim pl-2">{value.map((item, i) => <div key={i}><JsonNode value={item} depth={depth + 1} />{i < value.length - 1 && <span className="text-text-muted">,</span>}</div>)}</div>}
      </span>
    )
  }

  if (typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>)
    if (!entries.length) return <span className="text-text-muted">{'{}'}</span>
    if ('$date' in (value as Record<string, unknown>)) return <span className="text-cyan-400">ISODate("{(value as Record<string, string>).$date}")</span>

    return (
      <span>
        <button onClick={() => setOpen(!open)} className="text-text-muted hover:text-text-secondary">
          {open ? <ChevronDown size={11} className="inline" /> : <ChevronRight size={11} className="inline" />}
          {!open && <span className="text-[11px]"> {'{'}...{entries.length}{'}'}</span>}
        </button>
        {open && (
          <div className="ml-3 border-l border-border-dim pl-2">
            {entries.map(([k, v], i) => (
              <div key={k}>
                <span className="text-purple-400/90">"{k}"</span>
                <span className="text-text-muted">: </span>
                <JsonNode value={v} depth={depth + 1} />
                {i < entries.length - 1 && <span className="text-text-muted">,</span>}
              </div>
            ))}
          </div>
        )}
      </span>
    )
  }
  return <span>{String(value)}</span>
}

// --- Main panel ---
interface NoSQLPanelProps {
  connectionId: string
}

export function NoSQLPanel({ connectionId }: NoSQLPanelProps) {
  const [collections, setCollections] = useState<NoSQLCollection[]>([])
  const [selected, setSelected] = useState<NoSQLCollection | null>(null)
  const [documents, setDocuments] = useState<any[]>([])
  const [tab, setTab] = useState<'documents' | 'indexes' | 'schema'>('documents')
  const [expandedDoc, setExpandedDoc] = useState<string | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [showInsert, setShowInsert] = useState(false)
  const [showDropColl, setShowDropColl] = useState(false)
  const [deleteDocId, setDeleteDocId] = useState<string | null>(null)
  const [filter, setFilter] = useState('{}')
  const [loading, setLoading] = useState(true)
  const [newCollName, setNewCollName] = useState('')
  const [insertJson, setInsertJson] = useState(JSON.stringify({ email: '', username: '', profile: { firstName: '', lastName: '' }, roles: ['user'], status: 'active' }, null, 2))
  const [showEditDoc, setShowEditDoc] = useState<any | null>(null)
  const [editDocJson, setEditDocJson] = useState('')
  const collList = useResize({ direction: 'horizontal', initialSize: 220, minSize: 140, maxSize: 400 })

  // Load collections on mount
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    api.getCollections(connectionId).then(c => {
      if (cancelled) return
      setCollections(c || [])
      if (c?.length) setSelected(c[0])
      setLoading(false)
    }).catch(err => { if (!cancelled) { console.error(err); setLoading(false) } })
    return () => { cancelled = true }
  }, [connectionId])

  // Load documents when selected collection changes
  useEffect(() => {
    if (!selected) return
    api.getDocuments(connectionId, selected.name, '', 50)
      .then(d => setDocuments(d || []))
      .catch(err => console.error(err))
  }, [connectionId, selected?.name])

  const handleFind = async () => {
    if (!selected) return
    try {
      const d = await api.getDocuments(connectionId, selected.name, filter === '{}' ? '' : filter, 50)
      setDocuments(d || [])
    } catch (err: any) {
      toast.error(err.message || 'Query failed')
    }
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
      {/* LEFT: Collection list (resizable) */}
      <div className="shrink-0 border-r border-border-dim flex flex-col bg-surface-1" style={{ width: collList.size }}>
        <div className="h-9 flex items-center px-3 border-b border-border-dim shrink-0">
          <span className="text-[11px] font-medium text-text-muted tracking-widest uppercase">Collections</span>
          <button onClick={() => setShowCreate(true)} className="ml-auto p-0.5 rounded text-text-muted hover:text-accent-nosql transition-colors"><Plus size={13} /></button>
        </div>
        <div className="flex-1 overflow-y-auto py-1">
          {collections.map(c => (
            <button
              key={c.name}
              onClick={() => { setSelected(c); setTab('documents') }}
              className={`w-full flex items-center gap-2 px-3 py-[5px] text-[13px] transition-colors
                ${selected?.name === c.name
                  ? 'bg-accent-nosql/8 text-text-primary'
                  : 'text-text-secondary hover:text-text-primary hover:bg-surface-hover'
                }`}
            >
              <FileStack size={13} className={selected?.name === c.name ? 'text-accent-nosql' : 'text-text-muted'} />
              <span className="font-mono truncate text-[12px]">{c.name}</span>
              <span className="ml-auto text-[10px] text-text-muted tabular-nums font-mono">{fmt(c.documentCount)}</span>
            </button>
          ))}
          {collections.length === 0 && (
            <div className="flex items-center justify-center h-32 text-[12px] text-text-muted">No collections</div>
          )}
        </div>
      </div>

      <ResizeHandle direction="horizontal" onMouseDown={collList.onMouseDown} />

      {/* RIGHT: Detail */}
      {selected ? (
        <div className="flex-1 flex flex-col min-w-0">
          {/* Header bar */}
          <div className="h-9 flex items-center gap-3 px-4 border-b border-border-dim bg-surface-1 shrink-0">
            <span className="text-[13px] font-medium text-text-primary font-mono">{selected.name}</span>
            <span className="text-[11px] text-text-muted">{fmt(selected.documentCount)} docs</span>
            <span className="text-[11px] text-text-muted font-mono">{selected.totalSize}</span>

            <div className="ml-auto flex items-center gap-0.5">
              {(['documents', 'indexes', 'schema'] as const).map(t => (
                <button
                  key={t}
                  onClick={() => setTab(t)}
                  className={`px-2.5 py-1 text-[12px] font-medium rounded-md capitalize transition-colors
                    ${tab === t ? 'bg-surface-3 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}
                >
                  {t}
                </button>
              ))}
              <div className="w-px h-4 bg-border-dim mx-1" />
              <button onClick={() => setShowInsert(true)} className="p-1 rounded-md text-text-muted hover:text-accent-nosql transition-colors" title="Insert document"><Plus size={14} /></button>
              <button onClick={() => setShowDropColl(true)} className="p-1 rounded-md text-text-muted hover:text-red-400 transition-colors" title="Drop collection"><Trash2 size={14} /></button>
            </div>
          </div>

          {/* Filter bar */}
          {tab === 'documents' && (
            <div className="flex items-center gap-2 px-4 py-2 border-b border-border-dim bg-surface-1">
              <FileJson size={13} className="text-text-muted shrink-0" />
              <input
                value={filter}
                onChange={e => setFilter(e.target.value)}
                placeholder='{ "field": "value" }'
                className="flex-1 text-[12px] font-mono bg-transparent text-text-primary placeholder:text-text-muted outline-none"
              />
              <button onClick={handleFind} className="px-2.5 py-1 text-[12px] font-medium rounded-md bg-accent-nosql/90 text-white hover:bg-accent-nosql transition-colors">
                Find
              </button>
            </div>
          )}

          {/* Content */}
          <div className="flex-1 overflow-auto">
            {tab === 'documents' && (
              <div className="divide-y divide-border-dim/50">
                {documents.map((doc) => {
                  const isExpanded = expandedDoc === doc._id
                  return (
                    <div
                      key={doc._id}
                      className={`px-4 py-2.5 cursor-pointer transition-colors group
                        ${isExpanded ? 'bg-surface-hover/30' : 'hover:bg-surface-hover/20'}`}
                      onClick={() => setExpandedDoc(isExpanded ? null : doc._id)}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          {isExpanded ? <ChevronDown size={12} className="text-text-muted" /> : <ChevronRight size={12} className="text-text-muted" />}
                          <span className="text-[11px] font-mono text-accent-nosql/80">ObjectId("{doc._id}")</span>
                        </div>
                        <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                          <button onClick={e => { e.stopPropagation(); navigator.clipboard.writeText(JSON.stringify(doc, null, 2)); toast.success('Copied') }} className="p-1 rounded text-text-muted hover:text-text-primary transition-colors"><Copy size={12} /></button>
                          <button onClick={e => { e.stopPropagation(); setShowEditDoc(doc); setEditDocJson(JSON.stringify(doc, null, 2)) }} className="p-1 rounded text-text-muted hover:text-text-primary transition-colors"><Pencil size={12} /></button>
                          <button onClick={e => { e.stopPropagation(); setDeleteDocId(doc._id) }} className="p-1 rounded text-text-muted hover:text-red-400 transition-colors"><Trash2 size={12} /></button>
                        </div>
                      </div>

                      {isExpanded ? (
                        <div className="mt-2 ml-5 p-3 rounded-lg bg-surface-1 border border-border-dim font-mono text-[12px] leading-relaxed">
                          <JsonNode value={doc} />
                        </div>
                      ) : (
                        <div className="ml-5 mt-1 flex gap-1.5 flex-wrap">
                          {Object.entries(doc).filter(([k]) => k !== '_id').slice(0, 4).map(([k, v]) => (
                            <span key={k} className="text-[11px] px-1.5 py-px rounded bg-surface-3/60 text-text-muted">
                              {k}: <span className="text-text-secondary font-mono">{typeof v === 'string' ? v : typeof v === 'object' ? '{...}' : String(v)}</span>
                            </span>
                          ))}
                        </div>
                      )}
                    </div>
                  )
                })}
                {documents.length === 0 && (
                  <div className="flex items-center justify-center h-32 text-[12px] text-text-muted">No documents found</div>
                )}
              </div>
            )}

            {tab === 'indexes' && (
              <table className="w-full text-[13px]">
                <thead>
                  <tr className="bg-surface-1 sticky top-0 z-10 border-b border-border-dim">
                    {['Name', 'Keys', 'Unique'].map(h => (
                      <th key={h} className="px-4 py-2 text-left text-[10px] font-medium text-text-muted tracking-widest uppercase">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {selected.indexes.map((idx) => (
                    <tr key={idx.name} className="border-b border-border-dim/50 hover:bg-surface-hover/30 transition-colors">
                      <td className="px-4 py-2 font-mono text-[12px] text-text-primary">{idx.name}</td>
                      <td className="px-4 py-2">
                        <div className="flex gap-1 flex-wrap">
                          {Object.entries(idx.keys).map(([k, v]) => (
                            <span key={k} className="px-1.5 py-px text-[11px] font-mono rounded bg-surface-3 text-text-secondary">
                              {k}: <span className={v === 1 ? 'text-accent-nosql' : 'text-amber-400'}>{v}</span>
                            </span>
                          ))}
                        </div>
                      </td>
                      <td className="px-4 py-2">{idx.unique && <span className="text-[11px] text-amber-400/80 font-medium">UNIQUE</span>}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}

            {tab === 'schema' && (
              <div className="p-4">
                <div className="p-4 rounded-lg bg-surface-1 border border-border-dim font-mono text-[12px] leading-relaxed text-text-secondary">
                  <p className="text-text-muted mb-1">// Inferred from sample</p>
                  <JsonNode value={{
                    _id: 'ObjectId', email: 'string', username: 'string',
                    profile: { firstName: 'string', lastName: 'string', avatar: 'string | null', bio: 'string', timezone: 'string' },
                    preferences: { theme: 'string', language: 'string', notifications: { email: 'boolean', push: 'boolean', sms: 'boolean' } },
                    roles: ['string'], status: 'string',
                    metadata: { loginCount: 'number', lastIp: 'string', devices: ['string'] },
                    createdAt: 'ISODate', updatedAt: 'ISODate',
                  }} />
                </div>
              </div>
            )}
          </div>
        </div>
      ) : (
        <div className="flex-1 flex items-center justify-center text-[13px] text-text-muted">
          No collections found
        </div>
      )}

      {/* Modals */}
      <Modal open={showCreate} onClose={() => setShowCreate(false)} title="Create Collection" subtitle="Add new collection">
        <div className="space-y-3">
          <div>
            <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Name</label>
            <input type="text" placeholder="my_collection" value={newCollName} onChange={e => setNewCollName(e.target.value)} className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary placeholder:text-text-muted outline-none focus:border-border-focus transition-colors font-mono" />
          </div>
          <div className="flex gap-2 justify-end pt-1">
            <button onClick={() => setShowCreate(false)} className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-surface-3 text-text-secondary hover:text-text-primary transition-colors">Cancel</button>
            <button
              onClick={async () => {
                try {
                  await api.createCollection(connectionId, newCollName)
                  toast.success('Created')
                  setShowCreate(false)
                  setNewCollName('')
                  const c = await api.getCollections(connectionId)
                  setCollections(c || [])
                } catch (err: any) {
                  toast.error(err.message || 'Failed to create collection')
                }
              }}
              className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-accent-nosql text-white hover:bg-accent-nosql/90 transition-colors"
            >
              Create
            </button>
          </div>
        </div>
      </Modal>

      <Modal open={showInsert} onClose={() => setShowInsert(false)} title="Insert Document" subtitle={selected?.name} width="max-w-xl">
        <div className="space-y-3">
          <JsonEditor
            value={insertJson}
            onChange={v => v !== undefined && setInsertJson(v)}
            height="260px"
          />
          <div className="flex gap-2 justify-end">
            <button onClick={() => setShowInsert(false)} className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-surface-3 text-text-secondary hover:text-text-primary transition-colors">Cancel</button>
            <button
              onClick={async () => {
                if (!selected) return
                try {
                  const doc = JSON.parse(insertJson)
                  await api.insertDocument(connectionId, selected.name, doc)
                  toast.success('Inserted')
                  setShowInsert(false)
                  const d = await api.getDocuments(connectionId, selected.name, '', 50)
                  setDocuments(d || [])
                  const c = await api.getCollections(connectionId)
                  setCollections(c || [])
                } catch (err: any) {
                  toast.error(err.message || 'Failed to insert document')
                }
              }}
              className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-accent-nosql text-white hover:bg-accent-nosql/90 transition-colors"
            >
              Insert
            </button>
          </div>
        </div>
      </Modal>

      {/* Edit document modal */}
      <Modal open={showEditDoc !== null} onClose={() => setShowEditDoc(null)} title="Edit Document" subtitle={showEditDoc?._id} width="max-w-xl">
        {showEditDoc && (
          <div className="space-y-3">
            <JsonEditor
              value={editDocJson}
              onChange={v => v !== undefined && setEditDocJson(v)}
              height="300px"
            />
            <div className="flex gap-2 justify-end">
              <button onClick={() => setShowEditDoc(null)} className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-surface-3 text-text-secondary hover:text-text-primary transition-colors">Cancel</button>
              <button
                onClick={async () => {
                  if (!selected) return
                  try {
                    const doc = JSON.parse(editDocJson)
                    await api.updateDocument(connectionId, selected.name, showEditDoc._id, doc)
                    toast.success('Updated')
                    setShowEditDoc(null)
                    const d = await api.getDocuments(connectionId, selected.name, '', 50)
                    setDocuments(d || [])
                  } catch (err: any) {
                    toast.error(err.message || 'Failed to update document')
                  }
                }}
                className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-accent-nosql text-white hover:bg-accent-nosql/90 transition-colors"
              >
                Save
              </button>
            </div>
          </div>
        )}
      </Modal>

      <ConfirmDialog
        open={deleteDocId !== null}
        onClose={() => setDeleteDocId(null)}
        onConfirm={async () => {
          if (!selected || !deleteDocId) return
          try {
            await api.deleteDocument(connectionId, selected.name, deleteDocId)
            toast.success('Deleted')
            setDeleteDocId(null)
            const d = await api.getDocuments(connectionId, selected.name, '', 50)
            setDocuments(d || [])
          } catch (err: any) {
            toast.error(err.message || 'Failed to delete document')
          }
        }}
        title="Delete Document"
        message={`Delete document ${deleteDocId}?`}
        confirmLabel="Delete"
      />
      <ConfirmDialog
        open={showDropColl}
        onClose={() => setShowDropColl(false)}
        onConfirm={async () => {
          if (!selected) return
          try {
            await api.dropCollection(connectionId, selected.name)
            toast.success('Dropped')
            setShowDropColl(false)
            const c = await api.getCollections(connectionId)
            setCollections(c || [])
            if (c?.length) setSelected(c[0])
            else setSelected(null)
          } catch (err: any) {
            toast.error(err.message || 'Failed to drop collection')
          }
        }}
        title="Drop Collection"
        message={selected ? `Drop "${selected.name}" with ${selected.documentCount.toLocaleString()} documents?` : ''}
        confirmLabel="Drop"
      />
    </div>
  )
}
