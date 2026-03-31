import { useState, useMemo, useEffect } from 'react'
import {
  Table2, Rows3, Key, Link2, Play, Clock, Hash, Plus, Trash2, Download, Pencil, Activity,
  History, CheckCircle2, XCircle, Copy, RotateCcw,
  ChevronRight, ChevronDown, Database,
  Eye, Code2, Zap, ListOrdered, Tags,
} from 'lucide-react'
import type { SQLTable, SQLColumn, SQLQueryResult, SQLView, SQLFunction, SQLTrigger, SQLSequence, SQLEnum, SQLObjectType, QueryHistoryEntry } from '../../types/database'
import { SQLEditor } from './SQLEditor'
import { ResizeHandle } from '../shared/ResizeHandle'
import { useResize } from '../../hooks/useResize'
import { Modal } from '../shared/Modal'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import * as api from '../../api'
import toast from 'react-hot-toast'

// --- Helpers ---
const typeColor: Record<string, string> = {
  uuid: 'text-purple-400', bigserial: 'text-blue-400', serial: 'text-blue-400',
  bigint: 'text-blue-400', integer: 'text-blue-400', boolean: 'text-pink-400',
  timestamptz: 'text-cyan-400', inet: 'text-teal-400', text: 'text-amber-300', jsonb: 'text-orange-400',
}
function getTypeColor(t: string) {
  return typeColor[t] || (t.startsWith('varchar') ? 'text-amber-300' : t.startsWith('numeric') ? 'text-emerald-400' : 'text-text-secondary')
}
function fmt(n: number) { return n.toLocaleString('en-US') }
function formatValue(v: unknown): string {
  if (v === null || v === undefined) return 'NULL'
  if (typeof v === 'boolean') return v ? 'true' : 'false'
  if (typeof v === 'string' && v.includes('T') && v.includes('Z')) return new Date(v).toLocaleString()
  return String(v)
}
function valueClass(v: unknown): string {
  if (v === null || v === undefined) return 'text-text-muted italic'
  if (typeof v === 'boolean') return v ? 'text-emerald-400' : 'text-red-400'
  if (typeof v === 'number') return 'text-blue-400'
  return 'text-text-primary'
}
function timeAgo(iso: string) {
  const diff = Date.now() - new Date(iso).getTime()
  const m = Math.floor(diff / 60_000)
  if (m < 1) return 'just now'
  if (m < 60) return `${m}m ago`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}
