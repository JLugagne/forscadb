//go:build e2e

package e2eapi_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/JLugagne/forscadb/internal/dataforge/app"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound/memory"
	"github.com/JLugagne/forscadb/internal/domain"
)

func newKVServiceWithConn(t *testing.T) (*app.KVService, *app.ConnectionManager, string) {
	t.Helper()
	ctx := context.Background()

	store := memory.NewConnStore()
	factory := outbound.NewFactory()
	manager := app.NewConnectionManager(store, factory)
	kvSvc := app.NewKVService(manager)

	conn := connection.Connection{
		Name:     "test-redis",
		Engine:   domain.EngineRedis,
		Category: domain.CategoryKV,
		Host:     redisHost,
		Port:     redisPort,
	}

	created, err := manager.Create(ctx, conn)
	require.NoError(t, err, "failed to create connection")

	err = manager.Connect(ctx, created.ID)
	require.NoError(t, err, "failed to connect to redis")

	t.Cleanup(func() {
		_ = manager.Disconnect(context.Background(), created.ID)
	})

	return kvSvc, manager, created.ID
}

func TestRedisConnection(t *testing.T) {
	ctx := context.Background()

	store := memory.NewConnStore()
	factory := outbound.NewFactory()
	manager := app.NewConnectionManager(store, factory)

	conn := connection.Connection{
		Name:     "test-redis-conn",
		Engine:   domain.EngineRedis,
		Category: domain.CategoryKV,
		Host:     redisHost,
		Port:     redisPort,
	}

	created, err := manager.Create(ctx, conn)
	require.NoError(t, err)

	// Verify initial status is disconnected
	fetched, err := manager.Get(ctx, created.ID)
	require.NoError(t, err)
	if fetched.Status != domain.StatusDisconnected {
		t.Errorf("expected status %q, got %q", domain.StatusDisconnected, fetched.Status)
	}

	// Connect and verify status
	err = manager.Connect(ctx, created.ID)
	require.NoError(t, err)

	fetched, err = manager.Get(ctx, created.ID)
	require.NoError(t, err)
	if fetched.Status != domain.StatusConnected {
		t.Errorf("expected status %q after connect, got %q", domain.StatusConnected, fetched.Status)
	}

	// Disconnect and verify status
	err = manager.Disconnect(ctx, created.ID)
	require.NoError(t, err)

	fetched, err = manager.Get(ctx, created.ID)
	require.NoError(t, err)
	if fetched.Status != domain.StatusDisconnected {
		t.Errorf("expected status %q after disconnect, got %q", domain.StatusDisconnected, fetched.Status)
	}
}

func TestRedisSetAndGet(t *testing.T) {
	ctx := context.Background()
	kvSvc, _, connID := newKVServiceWithConn(t)

	t.Run("set and get string key", func(t *testing.T) {
		err := kvSvc.Set(ctx, connID, "test:hello", "world", nil)
		require.NoError(t, err)

		entry, err := kvSvc.Get(ctx, connID, "test:hello")
		require.NoError(t, err)

		if entry.Value != "world" {
			t.Errorf("expected value %q, got %q", "world", entry.Value)
		}
		if entry.Type != "string" {
			t.Errorf("expected type %q, got %q", "string", entry.Type)
		}
	})

	t.Run("set key with TTL", func(t *testing.T) {
		ttl := int64(60)
		err := kvSvc.Set(ctx, connID, "test:ttl", "expires-soon", &ttl)
		require.NoError(t, err)

		entry, err := kvSvc.Get(ctx, connID, "test:ttl")
		require.NoError(t, err)

		if entry.TTL == nil {
			t.Fatal("expected TTL to be set, got nil")
		}
		if *entry.TTL <= 0 {
			t.Errorf("expected TTL > 0, got %d", *entry.TTL)
		}
	})
}

func TestRedisGetKeys(t *testing.T) {
	ctx := context.Background()
	kvSvc, _, connID := newKVServiceWithConn(t)

	// Set several keys with a unique prefix to avoid cross-test pollution
	keys := []string{"getkeytest:a", "getkeytest:b", "getkeytest:c"}
	for _, k := range keys {
		err := kvSvc.Set(ctx, connID, k, "value", nil)
		require.NoError(t, err)
	}

	t.Run("get all keys matching pattern", func(t *testing.T) {
		entries, err := kvSvc.GetKeys(ctx, connID, "getkeytest:*", 100)
		require.NoError(t, err)

		if len(entries) < 3 {
			t.Errorf("expected at least 3 keys, got %d", len(entries))
		}

		keySet := make(map[string]bool)
		for _, e := range entries {
			keySet[e.Key] = true
		}
		for _, k := range keys {
			if !keySet[k] {
				t.Errorf("expected key %q in results", k)
			}
		}
	})

	t.Run("get keys with limit", func(t *testing.T) {
		entries, err := kvSvc.GetKeys(ctx, connID, "getkeytest:*", 2)
		require.NoError(t, err)

		if len(entries) > 2 {
			t.Errorf("expected at most 2 keys with limit=2, got %d", len(entries))
		}
	})
}

func TestRedisDelete(t *testing.T) {
	ctx := context.Background()
	kvSvc, _, connID := newKVServiceWithConn(t)

	// Set a key to delete
	err := kvSvc.Set(ctx, connID, "test:delete", "to-be-deleted", nil)
	require.NoError(t, err)

	// Verify it exists
	_, err = kvSvc.Get(ctx, connID, "test:delete")
	require.NoError(t, err)

	// Delete the key
	err = kvSvc.Delete(ctx, connID, "test:delete")
	require.NoError(t, err)

	// Try to get deleted key — expect an error
	_, err = kvSvc.Get(ctx, connID, "test:delete")
	if err == nil {
		t.Error("expected error when getting deleted key, got nil")
	}
}

func TestRedisGetStats(t *testing.T) {
	ctx := context.Background()
	kvSvc, _, connID := newKVServiceWithConn(t)

	stats, err := kvSvc.GetStats(ctx, connID)
	require.NoError(t, err)

	if stats.TotalKeys < 0 {
		t.Errorf("expected TotalKeys >= 0, got %d", stats.TotalKeys)
	}

	if stats.MemoryUsed == "" {
		t.Error("expected MemoryUsed to be non-empty")
	}

	if stats.ConnectedClients < 1 {
		t.Errorf("expected ConnectedClients >= 1, got %d", stats.ConnectedClients)
	}
}

func TestRedisDataTypes(t *testing.T) {
	ctx := context.Background()
	kvSvc, _, connID := newKVServiceWithConn(t)

	t.Run("string type and encoding", func(t *testing.T) {
		err := kvSvc.Set(ctx, connID, "test:dtype:string", "hello", nil)
		require.NoError(t, err)

		entry, err := kvSvc.Get(ctx, connID, "test:dtype:string")
		require.NoError(t, err)

		if entry.Type != "string" {
			t.Errorf("expected type %q, got %q", "string", entry.Type)
		}

		// Short strings use embstr encoding; longer ones use raw.
		// Either is acceptable — just verify encoding is populated.
		if entry.Encoding == "" {
			t.Error("expected encoding to be non-empty for a string key")
		}
		if entry.Encoding != "embstr" && entry.Encoding != "raw" && entry.Encoding != "int" {
			t.Errorf("unexpected encoding %q for string key", entry.Encoding)
		}
	})
}
