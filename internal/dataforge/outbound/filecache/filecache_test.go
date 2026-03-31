package filecache_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlquery"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound/filecache"
	"github.com/JLugagne/forscadb/internal/domain"
)

// testKey is a fixed 32-byte AES-256 key used in tests to avoid keyring access.
var testKey = []byte("test-key-32-bytes-padded-_______")

func TestConnStore_PersistAndReload(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// Create store, save a connection
	store, err := filecache.NewConnStoreWithKey(dir, testKey)
	if err != nil {
		t.Fatal(err)
	}

	conn := connection.Connection{
		ID:       "test-1",
		Name:     "My PG",
		Engine:   domain.EnginePostgreSQL,
		Category: domain.CategorySQL,
		Host:     "localhost",
		Port:     5432,
		User:     "admin",
		Password: "secret",
		Database: "mydb",
		Color:    "#336791",
	}
	if err := store.Save(ctx, conn); err != nil {
		t.Fatal(err)
	}

	// Create a new store from the same directory — should load persisted data
	store2, err := filecache.NewConnStoreWithKey(dir, testKey)
	if err != nil {
		t.Fatal(err)
	}

	got, err := store2.Get(ctx, "test-1")
	if err != nil {
		t.Fatalf("Get after reload: %v", err)
	}
	if got.Name != "My PG" {
		t.Errorf("expected name 'My PG', got %q", got.Name)
	}
	if got.Password != "secret" {
		t.Errorf("expected password 'secret', got %q", got.Password)
	}
	if got.Status != domain.StatusDisconnected {
		t.Errorf("expected status disconnected after reload, got %q", got.Status)
	}

	// List should return 1
	list, err := store2.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 connection, got %d", len(list))
	}

	// Delete and verify
	if err := store2.Delete(ctx, "test-1"); err != nil {
		t.Fatal(err)
	}
	store3, err := filecache.NewConnStoreWithKey(dir, testKey)
	if err != nil {
		t.Fatal(err)
	}
	list, err = store3.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 connections after delete, got %d", len(list))
	}
}

func TestQueryHistory_PersistAndReload(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	h, err := filecache.NewQueryHistory(dir)
	if err != nil {
		t.Fatal(err)
	}

	entry := sqlquery.HistoryEntry{
		ID:           "h-1",
		ConnectionID: "conn-1",
		Query:        "SELECT 1",
		ExecutedAt:   "2026-01-01T00:00:00Z",
		Duration:     1.5,
		RowCount:     1,
		Status:       "success",
	}
	if err := h.Save(ctx, entry); err != nil {
		t.Fatal(err)
	}

	// Reload from disk
	h2, err := filecache.NewQueryHistory(dir)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := h2.List(ctx, "conn-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Query != "SELECT 1" {
		t.Errorf("expected query 'SELECT 1', got %q", entries[0].Query)
	}

	// Different connID returns empty
	entries, err = h2.List(ctx, "conn-other")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for other conn, got %d", len(entries))
	}
}

func TestConnStore_FileCreated(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	store, err := filecache.NewConnStoreWithKey(dir, testKey)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Save(ctx, connection.Connection{ID: "x", Name: "X"}); err != nil {
		t.Fatal(err)
	}

	// File should exist
	path := filepath.Join(dir, "connections.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("connections.json not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("connections.json is empty")
	}
}