function truncateQuery(q: string, max = 80) {
  const oneLine = q.replace(/\s+/g, ' ').trim()
  return oneLine.length > max ? oneLine.slice(0, max) + '...' : oneLine
}
function fmtDuration(ms: number) { return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s` }

// --- Tree category config ---
const objectCategories: { key: SQLObjectType; label: string; icon: typeof Table2 }[] = [
  { key: 'table', label: 'Tables', icon: Table2 },
  { key: 'view', label: 'Views', icon: Eye },
  { key: 'function', label: 'Functions', icon: Code2 },
  { key: 'trigger', label: 'Triggers', icon: Zap },
  { key: 'sequence', label: 'Sequences', icon: ListOrdered },
  { key: 'enum', label: 'Types', icon: Tags },
]

// --- Selected object union ---
type SelectedObject =
  | { type: 'table'; obj: SQLTable }
  | { type: 'view'; obj: SQLView }
  | { type: 'function'; obj: SQLFunction }
  | { type: 'trigger'; obj: SQLTrigger }
  | { type: 'sequence'; obj: SQLSequence }
  | { type: 'enum'; obj: SQLEnum }
  | { type: 'query' }

// --- SQL definition viewer (reused for views, functions, triggers) ---
function DefinitionBlock({ sql }: { sql: string }) {
  return (
    <div className="p-4">
      <div className="relative">
        <button
          onClick={() => { navigator.clipboard.writeText(sql); toast.success('Copied') }}
          className="absolute right-3 top-3 p-1 rounded-md text-text-muted hover:text-text-primary bg-surface-3/80 hover:bg-surface-4 transition-colors z-10"
        >
          <Copy size={12} />
        </button>
        <pre className="p-4 rounded-lg bg-surface-1 border border-border-dim font-mono text-[12px] leading-relaxed text-text-secondary overflow-auto whitespace-pre-wrap">
          {sql}
        </pre>
      </div>
    </div>
  )
}

// --- Explain Plan Viewer ---
function ExplainPlanViewer({ plan }: { plan: ExplainPlan }) {
  return (
    <div className="border-b border-border-dim">
      <div className="flex items-center gap-4 px-4 py-1.5 bg-surface-1 border-b border-border-dim">
        <span className="flex items-center gap-1 text-[11px] text-amber-400 font-medium"><Activity size={10} /> Query Plan</span>
        <button
          onClick={() => { navigator.clipboard.writeText(plan.plan); toast.success('Plan copied') }}
          className="ml-auto p-0.5 rounded text-text-muted hover:text-text-primary transition-colors"
        >
          <Copy size={11} />
        </button>
      </div>
      <div className="overflow-auto max-h-[400px] p-4">
        <div className="font-mono text-[12px] leading-relaxed space-y-px">
          {plan.planRows.map((row, i) => (
            <div key={i} style={{ paddingLeft: row.level * 16 }} className={row.isNode ? 'text-text-primary' : 'text-text-muted'}>
              {row.isNode && <span className="text-accent-sql mr-1">{'\u2192'}</span>}
              <span className={row.isNode ? 'font-medium' : ''}>{row.text}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

// --- Query Editor ---
const defaultQuery = `SELECT id, email, username, role, is_active, last_login_at, created_at
FROM public.users
WHERE is_active = true
ORDER BY created_at DESC
LIMIT 50;`

interface ExplainPlan {
  plan: string
  format: string
  queryText: string
  planRows: { text: string; level: number; isNode: boolean }[]
}

function QueryEditor({ connectionId, onResult, onExplain, editorHeight, editorResize, historyWidth, historyResize, history, onHistoryRefresh, tables }: {
  connectionId: string
  onResult: (r: SQLQueryResult) => void
  onExplain: (plan: ExplainPlan | null) => void
  editorHeight: number; editorResize: (e: React.MouseEvent) => void
  historyWidth: number; historyResize: (e: React.MouseEvent) => void
  history: QueryHistoryEntry[]
  onHistoryRefresh: () => void
  tables: SQLTable[]
}) {
  const [query, setQuery] = useState(defaultQuery)
  const [showHistory, setShowHistory] = useState(false)

  const run = async () => {
    try {
      onExplain(null)
      const result = await api.executeQuery(connectionId, query)
      onResult(result)
      onHistoryRefresh()
    } catch (err: any) {
      toast.error(err.message || 'Query failed')
    }
  }

  const explain = async (analyze: boolean) => {
    try {
      const plan = await api.explainQuery(connectionId, query, analyze)
      onExplain(plan)
    } catch (err: any) {
      toast.error(err.message || 'Explain failed')
    }
  }

  const load = (entry: QueryHistoryEntry) => setQuery(entry.query)

  return (
    <div className="flex flex-col">
      <div className="flex">
        <div className="flex-1 relative min-w-0">
          <SQLEditor value={query} onChange={setQuery} onRun={run} height={`${editorHeight}px`} tables={tables} />
          <div className="absolute right-3 bottom-3 flex items-center gap-1.5 z-10">
            <button onClick={() => setShowHistory(!showHistory)} className={`flex items-center gap-1 px-2 py-1 text-[12px] font-medium rounded-md transition-colors ${showHistory ? 'bg-accent-sql/15 text-accent-sql' : 'text-text-muted hover:text-text-secondary bg-surface-3 hover:bg-surface-4'}`}>
              <History size={12} /> <span className="hidden sm:inline">History</span>
            </button>
            <button onClick={() => explain(false)} className="flex items-center gap-1 px-2 py-1 text-[12px] font-medium rounded-md text-text-muted hover:text-text-secondary bg-surface-3 hover:bg-surface-4 transition-colors" title="Show query execution plan">
              <Activity size={11} /> Explain
            </button>
            <button onClick={() => explain(true)} className="flex items-center gap-1 px-2 py-1 text-[12px] font-medium rounded-md text-amber-400/80 hover:text-amber-400 bg-amber-500/8 hover:bg-amber-500/15 transition-colors" title="Execute and show actual plan with timing">
              <Activity size={11} /> Analyze
            </button>
            <span className="text-[10px] text-text-muted font-mono">{'\u2318'}Enter</span>
            <button onClick={run} className="flex items-center gap-1 px-2.5 py-1 text-[12px] font-medium rounded-md bg-accent-sql/90 text-white hover:bg-accent-sql transition-colors shadow-lg shadow-black/30">
              <Play size={11} fill="currentColor" /> Run
            </button>
          </div>
        </div>
        {showHistory && (
          <>
            <ResizeHandle direction="horizontal" onMouseDown={historyResize} />
            <div className="shrink-0 bg-surface-1 flex flex-col overflow-hidden" style={{ width: historyWidth, height: editorHeight }}>
              <div className="h-8 flex items-center justify-between px-3 border-b border-border-dim shrink-0">
                <span className="text-[11px] font-medium text-text-muted tracking-widest uppercase">History</span>
                <span className="text-[10px] text-text-muted tabular-nums">{history.length}</span>
              </div>
              <div className="flex-1 overflow-y-auto">
                {history.map(entry => (
                  <div key={entry.id} className="group px-3 py-2 border-b border-border-dim/50 hover:bg-surface-hover/40 cursor-pointer transition-colors" onClick={() => load(entry)}>
                    <div className="flex items-center gap-1.5 mb-1">
                      {entry.status === 'success' ? <CheckCircle2 size={10} className="text-emerald-400/70 shrink-0" /> : <XCircle size={10} className="text-red-400/70 shrink-0" />}
                      <span className="text-[11px] text-text-muted">{timeAgo(entry.executedAt)}</span>
                      <span className="text-[10px] text-text-muted font-mono">{fmtDuration(entry.duration)}</span>
                      {entry.status === 'success' && <span className="text-[10px] text-text-muted font-mono">{entry.rowCount.toLocaleString()} rows</span>}
                      <div className="ml-auto flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                        <button onClick={e => { e.stopPropagation(); navigator.clipboard.writeText(entry.query); toast.success('Copied') }} className="p-0.5 rounded text-text-muted hover:text-text-primary transition-colors"><Copy size={10} /></button>
                        <button onClick={e => { e.stopPropagation(); load(entry) }} className="p-0.5 rounded text-text-muted hover:text-accent-sql transition-colors"><RotateCcw size={10} /></button>
                      </div>
                    </div>
                    <p className="text-[11px] font-mono text-text-secondary leading-snug truncate">{truncateQuery(entry.query)}</p>
                    {entry.error && <p className="text-[10px] text-red-400/70 mt-0.5 truncate">{entry.error}</p>}
                  </div>
                ))}
                {history.length === 0 && (
                  <div className="flex items-center justify-center h-full text-[12px] text-text-muted py-8">No history yet</div>
                )}
              </div>
            </div>
          </>
        )}
      </div>
      <ResizeHandle direction="vertical" onMouseDown={editorResize} />
    </div>
  )
}

type EditableRow = Record<string, unknown> & { __rowIdx: number }

function ResultsTable({ result }: { result: SQLQueryResult }) {
  const [rows, setRows] = useState<EditableRow[]>(() => result.rows.map((r, i) => ({ ...r, __rowIdx: i })))
  const [editingCell, setEditingCell] = useState<{ row: number; col: string } | null>(null)
  const [editValue, setEditValue] = useState('')
  const [modifiedCells, setModifiedCells] = useState<Set<string>>(new Set())
  const [deletedRows, setDeletedRows] = useState<Set<number>>(new Set())
  const [showDeleteConfirm, setShowDeleteConfirm] = useState<number | null>(null)

  const hasChanges = modifiedCells.size > 0 || deletedRows.size > 0
  const visibleRows = rows.filter(r => !deletedRows.has(r.__rowIdx as number))

  const startEdit = (rowIdx: number, col: string, value: unknown) => {
    setEditingCell({ row: rowIdx, col })
    setEditValue(value === null || value === undefined ? '' : String(value))
  }

  const commitEdit = () => {
    if (!editingCell) return
    const { row, col } = editingCell
    setRows(prev => prev.map(r =>
      (r.__rowIdx as number) === row ? { ...r, [col]: editValue === '' ? null : editValue } : r
    ))
    setModifiedCells(prev => new Set(prev).add(`${row}:${col}`))
    setEditingCell(null)
  }

  const cancelEdit = () => setEditingCell(null)

  const deleteRow = (rowIdx: number) => {
    setDeletedRows(prev => new Set(prev).add(rowIdx))
    setShowDeleteConfirm(null)
  }

  const applyChanges = () => {
    const mods = Array.from(modifiedCells).length
    const dels = deletedRows.size
    const parts = []
    if (mods > 0) parts.push(`${mods} cell${mods > 1 ? 's' : ''} updated`)
    if (dels > 0) parts.push(`${dels} row${dels > 1 ? 's' : ''} deleted`)
    toast.success(parts.join(', '))
    setModifiedCells(new Set())
    setDeletedRows(new Set())
  }

  const discardChanges = () => {
    setRows(result.rows.map((r, i) => ({ ...r, __rowIdx: i })))
    setModifiedCells(new Set())
    setDeletedRows(new Set())
    setEditingCell(null)
  }

  return (
    <div className="border-b border-border-dim">
      <div className="flex items-center gap-4 px-4 py-1.5 bg-surface-1 border-b border-border-dim">
        <span className="flex items-center gap-1 text-[11px] text-text-muted font-mono"><Clock size={10} /> {result.executionTime}ms</span>
        <span className="flex items-center gap-1 text-[11px] text-text-muted font-mono"><Rows3 size={10} /> {fmt(result.rowCount)} rows</span>
        <span className="text-[11px] text-emerald-400 font-medium">OK</span>
        {hasChanges && (
          <div className="ml-auto flex items-center gap-1.5">
            <span className="text-[11px] text-amber-400 font-medium">
              {modifiedCells.size > 0 && `${modifiedCells.size} modified`}
              {modifiedCells.size > 0 && deletedRows.size > 0 && ', '}
              {deletedRows.size > 0 && `${deletedRows.size} deleted`}
            </span>
            <button onClick={discardChanges} className="px-2 py-0.5 text-[11px] font-medium rounded-md text-text-muted hover:text-text-primary bg-surface-3 hover:bg-surface-4 transition-colors">
              Discard
            </button>
            <button onClick={applyChanges} className="px-2 py-0.5 text-[11px] font-medium rounded-md bg-accent-sql/90 text-white hover:bg-accent-sql transition-colors">
              Apply
            </button>
          </div>
        )}
      </div>
      <div className="overflow-auto max-h-[300px]">
        <table className="w-full text-[12px] font-mono">
          <thead><tr className="bg-surface-1 sticky top-0 z-10">
            <th className="px-3 py-1.5 text-left text-[10px] font-medium text-text-muted tracking-wider border-b border-border-dim w-8">#</th>
            {result.columns.map(c => <th key={c} className="px-3 py-1.5 text-left text-[10px] font-medium text-text-muted tracking-wider border-b border-border-dim whitespace-nowrap">{c}</th>)}
            <th className="px-2 py-1.5 border-b border-border-dim w-8"></th>
          </tr></thead>
          <tbody>{visibleRows.map((row, vi) => {
            const rowIdx = row.__rowIdx as number
            return (
              <tr key={rowIdx} className="border-b border-border-dim/50 hover:bg-surface-hover/30 transition-colors group">
                <td className="px-3 py-1 text-text-muted">{vi + 1}</td>
                {result.columns.map(c => {
                  const cellKey = `${rowIdx}:${c}`
                  const isEditing = editingCell?.row === rowIdx && editingCell?.col === c
                  const isModified = modifiedCells.has(cellKey)

                  if (isEditing) {
                    return (
                      <td key={c} className="px-1 py-0.5">
                        <input
                          autoFocus
                          value={editValue}
                          onChange={e => setEditValue(e.target.value)}
                          onKeyDown={e => {
                            if (e.key === 'Enter') commitEdit()
                            if (e.key === 'Escape') cancelEdit()
                            if (e.key === 'Tab') { e.preventDefault(); commitEdit(); const cols = result.columns; const idx = cols.indexOf(c); if (idx < cols.length - 1) startEdit(rowIdx, cols[idx + 1], row[cols[idx + 1]]) }
                          }}
                          onBlur={commitEdit}
                          className="w-full px-1.5 py-0.5 text-[12px] font-mono bg-surface-3 border border-accent-sql/40 rounded text-text-primary outline-none"
                        />
                      </td>
                    )
                  }

                  return (
                    <td
                      key={c}
                      className={`px-3 py-1 whitespace-nowrap max-w-[280px] truncate cursor-pointer
                        ${isModified ? 'bg-amber-500/8' : ''}
                        ${valueClass(row[c])}`}
                      onDoubleClick={() => startEdit(rowIdx, c, row[c])}
                      title="Double-click to edit"
                    >
                      {isModified && <span className="inline-block w-1 h-1 rounded-full bg-amber-400 mr-1.5 align-middle" />}
                      {formatValue(row[c])}
                    </td>
                  )
                })}
                <td className="px-2 py-1">
                  <button
                    onClick={() => setShowDeleteConfirm(rowIdx)}
                    className="p-0.5 rounded text-text-muted opacity-0 group-hover:opacity-100 hover:text-red-400 hover:bg-red-500/10 transition-all"
                    title="Delete row"
                  >
                    <Trash2 size={11} />
                  </button>
                </td>
              </tr>
            )
          })}</tbody>
        </table>
      </div>
      <ConfirmDialog
        open={showDeleteConfirm !== null}
        onClose={() => setShowDeleteConfirm(null)}
        onConfirm={() => { if (showDeleteConfirm !== null) deleteRow(showDeleteConfirm) }}
        title="Delete Row"
        message="Remove this row? This will be applied when you click Apply."
        confirmLabel="Delete"
      />
    </div>
  )
}

// --- Columns table (shared by tables and views) ---
function ColumnsTable({ columns, onEdit, onDrop }: { columns: SQLColumn[]; onEdit?: (col: SQLColumn) => void; onDrop?: (col: SQLColumn) => void }) {
  return (
    <table className="w-full text-[13px]">
      <thead><tr className="bg-surface-1 sticky top-0 z-10 border-b border-border-dim">
        {['Name', 'Type', 'Nullable', 'Default', 'Keys', ''].map(h => <th key={h} className="px-4 py-2 text-left text-[10px] font-medium text-text-muted tracking-widest uppercase">{h}</th>)}
      </tr></thead>
      <tbody>{columns.map((col) => (
        <tr key={col.name} className="border-b border-border-dim/50 hover:bg-surface-hover/30 transition-colors group">
          <td className="px-4 py-2 font-mono font-medium text-text-primary text-[12px]">{col.name}</td>
          <td className="px-4 py-2"><span className={`font-mono text-[12px] ${getTypeColor(col.type)}`}>{col.type}</span></td>
          <td className="px-4 py-2"><span className={`text-[12px] ${col.nullable ? 'text-text-muted' : 'text-amber-400/80'}`}>{col.nullable ? 'yes' : 'NOT NULL'}</span></td>
          <td className="px-4 py-2 font-mono text-[12px] text-text-muted">{col.defaultValue || '\u2014'}</td>
          <td className="px-4 py-2"><div className="flex gap-1">
            {col.primaryKey && <span className="inline-flex items-center gap-0.5 px-1.5 py-px text-[10px] font-semibold rounded bg-amber-500/10 text-amber-400/90"><Key size={9} /> PK</span>}
            {col.foreignKey && <span className="inline-flex items-center gap-0.5 px-1.5 py-px text-[10px] font-semibold rounded bg-blue-500/10 text-blue-400/90" title={`${col.foreignKey.table}.${col.foreignKey.column}`}><Link2 size={9} /> FK</span>}
          </div></td>
          <td className="px-2 py-2">
            {(onEdit || onDrop) && (
              <div className="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                {onEdit && <button onClick={() => onEdit(col)} className="p-0.5 rounded text-text-muted hover:text-text-primary transition-colors" title="Edit column"><Pencil size={11} /></button>}
                {onDrop && <button onClick={() => onDrop(col)} className="p-0.5 rounded text-text-muted hover:text-red-400 transition-colors" title="Drop column"><Trash2 size={11} /></button>}
              </div>
            )}
          </td>
        </tr>
      ))}</tbody>
    </table>
  )
}

// --- Main Panel ---
interface SQLPanelProps {
  connectionId: string
}

export function SQLPanel({ connectionId }: SQLPanelProps) {
  const [selected, setSelected] = useState<SelectedObject>({ type: 'query' })
  const [detailTab, setDetailTab] = useState<string>('query')
  const [queryResult, setQueryResult] = useState<SQLQueryResult | null>(null)
  const [explainPlan, setExplainPlan] = useState<ExplainPlan | null>(null)
  const [showAddCol, setShowAddCol] = useState(false)
  const [showDropTable, setShowDropTable] = useState(false)
  const [editCol, setEditCol] = useState<SQLColumn | null>(null)
  const [dropCol, setDropCol] = useState<SQLColumn | null>(null)

  // Add column form state
  const [addColName, setAddColName] = useState('')
  const [addColType, setAddColType] = useState('varchar(255)')
  const [addColDefault, setAddColDefault] = useState('')
  const [addColNullable, setAddColNullable] = useState(true)

  // Edit column form state
  const [editColName, setEditColName] = useState('')
  const [editColType, setEditColType] = useState('')
  const [editColDefault, setEditColDefault] = useState('')
  const [editColNullable, setEditColNullable] = useState(true)

  // Data from backend
  const [tables, setTables] = useState<SQLTable[]>([])
  const [views, setViews] = useState<SQLView[]>([])
  const [functions, setFunctions] = useState<SQLFunction[]>([])
  const [triggers, setTriggers] = useState<SQLTrigger[]>([])
  const [sequences, setSequences] = useState<SQLSequence[]>([])
  const [enums, setEnums] = useState<SQLEnum[]>([])
  const [history, setHistory] = useState<QueryHistoryEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [tableData, setTableData] = useState<SQLQueryResult | null>(null)

  const treeW = useResize({ direction: 'horizontal', initialSize: 240, minSize: 160, maxSize: 420 })
  const editorH = useResize({ direction: 'vertical', initialSize: 160, minSize: 80, maxSize: 500 })
  const historyW = useResize({ direction: 'horizontal', initialSize: 300, minSize: 180, maxSize: 500, inverted: true })

  // Load all schema data when connectionId changes
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    Promise.all([
      api.getTables(connectionId),
      api.getViews(connectionId),
      api.getFunctions(connectionId),
      api.getTriggers(connectionId),
      api.getSequences(connectionId),
      api.getEnums(connectionId),
      api.getQueryHistory(connectionId),
    ]).then(([t, v, f, tr, s, e, h]) => {
      if (cancelled) return
      setTables(t || [])
      setViews(v || [])
      setFunctions(f || [])
      setTriggers(tr || [])
      setSequences(s || [])
      setEnums(e || [])
      setHistory(h || [])
      if (t?.length) {
        setSelected({ type: 'table', obj: t[0] })
        setDetailTab('columns')
      }
      setLoading(false)
    }).catch(err => {
      if (!cancelled) { console.error(err); setLoading(false) }
    })
    return () => { cancelled = true }
  }, [connectionId])

  // Load table data when data tab is selected for a table
  useEffect(() => {
    if (selected.type !== 'table' || detailTab !== 'data') return
    let cancelled = false
    api.getTableData(connectionId, selected.obj.schema, selected.obj.name, 50, 0)
      .then(result => { if (!cancelled) setTableData(result) })
      .catch(err => { if (!cancelled) console.error(err) })
    return () => { cancelled = true }
  }, [connectionId, selected.type === 'table' ? `${selected.obj.schema}.${selected.obj.name}` : null, detailTab])

  // Init edit column form when editCol changes
  useEffect(() => {
    if (editCol) {
      setEditColName(editCol.name)
      setEditColType(editCol.type)
      setEditColDefault(editCol.defaultValue || '')
      setEditColNullable(editCol.nullable)
    }
  }, [editCol])

  const refreshTables = async () => {
    const t = await api.getTables(connectionId)
    setTables(t || [])
    if (selected.type === 'table') {
      const updated = (t || []).find((tb: SQLTable) => tb.schema === selected.obj.schema && tb.name === (editCol ? editColName : selected.obj.name))
        || (t || []).find((tb: SQLTable) => tb.schema === selected.obj.schema)
      if (updated) setSelected({ type: 'table', obj: updated })
    }
  }

  const refreshHistory = async () => {
    try {
      const h = await api.getQueryHistory(connectionId)
      setHistory(h || [])
    } catch { /* ignore */ }
  }

  // Build schema -> category -> objects tree
  const schemaTree = useMemo(() => {
    const allSchemas = new Set<string>()
    tables.forEach(t => allSchemas.add(t.schema))
    views.forEach(v => allSchemas.add(v.schema))
    functions.forEach(f => allSchemas.add(f.schema))
    triggers.forEach(t => allSchemas.add(t.schema))
    sequences.forEach(s => allSchemas.add(s.schema))
    enums.forEach(e => allSchemas.add(e.schema))

    const sorted = Array.from(allSchemas).sort((a, b) => {
      if (a === 'public') return -1; if (b === 'public') return 1; return a.localeCompare(b)
    })

    return sorted.map(schema => ({
      schema,
      tables: tables.filter(t => t.schema === schema),
      views: views.filter(v => v.schema === schema),
      functions: functions.filter(f => f.schema === schema),
      triggers: triggers.filter(t => t.schema === schema),
      sequences: sequences.filter(s => s.schema === schema),
      enums: enums.filter(e => e.schema === schema),
    }))
  }, [tables, views, functions, triggers, sequences, enums])

  const [expandedSchemas, setExpandedSchemas] = useState<Set<string>>(new Set())
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set())

  // Auto-expand schemas when data loads
  useEffect(() => {
    if (schemaTree.length > 0) {
      setExpandedSchemas(new Set(schemaTree.map(s => s.schema)))
      setExpandedCategories(new Set(schemaTree.map(s => `${s.schema}:table`)))
    }
  }, [schemaTree])

  const toggleSchema = (s: string) => setExpandedSchemas(prev => { const n = new Set(prev); n.has(s) ? n.delete(s) : n.add(s); return n })
  const toggleCat = (key: string) => setExpandedCategories(prev => { const n = new Set(prev); n.has(key) ? n.delete(key) : n.add(key); return n })

  const selectObj = (sel: SelectedObject) => {
    setSelected(sel)
    if (sel.type === 'table') setDetailTab('columns')
    else if (sel.type === 'view') setDetailTab('columns')
    else if (sel.type === 'function') setDetailTab('definition')
    else if (sel.type === 'trigger') setDetailTab('definition')
    else if (sel.type === 'sequence') setDetailTab('info')
    else if (sel.type === 'enum') setDetailTab('values')
    else setDetailTab('columns')
  }

  const isSelected = (type: SQLObjectType, name: string, schema: string) =>
    selected.type === type && 'obj' in selected && selected.obj.name === name && selected.obj.schema === schema

  const totalObjects = tables.length + views.length + functions.length + triggers.length + sequences.length + enums.length

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center text-[13px] text-text-muted">
        Loading...
      </div>
    )
  }

  return (
    <div className="flex h-full">
      {/* LEFT: Schema tree */}
      <div className="shrink-0 border-r border-border-dim flex flex-col bg-surface-1" style={{ width: treeW.size }}>
        <div className="h-9 flex items-center px-3 border-b border-border-dim shrink-0">
          <span className="text-[11px] font-medium text-text-muted tracking-widest uppercase">Objects</span>
          <span className="ml-auto text-[10px] text-text-muted tabular-nums">{totalObjects}</span>
        </div>
        {/* Query shortcut */}
        <div className="px-2 pt-1.5 pb-0.5">
          <button
            onClick={() => { setSelected({ type: 'query' }); setDetailTab('query') }}
            className={`w-full flex items-center gap-1.5 px-2 py-[4px] rounded-md text-[12px] transition-colors
              ${selected.type === 'query' ? 'bg-accent-sql/10 text-accent-sql' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}`}
          >
            <Play size={11} /> Query Editor
          </button>
        </div>
        <div className="flex-1 overflow-y-auto py-0.5">
          {schemaTree.map(({ schema, tables: schemaTables, views: schemaViews, functions: schemaFunctions, triggers: schemaTriggers, sequences: schemaSequences, enums: schemaEnums }) => {
            const isSchemaOpen = expandedSchemas.has(schema)
            const items: [SQLObjectType, string, { name: string; schema: string }[]][] = [
              ['table', 'Tables', schemaTables],
              ['view', 'Views', schemaViews],
              ['function', 'Functions', schemaFunctions],
              ['trigger', 'Triggers', schemaTriggers],
              ['sequence', 'Sequences', schemaSequences],
              ['enum', 'Types', schemaEnums],
            ]
            const totalInSchema = items.reduce((s, [, , arr]) => s + arr.length, 0)

            return (
              <div key={schema}>
                <button onClick={() => toggleSchema(schema)} className="w-full flex items-center gap-1.5 px-2 py-[4px] text-left hover:bg-surface-hover/40 transition-colors">
                  {isSchemaOpen ? <ChevronDown size={11} className="text-text-muted shrink-0" /> : <ChevronRight size={11} className="text-text-muted shrink-0" />}
                  <Database size={12} className={isSchemaOpen ? 'text-accent-sql/60 shrink-0' : 'text-text-muted shrink-0'} />
                  <span className="text-[12px] font-mono text-text-secondary">{schema}</span>
                  <span className="ml-auto text-[9px] text-text-muted pr-1">{totalInSchema}</span>
                </button>

                {isSchemaOpen && items.map(([catKey, catLabel, arr]) => {
                  if (arr.length === 0) return null
                  const catId = `${schema}:${catKey}`
                  const isCatOpen = expandedCategories.has(catId)
                  const CatIcon = objectCategories.find(c => c.key === catKey)!.icon

                  return (
                    <div key={catId}>
                      <button onClick={() => toggleCat(catId)} className="w-full flex items-center gap-1.5 py-[3px] text-left hover:bg-surface-hover/30 transition-colors" style={{ paddingLeft: 22 }}>
                        {isCatOpen ? <ChevronDown size={10} className="text-text-muted shrink-0" /> : <ChevronRight size={10} className="text-text-muted shrink-0" />}
                        <CatIcon size={11} className="text-text-muted shrink-0" />
                        <span className="text-[11px] text-text-muted">{catLabel}</span>
                        <span className="ml-auto text-[9px] text-text-muted pr-2">{arr.length}</span>
                      </button>

                      {isCatOpen && arr.map(item => {
                        const active = isSelected(catKey, item.name, item.schema)
                        return (
                          <button
                            key={`${item.schema}.${item.name}`}
                            onClick={() => selectObj({ type: catKey, obj: item as never })}
                            className={`w-full flex items-center gap-1.5 py-[3px] text-[12px] transition-colors
                              ${active ? 'bg-accent-sql/8 text-text-primary' : 'text-text-secondary hover:text-text-primary hover:bg-surface-hover'}`}
                            style={{ paddingLeft: 40 }}
                          >
                            <span className="font-mono truncate text-[11px]">{item.name}</span>
                            {catKey === 'table' && <span className="ml-auto text-[9px] text-text-muted font-mono pr-2">{fmt((item as SQLTable).rowCount)}</span>}
                            {catKey === 'view' && (item as SQLView).materialized && <span className="ml-auto text-[8px] text-amber-400/70 font-medium pr-2">MAT</span>}
                            {catKey === 'trigger' && !(item as SQLTrigger).enabled && <span className="ml-auto text-[8px] text-red-400/70 font-medium pr-2">OFF</span>}
                          </button>
                        )
                      })}
                    </div>
                  )
                })}
              </div>
            )
          })}
          {totalObjects === 0 && (
            <div className="flex items-center justify-center h-32 text-[12px] text-text-muted">No objects found</div>
          )}
        </div>
      </div>

      <ResizeHandle direction="horizontal" onMouseDown={treeW.onMouseDown} />

      {/* RIGHT: Detail pane */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* === QUERY === */}
        {selected.type === 'query' && (
          <>
            <div className="h-9 flex items-center gap-3 px-4 border-b border-border-dim bg-surface-1 shrink-0">
              <Play size={14} className="text-accent-sql" />
              <span className="text-[13px] font-medium text-text-primary">Query Editor</span>
            </div>
            <QueryEditor connectionId={connectionId} onResult={r => { setQueryResult(r); setExplainPlan(null) }} onExplain={p => { setExplainPlan(p); if (p) setQueryResult(null) }} editorHeight={editorH.size} editorResize={editorH.onMouseDown} historyWidth={historyW.size} historyResize={historyW.onMouseDown} history={history} onHistoryRefresh={refreshHistory} tables={tables} />
            {queryResult && <ResultsTable result={queryResult} />}
            {explainPlan && <ExplainPlanViewer plan={explainPlan} />}
            {!queryResult && !explainPlan && <div className="flex-1 flex flex-col items-center justify-center text-center py-16"><Hash size={20} className="text-text-muted mb-3" /><p className="text-[13px] text-text-muted">Run a query to see results</p></div>}
          </>
        )}

        {/* === TABLE === */}
        {selected.type === 'table' && (
          <>
            <div className="h-9 flex items-center gap-3 px-4 border-b border-border-dim bg-surface-1 shrink-0">
              <Table2 size={14} className="text-accent-sql" />
              <span className="text-[13px] font-medium text-text-primary font-mono">{selected.obj.schema}.{selected.obj.name}</span>
              <span className="text-[11px] text-text-muted">{selected.obj.columns.length} cols</span>
              <span className="text-[11px] text-text-muted font-mono">{fmt(selected.obj.rowCount)} rows</span>
              <span className="text-[11px] text-text-muted font-mono">{selected.obj.size}</span>
              <div className="ml-auto flex items-center gap-0.5">
                {(['columns', 'indexes', 'data', 'query'] as const).map(t => <button key={t} onClick={() => setDetailTab(t)} className={`px-2.5 py-1 text-[12px] font-medium rounded-md capitalize transition-colors ${detailTab === t ? 'bg-surface-3 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}>{t}</button>)}
                <div className="w-px h-4 bg-border-dim mx-1" />
                <button onClick={() => setShowAddCol(true)} className="p-1 rounded-md text-text-muted hover:text-accent-sql transition-colors" title="Add column"><Plus size={14} /></button>
                <button onClick={() => toast.success('Exported')} className="p-1 rounded-md text-text-muted hover:text-text-secondary transition-colors" title="Export"><Download size={14} /></button>
                <button onClick={() => setShowDropTable(true)} className="p-1 rounded-md text-text-muted hover:text-red-400 transition-colors" title="Drop table"><Trash2 size={14} /></button>
              </div>
            </div>
            {detailTab === 'query' && (
              <QueryEditor connectionId={connectionId} onResult={r => { setQueryResult(r); setExplainPlan(null) }} onExplain={p => { setExplainPlan(p); if (p) setQueryResult(null) }} editorHeight={editorH.size} editorResize={editorH.onMouseDown} historyWidth={historyW.size} historyResize={historyW.onMouseDown} history={history} onHistoryRefresh={refreshHistory} tables={tables} />
            )}
            {detailTab === 'query' && queryResult && <ResultsTable result={queryResult} />}
            {detailTab === 'query' && explainPlan && <ExplainPlanViewer plan={explainPlan} />}
            <div className="flex-1 overflow-auto">
              {detailTab === 'columns' && <ColumnsTable columns={selected.obj.columns} onEdit={col => setEditCol(col)} onDrop={col => setDropCol(col)} />}
              {detailTab === 'indexes' && (
                <table className="w-full text-[13px]"><thead><tr className="bg-surface-1 sticky top-0 z-10 border-b border-border-dim">
                  {['Name', 'Columns', 'Type', 'Unique'].map(h => <th key={h} className="px-4 py-2 text-left text-[10px] font-medium text-text-muted tracking-widest uppercase">{h}</th>)}
                </tr></thead><tbody>{selected.obj.indexes.map((idx) => (
                  <tr key={idx.name} className="border-b border-border-dim/50 hover:bg-surface-hover/30 transition-colors">
                    <td className="px-4 py-2 font-mono text-[12px] text-text-primary">{idx.name}</td>
                    <td className="px-4 py-2"><div className="flex gap-1 flex-wrap">{idx.columns.map(c => <span key={c} className="px-1.5 py-px text-[11px] font-mono rounded bg-surface-3 text-text-secondary">{c}</span>)}</div></td>
                    <td className="px-4 py-2 text-[12px] text-text-muted uppercase">{idx.type}</td>
                    <td className="px-4 py-2">{idx.unique && <span className="text-[11px] text-amber-400/80 font-medium">UNIQUE</span>}</td>
                  </tr>
                ))}</tbody></table>
              )}
              {detailTab === 'data' && tableData && <ResultsTable result={tableData} />}
              {detailTab === 'data' && !tableData && (
                <div className="flex items-center justify-center h-full text-[13px] text-text-muted">Loading data...</div>
              )}
              {detailTab === 'query' && !queryResult && (
                <div className="flex flex-col items-center justify-center h-full text-center py-16">
                  <Hash size={20} className="text-text-muted mb-3" />
                  <p className="text-[13px] text-text-muted">Run a query to see results</p>
                </div>
              )}
            </div>
          </>
        )}

        {/* === VIEW === */}
        {selected.type === 'view' && (
          <>
            <div className="h-9 flex items-center gap-3 px-4 border-b border-border-dim bg-surface-1 shrink-0">
              <Eye size={14} className="text-accent-sql" />
              <span className="text-[13px] font-medium text-text-primary font-mono">{selected.obj.schema}.{selected.obj.name}</span>
              {selected.obj.materialized && <span className="text-[10px] font-semibold text-amber-400 bg-amber-500/10 px-1.5 py-px rounded">MATERIALIZED</span>}
              <span className="text-[11px] text-text-muted">{selected.obj.columns.length} cols</span>
              <div className="ml-auto flex items-center gap-0.5">
                {(['columns', 'definition'] as const).map(t => <button key={t} onClick={() => setDetailTab(t)} className={`px-2.5 py-1 text-[12px] font-medium rounded-md capitalize transition-colors ${detailTab === t ? 'bg-surface-3 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}>{t}</button>)}
                {selected.obj.materialized && (
                  <>
                    <div className="w-px h-4 bg-border-dim mx-1" />
                    <button
                      onClick={async () => {
                        try {
                          await api.refreshMaterializedView(connectionId, selected.obj.schema, selected.obj.name)
                          toast.success('View refreshed')
                        } catch (err: any) {
                          toast.error(err.message || 'Failed to refresh view')
                        }
                      }}
                      className="p-1 rounded-md text-text-muted hover:text-accent-sql transition-colors"
                      title="Refresh materialized view"
                    >
                      <RotateCcw size={14} />
                    </button>
                  </>
                )}
              </div>
            </div>
            <div className="flex-1 overflow-auto">
              {detailTab === 'columns' && <ColumnsTable columns={selected.obj.columns} />}
              {detailTab === 'definition' && <DefinitionBlock sql={selected.obj.definition} />}
            </div>
          </>
        )}

        {/* === FUNCTION === */}
        {selected.type === 'function' && (
          <>
            <div className="h-9 flex items-center gap-3 px-4 border-b border-border-dim bg-surface-1 shrink-0">
              <Code2 size={14} className="text-accent-sql" />
              <span className="text-[13px] font-medium text-text-primary font-mono">{selected.obj.schema}.{selected.obj.name}()</span>
              <span className="text-[10px] font-mono text-text-muted">{selected.obj.language}</span>
              <span className="text-[10px] text-text-muted">{'\u2192'} {selected.obj.returnType}</span>
              <span className={`text-[9px] font-semibold px-1.5 py-px rounded ${selected.obj.volatility === 'IMMUTABLE' ? 'text-emerald-400 bg-emerald-500/10' : selected.obj.volatility === 'STABLE' ? 'text-blue-400 bg-blue-500/10' : 'text-amber-400 bg-amber-500/10'}`}>{selected.obj.volatility}</span>
              <div className="ml-auto flex items-center gap-0.5">
                {(['definition', 'args'] as const).map(t => <button key={t} onClick={() => setDetailTab(t)} className={`px-2.5 py-1 text-[12px] font-medium rounded-md capitalize transition-colors ${detailTab === t ? 'bg-surface-3 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}>{t === 'args' ? `Args (${selected.obj.args.length})` : t}</button>)}
              </div>
            </div>
            <div className="flex-1 overflow-auto">
              {detailTab === 'definition' && <DefinitionBlock sql={selected.obj.definition} />}
              {detailTab === 'args' && (
                selected.obj.args.length > 0 ? (
                  <table className="w-full text-[13px]"><thead><tr className="bg-surface-1 sticky top-0 z-10 border-b border-border-dim">
                    {['Name', 'Type', 'Mode'].map(h => <th key={h} className="px-4 py-2 text-left text-[10px] font-medium text-text-muted tracking-widest uppercase">{h}</th>)}
                  </tr></thead><tbody>{selected.obj.args.map((a, i) => (
                    <tr key={i} className="border-b border-border-dim/50"><td className="px-4 py-2 font-mono text-[12px] text-text-primary">{a.name}</td><td className="px-4 py-2"><span className={`font-mono text-[12px] ${getTypeColor(a.type)}`}>{a.type}</span></td><td className="px-4 py-2 text-[12px] text-text-muted">{a.mode}</td></tr>
                  ))}</tbody></table>
                ) : <div className="flex items-center justify-center h-full text-[13px] text-text-muted">No arguments</div>
              )}
            </div>
          </>
        )}

        {/* === TRIGGER === */}
        {selected.type === 'trigger' && (
          <>
            <div className="h-9 flex items-center gap-3 px-4 border-b border-border-dim bg-surface-1 shrink-0">
              <Zap size={14} className="text-accent-sql" />
              <span className="text-[13px] font-medium text-text-primary font-mono">{selected.obj.name}</span>
              <span className={`text-[10px] font-semibold px-1.5 py-px rounded ${selected.obj.enabled ? 'text-emerald-400 bg-emerald-500/10' : 'text-red-400 bg-red-500/10'}`}>{selected.obj.enabled ? 'ENABLED' : 'DISABLED'}</span>
              <div className="ml-auto flex items-center gap-0.5">
                {(['info', 'definition'] as const).map(t => <button key={t} onClick={() => setDetailTab(t)} className={`px-2.5 py-1 text-[12px] font-medium rounded-md capitalize transition-colors ${detailTab === t ? 'bg-surface-3 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}>{t}</button>)}
              </div>
            </div>
            <div className="flex-1 overflow-auto">
              {detailTab === 'info' && (
                <div className="p-4 space-y-3">
                  {[
                    ['Table', `${selected.obj.schema}.${selected.obj.table}`],
                    ['Timing', selected.obj.timing],
                    ['Event', selected.obj.event],
                    ['For Each', selected.obj.forEach],
                    ['Function', selected.obj.function],
                    ['Status', selected.obj.enabled ? 'Enabled' : 'Disabled'],
                  ].map(([label, value]) => (
                    <div key={label} className="flex items-baseline gap-3">
                      <span className="text-[11px] text-text-muted uppercase tracking-wider w-24 shrink-0">{label}</span>
                      <span className="text-[13px] font-mono text-text-primary">{value}</span>
                    </div>
                  ))}
                </div>
              )}
              {detailTab === 'definition' && <DefinitionBlock sql={selected.obj.definition} />}
            </div>
          </>
        )}

        {/* === SEQUENCE === */}
        {selected.type === 'sequence' && (
          <>
            <div className="h-9 flex items-center gap-3 px-4 border-b border-border-dim bg-surface-1 shrink-0">
              <ListOrdered size={14} className="text-accent-sql" />
              <span className="text-[13px] font-medium text-text-primary font-mono">{selected.obj.schema}.{selected.obj.name}</span>
              <span className="text-[11px] text-text-muted font-mono">current: {fmt(selected.obj.currentValue)}</span>
            </div>
            <div className="flex-1 overflow-auto p-4 space-y-3">
              {[
                ['Data Type', selected.obj.dataType],
                ['Current Value', fmt(selected.obj.currentValue)],
                ['Start', fmt(selected.obj.startValue)],
                ['Increment', fmt(selected.obj.increment)],
                ['Min', fmt(selected.obj.minValue)],
                ['Max', selected.obj.maxValue > 1e15 ? '9.2e18' : fmt(selected.obj.maxValue)],
                ['Cache', fmt(selected.obj.cacheSize)],
                ['Cycle', selected.obj.cycle ? 'Yes' : 'No'],
                ['Owned By', selected.obj.ownedBy || '\u2014'],
              ].map(([label, value]) => (
                <div key={label} className="flex items-baseline gap-3">
                  <span className="text-[11px] text-text-muted uppercase tracking-wider w-28 shrink-0">{label}</span>
                  <span className="text-[13px] font-mono text-text-primary">{value}</span>
                </div>
              ))}
            </div>
          </>
        )}

        {/* === ENUM === */}
        {selected.type === 'enum' && (
          <>
            <div className="h-9 flex items-center gap-3 px-4 border-b border-border-dim bg-surface-1 shrink-0">
              <Tags size={14} className="text-accent-sql" />
              <span className="text-[13px] font-medium text-text-primary font-mono">{selected.obj.schema}.{selected.obj.name}</span>
              <span className="text-[11px] text-text-muted">{selected.obj.values.length} values</span>
            </div>
            <div className="flex-1 overflow-auto p-4">
              <div className="flex flex-wrap gap-2">
                {selected.obj.values.map((v) => (
                  <span
                    key={v}
                    className="px-3 py-1.5 text-[12px] font-mono rounded-lg bg-surface-2 border border-border-dim text-text-primary"
                  >
                    {v}
                  </span>
                ))}
              </div>
            </div>
          </>
        )}
      </div>

      {/* Modals */}
      <Modal open={showAddCol} onClose={() => setShowAddCol(false)} title="Add Column" subtitle={selected.type === 'table' ? selected.obj.name : ''}>
        <div className="space-y-3">
          <div><label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Name</label><input type="text" placeholder="column_name" value={addColName} onChange={e => setAddColName(e.target.value)} className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary placeholder:text-text-muted outline-none focus:border-border-focus transition-colors font-mono" /></div>
          <div className="grid grid-cols-2 gap-2">
            <div><label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Type</label><select value={addColType} onChange={e => setAddColType(e.target.value)} className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary outline-none focus:border-border-focus transition-colors"><option>varchar(255)</option><option>text</option><option>integer</option><option>bigint</option><option>boolean</option><option>uuid</option><option>timestamptz</option><option>numeric(10,2)</option><option>jsonb</option></select></div>
            <div><label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Default</label><input type="text" placeholder="NULL" value={addColDefault} onChange={e => setAddColDefault(e.target.value)} className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary placeholder:text-text-muted outline-none focus:border-border-focus transition-colors font-mono" /></div>
          </div>
          <div className="flex items-center gap-4"><label className="flex items-center gap-1.5 text-[13px] text-text-secondary cursor-pointer"><input type="checkbox" className="rounded" checked={addColNullable} onChange={e => setAddColNullable(e.target.checked)} /> Nullable</label></div>
          <div className="flex gap-2 justify-end pt-1">
            <button onClick={() => setShowAddCol(false)} className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-surface-3 text-text-secondary hover:text-text-primary transition-colors">Cancel</button>
            <button
              onClick={async () => {
                if (selected.type !== 'table') return
                try {
                  await api.addColumn(connectionId, selected.obj.schema, selected.obj.name, addColName, addColType, addColNullable, addColDefault)
                  toast.success('Column added')
                  setShowAddCol(false)
                  setAddColName(''); setAddColDefault('')
                  const t = await api.getTables(connectionId)
                  setTables(t || [])
                  const updated = (t || []).find((tb: SQLTable) => tb.name === selected.obj.name && tb.schema === selected.obj.schema)
                  if (updated) setSelected({ type: 'table', obj: updated })
                } catch (err: any) {
                  toast.error(err.message || 'Failed to add column')
                }
              }}
              className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-accent-sql text-white hover:bg-accent-sql/90 transition-colors"
            >
              Add column
            </button>
          </div>
        </div>
      </Modal>
      <Modal open={editCol !== null} onClose={() => setEditCol(null)} title="Edit Column" subtitle={editCol?.name}>
        {editCol && selected.type === 'table' && (
          <div className="space-y-3">
            <div>
              <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Name</label>
              <input type="text" value={editColName} onChange={e => setEditColName(e.target.value)} className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary outline-none focus:border-border-focus transition-colors font-mono" />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <div>
                <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Type</label>
                <input type="text" value={editColType} onChange={e => setEditColType(e.target.value)} className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary outline-none focus:border-border-focus transition-colors font-mono" />
              </div>
              <div>
                <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Default</label>
                <input type="text" value={editColDefault} onChange={e => setEditColDefault(e.target.value)} placeholder="empty = drop default" className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary placeholder:text-text-muted outline-none focus:border-border-focus transition-colors font-mono" />
              </div>
            </div>
            <div className="flex items-center gap-4">
              <label className="flex items-center gap-1.5 text-[13px] text-text-secondary cursor-pointer">
                <input type="checkbox" className="rounded" checked={editColNullable} onChange={e => setEditColNullable(e.target.checked)} /> Nullable
              </label>
            </div>
            <div className="flex gap-2 justify-end pt-1">
              <button onClick={() => setEditCol(null)} className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-surface-3 text-text-secondary hover:text-text-primary transition-colors">Cancel</button>
              <button
                onClick={async () => {
                  const s = selected.type === 'table' ? selected.obj.schema : ''
                  const t = selected.type === 'table' ? selected.obj.name : ''
                  try {
                    if (editColName !== editCol.name) {
                      await api.renameColumn(connectionId, s, t, editCol.name, editColName)
                    }
                    if (editColType !== editCol.type) {
                      await api.alterColumnType(connectionId, s, t, editColName, editColType)
                    }
                    if (editColNullable !== editCol.nullable) {
                      await api.setColumnNullable(connectionId, s, t, editColName, editColNullable)
                    }
                    if (editColDefault !== (editCol.defaultValue || '')) {
                      await api.setColumnDefault(connectionId, s, t, editColName, editColDefault)
                    }
                    toast.success('Column updated')
                    setEditCol(null)
                    await refreshTables()
                  } catch (err: any) {
                    toast.error(err.message || 'Failed to update column')
                  }
                }}
                className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-accent-sql text-white hover:bg-accent-sql/90 transition-colors"
              >
                Save
              </button>
            </div>
          </div>
        )}
      </Modal>
      <ConfirmDialog
        open={dropCol !== null}
        onClose={() => setDropCol(null)}
        onConfirm={async () => {
          if (!dropCol || selected.type !== 'table') return
          try {
            await api.dropColumn(connectionId, selected.obj.schema, selected.obj.name, dropCol.name)
            toast.success(`Column ${dropCol.name} dropped`)
            setDropCol(null)
            await refreshTables()
          } catch (err: any) {
            toast.error(err.message || 'Failed to drop column')
          }
        }}
        title="Drop Column"
        message={`Permanently remove "${dropCol?.name}" and all its data?`}
        confirmLabel="Drop"
      />
      <ConfirmDialog
        open={showDropTable}
        onClose={() => setShowDropTable(false)}
        onConfirm={async () => {
          if (selected.type !== 'table') return
          try {
            await api.dropTable(connectionId, selected.obj.schema, selected.obj.name)
            toast.success('Table dropped')
            const t = await api.getTables(connectionId)
            setTables(t || [])
            if (t?.length) setSelected({ type: 'table', obj: t[0] })
            else setSelected({ type: 'query' })
          } catch (err: any) {
            toast.error(err.message || 'Failed to drop table')
          }
        }}
        title="Drop Table"
        message={selected.type === 'table' ? `Drop "${selected.obj.schema}.${selected.obj.name}" with ${fmt(selected.obj.rowCount)} rows?` : ''}
        confirmLabel="Drop table"
      />
    </div>
  )
}
