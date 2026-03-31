package filecache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	"github.com/JLugagne/forscadb/internal/domain"
)

type connRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Engine   string `json:"engine"`
	Category string `json:"category"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password,omitempty"`
	Database string `json:"database,omitempty"`
	SSLMode  string `json:"sslMode,omitempty"`
	Color    string `json:"color,omitempty"`
}

func toRecord(c connection.Connection) connRecord {
	return connRecord{
		ID:       c.ID,
		Name:     c.Name,
		Engine:   string(c.Engine),
		Category: string(c.Category),
		Host:     c.Host,
		Port:     c.Port,
		User:     c.User,
		Password: c.Password,
		Database: c.Database,
		SSLMode:  c.SSLMode,
		Color:    c.Color,
	}
}

func fromRecord(r connRecord) connection.Connection {
	return connection.Connection{
		ID:       r.ID,
		Name:     r.Name,
		Engine:   domain.DatabaseEngine(r.Engine),
		Category: domain.DatabaseCategory(r.Category),
		Host:     r.Host,
		Port:     r.Port,
		User:     r.User,
		Password: r.Password,
		Database: r.Database,
		SSLMode:  r.SSLMode,
		Status:   domain.StatusDisconnected,
		Color:    r.Color,
	}
}

type ConnStore struct {
	mu   sync.RWMutex
	path string
	data map[string]connRecord
	key  []byte
}

func NewConnStore(dir string) (*ConnStore, error) {
	key, err := masterKey()
	if err != nil {
		return nil, fmt.Errorf("filecache: init encryption key: %w", err)
	}
	return newConnStoreWithKey(dir, key)
}

// NewConnStoreWithKey creates a ConnStore using the provided AES-256 key.
// Intended for testing; production code should use NewConnStore.
func NewConnStoreWithKey(dir string, key []byte) (*ConnStore, error) {
	return newConnStoreWithKey(dir, key)
}

func newConnStoreWithKey(dir string, key []byte) (*ConnStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("filecache: create dir: %w", err)
	}
	s := &ConnStore{
		path: filepath.Join(dir, "connections.json"),
		data: make(map[string]connRecord),
		key:  key,
	}
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("filecache: load connections: %w", err)
	}
	return s, nil
}

func (s *ConnStore) load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var records []connRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		return err
	}
	s.data = make(map[string]connRecord, len(records))
	for _, r := range records {
		plain, err := decryptPassword(r.Password, s.key)
		if err != nil {
			return fmt.Errorf("filecache: decrypt password for %s: %w", r.ID, err)
		}
		r.Password = plain
		s.data[r.ID] = r
	}
	return nil
}

func (s *ConnStore) flush() error {
	records := make([]connRecord, 0, len(s.data))
	for _, r := range s.data {
		encrypted, err := encryptPassword(r.Password, s.key)
		if err != nil {
			return fmt.Errorf("filecache: encrypt password for %s: %w", r.ID, err)
		}
		r.Password = encrypted
		records = append(records, r)
	}
	raw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

func (s *ConnStore) Save(_ context.Context, conn connection.Connection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[conn.ID] = toRecord(conn)
	return s.flush()
}

func (s *ConnStore) Get(_ context.Context, id string) (connection.Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.data[id]
	if !ok {
		return connection.Connection{}, fmt.Errorf("filecache: connection not found: %s", id)
	}
	return fromRecord(r), nil
}

func (s *ConnStore) List(_ context.Context) ([]connection.Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conns := make([]connection.Connection, 0, len(s.data))
	for _, r := range s.data {
		conns = append(conns, fromRecord(r))
	}
	return conns, nil
}

func (s *ConnStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return fmt.Errorf("filecache: connection not found: %s", id)
	}
	delete(s.data, id)
	return s.flush()
}
