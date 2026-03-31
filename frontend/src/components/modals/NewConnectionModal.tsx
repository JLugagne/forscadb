import { useState, useRef, useEffect } from 'react'
import { Modal } from '../shared/Modal'
import { Database, FileJson, HardDrive, Eye, EyeOff, Zap, ChevronDown, Check, Shield, ShieldCheck, Link } from 'lucide-react'
import type { DatabaseEngine } from '../../types/database'
import * as api from '../../api'
import toast from 'react-hot-toast'

interface NewConnectionModalProps {
  open: boolean
  onClose: () => void
  onConnectionsChanged: () => void
}

interface EngineInfo {
  id: DatabaseEngine
  label: string
  icon: typeof Database
  cat: 'SQL' | 'NoSQL' | 'Key-Value'
  category: 'sql' | 'nosql' | 'kv'
  port: number
  color: string
  sslModes: string[]
  hideDatabase?: boolean
  uriScheme: string
}

const engines: EngineInfo[] = [
  { id: 'postgresql',  label: 'PostgreSQL',  icon: Database,  cat: 'SQL',       category: 'sql',   port: 5432,  color: '#336791', sslModes: ['disable', 'require', 'verify-ca', 'verify-full'], uriScheme: 'postgresql' },
  { id: 'mysql',       label: 'MySQL',       icon: Database,  cat: 'SQL',       category: 'sql',   port: 3306,  color: '#00758f', sslModes: ['disable', 'true', 'skip-verify'], uriScheme: 'mysql' },
  { id: 'mariadb',     label: 'MariaDB',     icon: Database,  cat: 'SQL',       category: 'sql',   port: 3306,  color: '#003545', sslModes: ['disable', 'true', 'skip-verify'], uriScheme: 'mariadb' },
  { id: 'cockroachdb', label: 'CockroachDB', icon: Database,  cat: 'SQL',       category: 'sql',   port: 26257, color: '#6933ff', sslModes: ['disable', 'require', 'verify-ca', 'verify-full'], uriScheme: 'postgresql' },
  { id: 'sqlite',      label: 'SQLite',      icon: Database,  cat: 'SQL',       category: 'sql',   port: 0,     color: '#003b57', sslModes: ['disable'], uriScheme: 'sqlite' },
  { id: 'sqlserver',   label: 'SQL Server',  icon: Database,  cat: 'SQL',       category: 'sql',   port: 1433,  color: '#cc2927', sslModes: ['disable', 'true'], uriScheme: 'sqlserver' },
  { id: 'mongodb',     label: 'MongoDB',     icon: FileJson,  cat: 'NoSQL',     category: 'nosql', port: 27017, color: '#00ed64', sslModes: ['disable', 'require'], uriScheme: 'mongodb' },
  { id: 'documentdb',  label: 'DocumentDB',  icon: FileJson,  cat: 'NoSQL',     category: 'nosql', port: 27017, color: '#527fff', sslModes: ['disable', 'require'], uriScheme: 'mongodb' },
  { id: 'redis',       label: 'Redis',       icon: HardDrive, cat: 'Key-Value', category: 'kv',    port: 6379,  color: '#d82c20', sslModes: ['disable', 'true'], hideDatabase: true, uriScheme: 'redis' },
  { id: 'valkey',      label: 'Valkey',      icon: HardDrive, cat: 'Key-Value', category: 'kv',    port: 6379,  color: '#5b21b6', sslModes: ['disable', 'true'], hideDatabase: true, uriScheme: 'redis' },
  { id: 'keydb',       label: 'KeyDB',       icon: HardDrive, cat: 'Key-Value', category: 'kv',    port: 6379,  color: '#ff6600', sslModes: ['disable', 'true'], hideDatabase: true, uriScheme: 'redis' },
  { id: 'dragonfly',   label: 'Dragonfly',   icon: HardDrive, cat: 'Key-Value', category: 'kv',    port: 6379,  color: '#22c55e', sslModes: ['disable', 'true'], hideDatabase: true, uriScheme: 'redis' },
  { id: 'memcached',   label: 'Memcached',   icon: HardDrive, cat: 'Key-Value', category: 'kv',    port: 11211, color: '#768a2e', sslModes: ['disable'], hideDatabase: true, uriScheme: 'memcached' },
]

