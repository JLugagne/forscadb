package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/kv"
)

type Driver struct {
	client *goredis.Client
}

func NewRedisDriver(client *goredis.Client) *Driver {
	return &Driver{client: client}
}

func (d *Driver) Ping(ctx context.Context) error {
	return d.client.Ping(ctx).Err()
}

func (d *Driver) Close() error {
	return d.client.Close()
}

func (d *Driver) GetStats(ctx context.Context) (kv.Stats, error) {
	info, err := d.client.Info(ctx).Result()
	if err != nil {
		return kv.Stats{}, fmt.Errorf("redis: GetStats: %w", err)
	}

	sections := parseINFO(info)

	var stats kv.Stats

	if mem, ok := sections["memory"]; ok {
		if v, ok := mem["used_memory"]; ok {
			stats.MemoryUsed = formatMemBytes(parseINTField(v))
		}
		if v, ok := mem["used_memory_peak"]; ok {
			stats.MemoryPeak = formatMemBytes(parseINTField(v))
		}
	}

	if clients, ok := sections["clients"]; ok {
		if v, ok := clients["connected_clients"]; ok {
			stats.ConnectedClients = parseINTField(v)
		}
	}

	if server, ok := sections["server"]; ok {
		if v, ok := server["uptime_in_days"]; ok {
			stats.UptimeDays = parseINTField(v)
		}
	}

	if statsSection, ok := sections["stats"]; ok {
		if v, ok := statsSection["instantaneous_ops_per_sec"]; ok {
			stats.OpsPerSec = parseINTField(v)
		}
		if v, ok := statsSection["keyspace_hits"]; ok {
			stats.KeyspaceHits = parseINTField(v)
		}
		if v, ok := statsSection["keyspace_misses"]; ok {
			stats.KeyspaceMisses = parseINTField(v)
		}
	}

	if stats.KeyspaceHits+stats.KeyspaceMisses > 0 {
		stats.HitRate = float64(stats.KeyspaceHits) / float64(stats.KeyspaceHits+stats.KeyspaceMisses)
	}

	if ks, ok := sections["keyspace"]; ok {
		var total int64
		for _, v := range ks {
			parts := strings.Split(v, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(p, "keys=") {
					n, _ := strconv.ParseInt(strings.TrimPrefix(p, "keys="), 10, 64)
					total += n
				}
			}
		}
		stats.TotalKeys = total
	}

	return stats, nil
}

func (d *Driver) GetKeys(ctx context.Context, pattern string, limit int) ([]kv.Entry, error) {
	if pattern == "" {
		pattern = "*"
	}

	var keys []string
	var cursor uint64
	for {
		batch, next, err := d.client.Scan(ctx, cursor, pattern, int64(limit)).Result()
		if err != nil {
			return nil, fmt.Errorf("redis: GetKeys: scan: %w", err)
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 || len(keys) >= limit {
			break
		}
	}
	if len(keys) > limit {
		keys = keys[:limit]
	}

	entries := make([]kv.Entry, 0, len(keys))
	for _, key := range keys {
		entry, err := d.Get(ctx, key)
		if err != nil {
			entry = kv.Entry{Key: key, Type: "unknown"}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (d *Driver) Get(ctx context.Context, key string) (kv.Entry, error) {
	keyType, err := d.client.Type(ctx, key).Result()
	if err != nil {
		return kv.Entry{}, fmt.Errorf("redis: Get: type: %w", err)
	}
	if keyType == "none" {
		return kv.Entry{}, fmt.Errorf("redis: Get: key %q not found", key)
	}

	var value string
	switch keyType {
	case "string":
		value, err = d.client.Get(ctx, key).Result()
		if err != nil {
			return kv.Entry{}, fmt.Errorf("redis: Get: get: %w", err)
		}
	case "list":
		vals, e := d.client.LRange(ctx, key, 0, 99).Result()
		if e != nil {
			return kv.Entry{}, fmt.Errorf("redis: Get: lrange: %w", e)
		}
		value = strings.Join(vals, "\n")
	case "set":
		vals, e := d.client.SMembers(ctx, key).Result()
		if e != nil {
			return kv.Entry{}, fmt.Errorf("redis: Get: smembers: %w", e)
		}
		value = strings.Join(vals, "\n")
	case "zset":
		vals, e := d.client.ZRangeWithScores(ctx, key, 0, 99).Result()
		if e != nil {
			return kv.Entry{}, fmt.Errorf("redis: Get: zrange: %w", e)
		}
		parts := make([]string, 0, len(vals))
		for _, z := range vals {
			parts = append(parts, fmt.Sprintf("%v:%v", z.Member, z.Score))
		}
		value = strings.Join(parts, "\n")
	case "hash":
		fields, e := d.client.HGetAll(ctx, key).Result()
		if e != nil {
			return kv.Entry{}, fmt.Errorf("redis: Get: hgetall: %w", e)
		}
		parts := make([]string, 0, len(fields))
		for k, v := range fields {
			parts = append(parts, fmt.Sprintf("%s: %s", k, v))
		}
		value = strings.Join(parts, "\n")
	case "stream":
		msgs, e := d.client.XRange(ctx, key, "-", "+").Result()
		if e != nil {
			return kv.Entry{}, fmt.Errorf("redis: Get: xrange: %w", e)
		}
		parts := make([]string, 0, len(msgs))
		for _, m := range msgs {
			parts = append(parts, fmt.Sprintf("%s: %v", m.ID, m.Values))
		}
		value = strings.Join(parts, "\n")
	default:
		value = ""
	}

	ttlDur, err := d.client.TTL(ctx, key).Result()
	if err != nil {
		return kv.Entry{}, fmt.Errorf("redis: Get: ttl: %w", err)
	}

	var ttl *int64
	if ttlDur >= 0 {
		secs := int64(ttlDur / time.Second)
		ttl = &secs
	}

	memUsage, err := d.client.MemoryUsage(ctx, key).Result()
	var size string
	if err == nil {
		size = formatMemBytes(memUsage)
	}

	encoding := d.getEncoding(ctx, key)

	return kv.Entry{
		Key:      key,
		Value:    value,
		Type:     keyType,
		TTL:      ttl,
		Size:     size,
		Encoding: encoding,
	}, nil
}

func (d *Driver) getEncoding(ctx context.Context, key string) string {
	enc, err := d.client.ObjectEncoding(ctx, key).Result()
	if err != nil {
		return ""
	}
	return enc
}

func (d *Driver) Set(ctx context.Context, key string, value string, ttlSeconds *int64) error {
	var dur time.Duration
	if ttlSeconds != nil && *ttlSeconds > 0 {
		dur = time.Duration(*ttlSeconds) * time.Second
	}
	if err := d.client.Set(ctx, key, value, dur).Err(); err != nil {
		return fmt.Errorf("redis: Set: %w", err)
	}
	return nil
}

func (d *Driver) Delete(ctx context.Context, key string) error {
	if err := d.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis: Delete: %w", err)
	}
	return nil
}

func parseINFO(info string) map[string]map[string]string {
	sections := make(map[string]map[string]string)
	var currentSection string
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "#") {
			currentSection = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "#")))
			sections[currentSection] = make(map[string]string)
			continue
		}
		if currentSection == "" || line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			sections[currentSection][strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return sections
}

func parseINTField(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func formatMemBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.2f GB", float64(b)/gb)
	case b >= mb:
		return fmt.Sprintf("%.2f MB", float64(b)/mb)
	case b >= kb:
		return fmt.Sprintf("%.2f KB", float64(b)/kb)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
