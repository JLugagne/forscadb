package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
)

type ConnStore struct {
	mu    sync.RWMutex
	store map[string]connection.Connection
}

func NewConnStore() *ConnStore {
	return &ConnStore{
		store: make(map[string]connection.Connection),
	}
}

func (s *ConnStore) Save(_ context.Context, conn connection.Connection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[conn.ID] = conn
	return nil
}

func (s *ConnStore) Get(_ context.Context, id string) (connection.Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, ok := s.store[id]
	if !ok {
		return connection.Connection{}, fmt.Errorf("memory: connstore: connection not found: %s", id)
	}
	return conn, nil
}

func (s *ConnStore) List(_ context.Context) ([]connection.Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conns := make([]connection.Connection, 0, len(s.store))
	for _, c := range s.store {
		conns = append(conns, c)
	}
	return conns, nil
}

func (s *ConnStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.store[id]; !ok {
		return fmt.Errorf("memory: connstore: connection not found: %s", id)
	}
	delete(s.store, id)
	return nil
}
