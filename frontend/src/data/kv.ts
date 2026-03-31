import type { KVEntry, KVStats } from '../types/database'

export const kvEntries: KVEntry[] = [
  { key: 'session:a1b2c3d4', value: '{"userId":"usr_001","email":"alice@company.com","role":"admin","exp":1743350400}', type: 'string', ttl: 3600, size: '128 B', encoding: 'raw' },
  { key: 'session:b2c3d4e5', value: '{"userId":"usr_002","email":"bob@company.com","role":"user","exp":1743347200}', type: 'string', ttl: 1847, size: '124 B', encoding: 'raw' },
  { key: 'cache:products:featured', value: '[{"id":1,"name":"Widget Pro","price":29.99},{"id":2,"name":"Gadget X","price":49.99},{"id":3,"name":"Tool+","price":19.99}]', type: 'string', ttl: 300, size: '2.1 KB', encoding: 'raw' },
  { key: 'cache:user:usr_001:profile', value: '{"firstName":"Alice","lastName":"Martin","avatar":"https://avatars.example.com/alice.jpg","role":"admin"}', type: 'string', ttl: 900, size: '156 B', encoding: 'raw' },
  { key: 'ratelimit:api:192.168.1.42', value: '47', type: 'string', ttl: 60, size: '8 B', encoding: 'int' },
  { key: 'ratelimit:api:10.0.0.88', value: '12', type: 'string', ttl: 45, size: '8 B', encoding: 'int' },
  { key: 'queue:emails:pending', value: '["msg_001","msg_002","msg_003","msg_004","msg_005"]', type: 'list', ttl: null, size: '320 B', encoding: 'listpack' },
  { key: 'queue:webhooks:retry', value: '["whk_019","whk_022"]', type: 'list', ttl: null, size: '96 B', encoding: 'listpack' },
  { key: 'set:online_users', value: '{"usr_001","usr_002","usr_005","usr_007","usr_012","usr_019","usr_023"}', type: 'set', ttl: null, size: '256 B', encoding: 'listpack' },
  { key: 'leaderboard:weekly', value: '[["usr_007",2847],["usr_001",2341],["usr_019",1982],["usr_005",1876],["usr_023",1654]]', type: 'zset', ttl: null, size: '384 B', encoding: 'skiplist' },
  { key: 'config:feature_flags', value: '{"dark_mode":true,"beta_api":false,"new_checkout":true,"ai_search":true}', type: 'hash', ttl: null, size: '192 B', encoding: 'listpack' },
  { key: 'config:maintenance', value: '{"enabled":false,"message":"","scheduled_at":null}', type: 'hash', ttl: null, size: '96 B', encoding: 'listpack' },
  { key: 'metrics:api:latency:p99', value: '142', type: 'string', ttl: 60, size: '8 B', encoding: 'int' },
  { key: 'metrics:api:latency:p50', value: '23', type: 'string', ttl: 60, size: '8 B', encoding: 'int' },
  { key: 'lock:deploy:production', value: '{"holder":"ci-runner-03","acquired_at":"2026-03-29T08:30:00Z"}', type: 'string', ttl: 600, size: '96 B', encoding: 'raw' },
  { key: 'stream:events', value: '1743235200000-0: {"type":"order.created","payload":{"orderId":"ord_847"}}', type: 'stream', ttl: null, size: '12.4 KB', encoding: 'stream' },
]

export const kvStats: KVStats = {
  totalKeys: 284_192,
  memoryUsed: '1.24 GB',
  memoryPeak: '1.87 GB',
  connectedClients: 42,
  opsPerSec: 12_847,
  hitRate: 94.7,
  uptimeDays: 127,
  keyspaceHits: 948_291_042,
  keyspaceMisses: 52_847_291,
}