const groups = [
  { label: 'SQL', items: engines.filter(e => e.cat === 'SQL') },
  { label: 'NoSQL', items: engines.filter(e => e.cat === 'NoSQL') },
  { label: 'Key-Value', items: engines.filter(e => e.cat === 'Key-Value') },
]

const sslLabels: Record<string, string> = {
  'disable': 'Disabled', 'require': 'Required', 'verify-ca': 'Verify CA',
  'verify-full': 'Verify Full', 'true': 'Enabled', 'skip-verify': 'Skip Verify',
}

// --- Combobox ---
function Dropdown({ value, options, onChange, renderTrigger, renderOption }: {
  value: string
  options: { id: string }[]
  onChange: (id: string) => void
  renderTrigger: () => React.ReactNode
  renderOption: (opt: any, active: boolean) => React.ReactNode
  groups?: { label: string; items: any[] }[]
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  return (
    <div ref={ref} className="relative">
      <button type="button" onClick={() => setOpen(!open)} className="w-full flex items-center gap-2 px-3 py-1.5 bg-surface-2 border border-border-base rounded-lg text-left hover:border-border-bright transition-colors">
        {renderTrigger()}
        <ChevronDown size={13} className={`text-text-muted transition-transform duration-150 ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && (
        <div className="absolute z-50 mt-1 w-full bg-surface-2 border border-border-base rounded-lg shadow-2xl shadow-black/50 py-1 animate-scale-in max-h-[280px] overflow-y-auto">
          {options.map(opt => (
            <button key={opt.id} onClick={() => { onChange(opt.id); setOpen(false) }}
              className={`w-full flex items-center gap-2 px-3 py-1.5 text-left transition-colors ${opt.id === value ? 'bg-surface-hover' : 'hover:bg-surface-hover/60'}`}>
              {renderOption(opt, opt.id === value)}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

// --- Grouped Engine Combobox ---
function EngineCombobox({ value, onChange }: { value: DatabaseEngine; onChange: (id: DatabaseEngine) => void }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const selected = engines.find(e => e.id === value)!

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  return (
    <div ref={ref} className="relative">
      <button type="button" onClick={() => setOpen(!open)}
        className="w-full flex items-center gap-2.5 px-3 py-2 bg-surface-2 border border-border-base rounded-lg text-left hover:border-border-bright transition-colors">
        <div className="w-5 h-5 rounded-md flex items-center justify-center" style={{ backgroundColor: selected.color + '20' }}>
          <selected.icon size={12} style={{ color: selected.color }} />
        </div>
        <span className="text-[13px] font-medium text-text-primary flex-1">{selected.label}</span>
        <span className="text-[10px] text-text-muted uppercase tracking-wider mr-1">{selected.cat}</span>
        <ChevronDown size={13} className={`text-text-muted transition-transform duration-150 ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && (
        <div className="absolute z-50 mt-1.5 w-full bg-surface-2 border border-border-base rounded-lg shadow-2xl shadow-black/50 py-1 animate-scale-in max-h-[320px] overflow-y-auto">
          {groups.map((group, gi) => (
            <div key={group.label}>
              {gi > 0 && <div className="my-1 border-t border-border-dim" />}
              <div className="px-3 py-1"><span className="text-[10px] font-semibold text-text-muted uppercase tracking-widest">{group.label}</span></div>
              {group.items.map(e => {
                const active = e.id === value
                return (
                  <button key={e.id} onClick={() => { onChange(e.id); setOpen(false) }}
                    className={`w-full flex items-center gap-2.5 px-3 py-1.5 text-left transition-colors ${active ? 'bg-surface-hover' : 'hover:bg-surface-hover/60'}`}>
                    <div className="w-5 h-5 rounded-md flex items-center justify-center" style={{ backgroundColor: e.color + '20' }}>
                      <e.icon size={12} style={{ color: e.color }} />
                    </div>
                    <span className={`text-[13px] flex-1 ${active ? 'text-text-primary font-medium' : 'text-text-secondary'}`}>{e.label}</span>
                    {active && <Check size={13} className="text-accent-sql" />}
                  </button>
                )
              })}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// --- URI helpers ---
const schemeMap: Record<string, DatabaseEngine> = {
  postgresql: 'postgresql', postgres: 'postgresql', cockroachdb: 'cockroachdb',
  mysql: 'mysql', mariadb: 'mariadb',
  mongodb: 'mongodb', 'mongodb+srv': 'mongodb',
  redis: 'redis', rediss: 'redis',
}

function buildPreviewUri(engine: DatabaseEngine, form: { host: string; port: string; user: string; database: string }, sslMode: string) {
  const sel = engines.find(e => e.id === engine)!
  const host = form.host || 'localhost'
  const port = form.port || String(sel.port)
  const user = form.user || '<user>'
  const db = form.database || '<db>'
  if (sel.category === 'kv') return `${sel.uriScheme}://${user}@${host}:${port}`
  let uri = `${sel.uriScheme}://${user}@${host}:${port}/${db}`
  if (sslMode !== 'disable') uri += `?sslmode=${sslMode}`
  return uri
}

// --- Main Modal ---
export function NewConnectionModal({ open, onClose, onConnectionsChanged }: NewConnectionModalProps) {
  const [engine, setEngine] = useState<DatabaseEngine>('postgresql')
  const [showPw, setShowPw] = useState(false)
  const [form, setForm] = useState({ name: '', host: 'localhost', port: '5432', user: '', password: '', database: '' })
  const [sslMode, setSSLMode] = useState('disable')
  const [tab, setTab] = useState<'basic' | 'advanced'>('basic')
  const [inputMode, setInputMode] = useState<'detailed' | 'uri'>('detailed')
  const [uri, setUri] = useState('')
  const [testing, setTesting] = useState(false)
  const [saving, setSaving] = useState(false)

  const sel = engines.find(e => e.id === engine)!

  const handleEngineChange = (id: DatabaseEngine) => {
    const eng = engines.find(e => e.id === id)!
    setEngine(id)
    setForm(f => ({ ...f, port: eng.port ? String(eng.port) : '' }))
    setSSLMode('disable')
  }

  const parseUri = (raw: string) => {
    try {
      const parsed = new URL(raw)
      const proto = parsed.protocol.replace(':', '')
      const detected = schemeMap[proto]
      if (detected) handleEngineChange(detected)
      setForm({
        name: form.name,
        host: parsed.hostname || 'localhost',
        port: parsed.port || String(engines.find(e => e.id === (detected || engine))?.port || ''),
        user: decodeURIComponent(parsed.username || ''),
        password: decodeURIComponent(parsed.password || ''),
        database: parsed.pathname.replace(/^\//, '') || '',
      })
      const ssl = parsed.searchParams.get('sslmode') || parsed.searchParams.get('tls') || parsed.searchParams.get('ssl')
      if (ssl) setSSLMode(ssl)
    } catch {
      // Not a valid URI yet, that's fine while typing
    }
  }

  const reset = () => {
    setEngine('postgresql')
    setForm({ name: '', host: 'localhost', port: '5432', user: '', password: '', database: '' })
    setSSLMode('disable')
    setShowPw(false)
    setTab('basic')
    setInputMode('detailed')
    setUri('')
    onClose()
  }

  const buildInput = () => ({
    name: form.name || `My ${sel.label}`,
    engine,
    category: sel.category,
    host: form.host,
    port: parseInt(form.port) || sel.port,
    user: form.user,
    password: form.password,
    database: form.database,
    sslMode,
    color: sel.color,
  })

  const handleTest = async () => {
    setTesting(true)
    try {
      await api.testConnection(buildInput())
      toast.success('Connection OK!')
    } catch (err: any) {
      toast.error(err.message || 'Connection failed')
    } finally {
      setTesting(false)
    }
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      const conn = await api.createConnection(buildInput())
      try { await api.connectToDatabase(conn.id) } catch { /* OK */ }
      onConnectionsChanged()
      reset()
      toast.success('Connection created')
    } catch (err: any) {
      toast.error(err.message || 'Failed to create connection')
    } finally {
      setSaving(false)
    }
  }

  const showDatabase = !sel.hideDatabase

  return (
    <Modal open={open} onClose={reset} title="New connection" width="max-w-md">
      <div className="space-y-3">
        {/* Engine */}
        <div>
          <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Engine</label>
          <EngineCombobox value={engine} onChange={handleEngineChange} />
        </div>

        {/* Basic / Advanced tabs */}
        <div className="flex bg-surface-2 rounded-lg p-0.5 border border-border-dim">
          <button onClick={() => setTab('basic')} className={`flex-1 px-3 py-1 text-[12px] font-medium rounded-md transition-colors ${tab === 'basic' ? 'bg-surface-3 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}>
            Basic
          </button>
          <button onClick={() => setTab('advanced')} className={`flex-1 px-3 py-1 text-[12px] font-medium rounded-md transition-colors ${tab === 'advanced' ? 'bg-surface-3 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}>
            Advanced
          </button>
        </div>

        {tab === 'basic' && (
          <>
            {/* Detailed / URI toggle */}
            <div className="flex items-center gap-2">
              <button
                onClick={() => setInputMode('detailed')}
                className={`flex items-center gap-1.5 px-2.5 py-1 text-[12px] font-medium rounded-md transition-colors ${inputMode === 'detailed' ? 'bg-surface-3 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}
              >
                Detailed
              </button>
              <button
                onClick={() => {
                  setInputMode('uri')
                  if (!uri) setUri(buildPreviewUri(engine, form, sslMode))
                }}
                className={`flex items-center gap-1.5 px-2.5 py-1 text-[12px] font-medium rounded-md transition-colors ${inputMode === 'uri' ? 'bg-surface-3 text-text-primary' : 'text-text-muted hover:text-text-secondary'}`}
              >
                <Link size={11} /> URI
              </button>
            </div>

            {inputMode === 'detailed' ? (
              <>
                {/* Name */}
                <div>
                  <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Name</label>
                  <input type="text" placeholder={`My ${sel.label}`} value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                    className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary placeholder:text-text-muted outline-none focus:border-border-focus transition-colors" />
                </div>

                {/* Host + Port */}
                <div className="grid grid-cols-3 gap-2">
                  <div className="col-span-2">
                    <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Host</label>
                    <input type="text" value={form.host} onChange={e => setForm(f => ({ ...f, host: e.target.value }))}
                      className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary outline-none focus:border-border-focus transition-colors" />
                  </div>
                  <div>
                    <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Port</label>
                    <input type="text" value={form.port} onChange={e => setForm(f => ({ ...f, port: e.target.value }))}
                      className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary outline-none focus:border-border-focus transition-colors font-mono" />
                  </div>
                </div>

                {/* User + Password */}
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">User</label>
                    <input type="text" value={form.user} onChange={e => setForm(f => ({ ...f, user: e.target.value }))}
                      className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary outline-none focus:border-border-focus transition-colors" />
                  </div>
                  <div>
                    <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Password</label>
                    <div className="relative">
                      <input type={showPw ? 'text' : 'password'} value={form.password} onChange={e => setForm(f => ({ ...f, password: e.target.value }))}
                        className="w-full px-3 py-1.5 pr-8 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary outline-none focus:border-border-focus transition-colors" />
                      <button type="button" onClick={() => setShowPw(!showPw)} className="absolute right-2 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors">
                        {showPw ? <EyeOff size={13} /> : <Eye size={13} />}
                      </button>
                    </div>
                  </div>
                </div>

                {/* Database */}
                {showDatabase && (
                  <div>
                    <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Database</label>
                    <input type="text" value={form.database} onChange={e => setForm(f => ({ ...f, database: e.target.value }))}
                      className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary outline-none focus:border-border-focus transition-colors" />
                  </div>
                )}
              </>
            ) : (
              <>
                {/* Name (still needed even in URI mode) */}
                <div>
                  <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Name</label>
                  <input type="text" placeholder={`My ${sel.label}`} value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                    className="w-full px-3 py-1.5 text-[13px] bg-surface-2 border border-border-base rounded-lg text-text-primary placeholder:text-text-muted outline-none focus:border-border-focus transition-colors" />
                </div>

                {/* URI input */}
                <div>
                  <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Connection URI</label>
                  <textarea
                    value={uri}
                    onChange={e => { setUri(e.target.value); parseUri(e.target.value) }}
                    placeholder={`${sel.uriScheme}://user:password@localhost:${sel.port}${showDatabase ? '/mydb' : ''}`}
                    rows={2}
                    className="w-full px-3 py-2 text-[12px] font-mono bg-surface-2 border border-border-base rounded-lg text-text-primary placeholder:text-text-muted outline-none focus:border-border-focus transition-colors resize-none"
                    spellCheck={false}
                  />
                  <p className="mt-1 text-[11px] text-text-muted">
                    Engine, host, port, credentials, and database will be parsed automatically.
                  </p>
                </div>
              </>
            )}
          </>
        )}

        {tab === 'advanced' && (
          <>
            {/* SSL Mode */}
            <div>
              <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider flex items-center gap-1">
                {sslMode === 'disable' ? <Shield size={10} /> : <ShieldCheck size={10} className="text-emerald-400" />}
                SSL Mode
              </label>
              <Dropdown
                value={sslMode}
                options={sel.sslModes.map(m => ({ id: m }))}
                onChange={setSSLMode}
                renderTrigger={() => <><span className="text-[13px] text-text-primary flex-1">{sslLabels[sslMode] || sslMode}</span></>}
                renderOption={(opt: { id: string }, active: boolean) => (
                  <>
                    <span className={`text-[13px] flex-1 ${active ? 'text-text-primary font-medium' : 'text-text-secondary'}`}>{sslLabels[opt.id] || opt.id}</span>
                    {active && <Check size={13} className="text-accent-sql" />}
                  </>
                )}
              />
              <p className="mt-1 text-[11px] text-text-muted">
                {sslMode === 'disable' && 'No encryption. Only use for local development.'}
                {sslMode === 'require' && 'Encrypted connection required. Server certificate not verified.'}
                {sslMode === 'verify-ca' && 'Encrypted connection with CA certificate verification.'}
                {sslMode === 'verify-full' && 'Full verification: CA + hostname match. Most secure.'}
                {sslMode === 'true' && 'SSL/TLS encryption enabled.'}
                {sslMode === 'skip-verify' && 'Encrypted but server certificate not verified.'}
              </p>
            </div>

            {/* Connection preview */}
            <div>
              <label className="block text-[11px] font-medium text-text-muted mb-1 uppercase tracking-wider">Connection preview</label>
              <div className="px-3 py-2 bg-surface-0 border border-border-dim rounded-lg font-mono text-[11px] text-text-muted break-all leading-relaxed">
                {buildPreviewUri(engine, form, sslMode)}
              </div>
            </div>
          </>
        )}

        {/* Actions */}
        <div className="flex gap-2 pt-1">
          <button onClick={reset} className="px-3 py-1.5 text-[13px] font-medium rounded-lg bg-surface-3 text-text-secondary hover:text-text-primary transition-colors">
            Cancel
          </button>
          <button onClick={handleTest} disabled={testing} className="flex items-center gap-1.5 px-3 py-1.5 text-[13px] font-medium rounded-lg border border-border-base text-text-secondary hover:text-text-primary hover:bg-surface-2 transition-colors disabled:opacity-50">
            <Zap size={12} /> {testing ? 'Testing...' : 'Test'}
          </button>
          <button onClick={handleSave} disabled={saving} className="flex-1 px-3 py-1.5 text-[13px] font-medium rounded-lg text-white transition-colors disabled:opacity-50" style={{ backgroundColor: sel.color }}>
            {saving ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>
    </Modal>
  )
}
