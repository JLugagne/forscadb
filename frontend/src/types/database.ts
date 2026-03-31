export type DatabaseEngine = 'postgresql' | 'mysql' | 'mariadb' | 'sqlite' | 'cockroachdb' | 'sqlserver' | 'mongodb' | 'documentdb' | 'redis' | 'valkey' | 'keydb' | 'dragonfly' | 'memcached'
export type DatabaseCategory = 'sql' | 'nosql' | 'kv'

export interface Connection {
  id: string
  name: string
  engine: DatabaseEngine
  category: DatabaseCategory
  host: string
  port: number
  user: string
  database?: string
  sslMode?: string
  status: 'connected' | 'disconnected' | 'error'
  color: string
  lastAccess: string
}

// SQL types
export interface SQLColumn {
  name: string
  type: string
  nullable: boolean
  primaryKey: boolean
  defaultValue: string | null
  foreignKey?: { table: string; column: string }
}

export interface SQLTable {
  name: string
  schema: string
  columns: SQLColumn[]
  rowCount: number
  size: string
  indexes: SQLIndex[]
}

export interface SQLIndex {
  name: string
  columns: string[]
  unique: boolean
  type: string
}

export interface SQLView {
  name: string
  schema: string
  definition: string
  columns: SQLColumn[]
  materialized: boolean
}

export interface SQLFunction {
  name: string
  schema: string
  language: string
  returnType: string
  args: { name: string; type: string; mode: 'IN' | 'OUT' | 'INOUT' | 'VARIADIC' }[]
  volatility: 'VOLATILE' | 'STABLE' | 'IMMUTABLE'
  definition: string
}

export interface SQLTrigger {
  name: string
  schema: string
  table: string
  event: string
  timing: 'BEFORE' | 'AFTER' | 'INSTEAD OF'
  forEach: 'ROW' | 'STATEMENT'
  function: string
  enabled: boolean
  definition: string
}

export interface SQLSequence {
  name: string
  schema: string
  dataType: string
  startValue: number
  increment: number
  minValue: number
  maxValue: number
  currentValue: number
  cacheSize: number
  cycle: boolean
  ownedBy: string | null
}

export interface SQLEnum {
  name: string
  schema: string
  values: string[]
}

export type SQLObjectType = 'table' | 'view' | 'function' | 'trigger' | 'sequence' | 'enum'

export interface SQLQueryResult {
  columns: string[]
  rows: Record<string, unknown>[]
  rowCount: number
  executionTime: number
  affectedRows?: number
}

// NoSQL types
export interface NoSQLCollection {
  name: string
  documentCount: number
  avgDocSize: string
  totalSize: string
  indexes: NoSQLIndex[]
}

export interface NoSQLIndex {
  name: string
  keys: Record<string, number>
  unique: boolean
}

export interface NoSQLDocument {
  _id: string
  [key: string]: unknown
}

// KV types
export interface KVEntry {
  key: string
  value: string
  type: 'string' | 'list' | 'set' | 'zset' | 'hash' | 'stream'
  ttl: number | null
  size: string
  encoding: string
}

export interface KVStats {
  totalKeys: number
  memoryUsed: string
  memoryPeak: string
  connectedClients: number
  opsPerSec: number
  hitRate: number
  uptimeDays: number
  keyspaceHits: number
  keyspaceMisses: number
}

export interface QueryHistoryEntry {
  id: string
  query: string
  executedAt: string
  duration: number
  rowCount: number
  status: 'success' | 'error'
  error?: string
}
