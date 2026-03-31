package memory

import (
	"context"
	"sync"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlquery"
)

const maxHistoryPerConnection = 100

type QueryHistory struct {
	mu    sync.RWMutex
	store map[string][]sqlquery.HistoryEntry
}

func NewQueryHistory() *QueryHistory {
	return &QueryHistory{
		store: make(map[string][]sqlquery.HistoryEntry),
	}
}

func (h *QueryHistory) Save(_ context.Context, entry sqlquery.HistoryEntry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	entries := h.store[entry.ConnectionID]
	entries = append(entries, entry)
	if len(entries) > maxHistoryPerConnection {
		entries = entries[len(entries)-maxHistoryPerConnection:]
	}
	h.store[entry.ConnectionID] = entries
	return nil
}

func (h *QueryHistory) List(_ context.Context, connID string) ([]sqlquery.HistoryEntry, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entries := h.store[connID]
	if entries == nil {
		return []sqlquery.HistoryEntry{}, nil
	}
	result := make([]sqlquery.HistoryEntry, len(entries))
	copy(result, entries)
	return result, nil
}
