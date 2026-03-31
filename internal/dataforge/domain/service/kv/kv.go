package kv

import "context"

type Entry struct {
	Key      string
	Value    string
	Type     string
	TTL      *int64
	Size     string
	Encoding string
}

type Stats struct {
	TotalKeys        int64
	MemoryUsed       string
	MemoryPeak       string
	ConnectedClients int64
	OpsPerSec        int64
	HitRate          float64
	UptimeDays       int64
	KeyspaceHits     int64
	KeyspaceMisses   int64
}

type Commands interface {
	Set(ctx context.Context, connID string, key string, value string, ttlSeconds *int64) error
	Delete(ctx context.Context, connID string, key string) error
}

type Queries interface {
	GetStats(ctx context.Context, connID string) (Stats, error)
	GetKeys(ctx context.Context, connID string, pattern string, limit int) ([]Entry, error)
	Get(ctx context.Context, connID string, key string) (Entry, error)
}
