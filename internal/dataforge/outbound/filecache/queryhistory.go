package filecache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlquery"
)

const maxHistoryPerConnection = 200

type QueryHistory struct {
	mu   sync.RWMutex
	path string
	data map[string][]sqlquery.HistoryEntry // connID -> entries
}

func NewQueryHistory(dir string) (*QueryHistory, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("filecache: create dir: %w", err)
	}
	h := &QueryHistory{
		path: filepath.Join(dir, "history.json"),
		data: make(map[string][]sqlquery.HistoryEntry),
	}
	if err := h.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("filecache: load history: %w", err)
	}
	return h, nil
}

func (h *QueryHistory) load() error {
	raw, err := os.ReadFile(h.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &h.data)
}

func (h *QueryHistory) flush() error {
	raw, err := json.MarshalIndent(h.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.path, raw, 0o644)
}

func (h *QueryHistory) Save(_ context.Context, entry sqlquery.HistoryEntry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	entries := h.data[entry.ConnectionID]
	entries = append(entries, entry)
	if len(entries) > maxHistoryPerConnection {
		entries = entries[len(entries)-maxHistoryPerConnection:]
	}
	h.data[entry.ConnectionID] = entries
	return h.flush()
}

func (h *QueryHistory) List(_ context.Context, connID string) ([]sqlquery.HistoryEntry, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entries := h.data[connID]
	if entries == nil {
		return []sqlquery.HistoryEntry{}, nil
	}
	result := make([]sqlquery.HistoryEntry, len(entries))
	copy(result, entries)
	return result, nil
}
