import type { NoSQLCollection, NoSQLDocument } from '../types/database'

export const nosqlCollections: NoSQLCollection[] = [
  {
    name: 'users',
    documentCount: 1_284_301,
    avgDocSize: '2.1 KB',
    totalSize: '2.6 GB',
    indexes: [
      { name: '_id_', keys: { _id: 1 }, unique: true },
      { name: 'email_1', keys: { email: 1 }, unique: true },
      { name: 'status_1_createdAt_-1', keys: { status: 1, createdAt: -1 }, unique: false },
    ],
  },
  {
    name: 'events',
    documentCount: 48_291_042,
    avgDocSize: '856 B',
    totalSize: '38.7 GB',
    indexes: [
      { name: '_id_', keys: { _id: 1 }, unique: true },
      { name: 'userId_1_timestamp_-1', keys: { userId: 1, timestamp: -1 }, unique: false },
      { name: 'type_1', keys: { type: 1 }, unique: false },
      { name: 'timestamp_1', keys: { timestamp: 1 }, unique: false },
    ],
  },
  {
    name: 'products',
    documentCount: 34_891,
    avgDocSize: '4.7 KB',
    totalSize: '156.2 MB',
    indexes: [
      { name: '_id_', keys: { _id: 1 }, unique: true },
      { name: 'sku_1', keys: { sku: 1 }, unique: true },
      { name: 'categories_1', keys: { categories: 1 }, unique: false },
      { name: 'price_1', keys: { price: 1 }, unique: false },
    ],
  },
  {
    name: 'sessions',
    documentCount: 892_104,
    avgDocSize: '512 B',
    totalSize: '435.1 MB',
    indexes: [
      { name: '_id_', keys: { _id: 1 }, unique: true },
      { name: 'userId_1', keys: { userId: 1 }, unique: false },
      { name: 'expiresAt_1', keys: { expiresAt: 1 }, unique: false },
    ],
  },
  {
    name: 'audit_logs',
    documentCount: 127_482_019,
    avgDocSize: '1.3 KB',
    totalSize: '154.8 GB',
    indexes: [
      { name: '_id_', keys: { _id: 1 }, unique: true },
      { name: 'actor_1_timestamp_-1', keys: { actor: 1, timestamp: -1 }, unique: false },
      { name: 'resource_1_action_1', keys: { resource: 1, action: 1 }, unique: false },
    ],
  },
  {
    name: 'notifications',
    documentCount: 5_291_847,
    avgDocSize: '768 B',
    totalSize: '3.8 GB',
    indexes: [
      { name: '_id_', keys: { _id: 1 }, unique: true },
      { name: 'userId_1_read_1', keys: { userId: 1, read: -1 }, unique: false },
      { name: 'createdAt_1', keys: { createdAt: 1 }, unique: false },
    ],
  },
]

export const sampleDocuments: NoSQLDocument[] = [
  {
    _id: '65f2a1b3c4d5e6f7a8b9c0d1',
    email: 'alice@company.com',
    username: 'alice_dev',
    profile: {
      firstName: 'Alice',
      lastName: 'Martin',
      avatar: 'https://avatars.example.com/alice.jpg',
      bio: 'Full-stack developer',
      timezone: 'Europe/Paris',
    },
    preferences: {
      theme: 'dark',
      language: 'fr',
      notifications: { email: true, push: true, sms: false },
    },
    roles: ['admin', 'developer'],
    status: 'active',
    metadata: {
      loginCount: 1842,
      lastIp: '192.168.1.42',
      devices: ['macbook-pro', 'iphone-15'],
    },
    createdAt: { $date: '2024-01-15T10:30:00.000Z' },
    updatedAt: { $date: '2026-03-29T08:12:00.000Z' },
  },
  {
    _id: '65f2a1b3c4d5e6f7a8b9c0d2',
    email: 'bob@company.com',
    username: 'bob_ops',
    profile: {
      firstName: 'Bob',
      lastName: 'Chen',
      avatar: null,
      bio: 'DevOps engineer',
      timezone: 'America/New_York',
    },
    preferences: {
      theme: 'light',
      language: 'en',
      notifications: { email: true, push: false, sms: false },
    },
    roles: ['user', 'ops'],
    status: 'active',
    metadata: {
      loginCount: 923,
      lastIp: '10.0.0.88',
      devices: ['thinkpad-x1'],
    },
    createdAt: { $date: '2024-02-20T14:15:00.000Z' },
    updatedAt: { $date: '2026-03-28T22:45:00.000Z' },
  },
  {
    _id: '65f2a1b3c4d5e6f7a8b9c0d3',
    email: 'carol@company.com',
    username: 'carol_pm',
    profile: {
      firstName: 'Carol',
      lastName: 'Dubois',
      avatar: 'https://avatars.example.com/carol.jpg',
      bio: 'Product manager',
      timezone: 'Europe/London',
    },
    preferences: {
      theme: 'dark',
      language: 'en',
      notifications: { email: true, push: true, sms: true },
    },
    roles: ['user', 'manager'],
    status: 'active',
    metadata: {
      loginCount: 2104,
      lastIp: '172.16.0.15',
      devices: ['macbook-air', 'ipad-pro'],
    },
    createdAt: { $date: '2024-03-10T09:00:00.000Z' },
    updatedAt: { $date: '2026-03-29T07:30:00.000Z' },
  },
]
