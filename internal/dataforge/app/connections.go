package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/repositories/connstore"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	"github.com/JLugagne/forscadb/internal/domain"
)

type ConnectionManager struct {
	store   connstore.Repository
	factory DriverFactory

	mu      sync.RWMutex
	drivers map[string]any
}

func NewConnectionManager(store connstore.Repository, factory DriverFactory) *ConnectionManager {
	return &ConnectionManager{
		store:   store,
		factory: factory,
		drivers: make(map[string]any),
	}
}

func (m *ConnectionManager) Create(ctx context.Context, conn connection.Connection) (connection.Connection, error) {
	conn.ID = uuid.New().String()
	conn.Status = domain.StatusDisconnected
	if err := m.store.Save(ctx, conn); err != nil {
		return connection.Connection{}, fmt.Errorf("save connection: %w", err)
	}
	return conn, nil
}

func (m *ConnectionManager) Update(ctx context.Context, conn connection.Connection) (connection.Connection, error) {
	existing, err := m.store.Get(ctx, conn.ID)
	if err != nil {
		return connection.Connection{}, fmt.Errorf("connection %s not found", conn.ID)
	}
	conn.Status = existing.Status
	if err := m.store.Save(ctx, conn); err != nil {
		return connection.Connection{}, fmt.Errorf("save connection: %w", err)
	}
	return conn, nil
}

func (m *ConnectionManager) Delete(ctx context.Context, id string) error {
	if err := m.Disconnect(ctx, id); err != nil {
		return fmt.Errorf("disconnect before delete: %w", err)
	}
	return m.store.Delete(ctx, id)
}

func (m *ConnectionManager) Connect(ctx context.Context, id string) error {
	conn, err := m.store.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("connection %s not found", id)
	}

	var driver any
	switch conn.Category {
	case domain.CategorySQL:
		d, err := m.factory.CreateSQLDriver(ctx, conn)
		if err != nil {
			return m.setErrorStatus(ctx, conn, fmt.Errorf("create SQL driver: %w", err))
		}
		if err := d.Ping(ctx); err != nil {
			_ = d.Close()
			return m.setErrorStatus(ctx, conn, fmt.Errorf("ping SQL driver: %w", err))
		}
		driver = d
	case domain.CategoryNoSQL:
		d, err := m.factory.CreateNoSQLDriver(ctx, conn)
		if err != nil {
			return m.setErrorStatus(ctx, conn, fmt.Errorf("create NoSQL driver: %w", err))
		}
		if err := d.Ping(ctx); err != nil {
			_ = d.Close()
			return m.setErrorStatus(ctx, conn, fmt.Errorf("ping NoSQL driver: %w", err))
		}
		driver = d
	case domain.CategoryKV:
		d, err := m.factory.CreateKVDriver(ctx, conn)
		if err != nil {
			return m.setErrorStatus(ctx, conn, fmt.Errorf("create KV driver: %w", err))
		}
		if err := d.Ping(ctx); err != nil {
			_ = d.Close()
			return m.setErrorStatus(ctx, conn, fmt.Errorf("ping KV driver: %w", err))
		}
		driver = d
	default:
		return fmt.Errorf("unknown category %s for connection %s", conn.Category, id)
	}

	m.mu.Lock()
	m.drivers[id] = driver
	m.mu.Unlock()

	conn.Status = domain.StatusConnected
	return m.store.Save(ctx, conn)
}

func (m *ConnectionManager) Disconnect(ctx context.Context, id string) error {
	m.mu.Lock()
	driver, ok := m.drivers[id]
	if ok {
		delete(m.drivers, id)
	}
	m.mu.Unlock()

	if ok {
		switch d := driver.(type) {
		case SQLDriver:
			_ = d.Close()
		case NoSQLDriver:
			_ = d.Close()
		case KVDriver:
			_ = d.Close()
		}
	}

	conn, err := m.store.Get(ctx, id)
	if err != nil {
		return nil
	}
	conn.Status = domain.StatusDisconnected
	return m.store.Save(ctx, conn)
}

func (m *ConnectionManager) TestConnection(ctx context.Context, conn connection.Connection) error {
	switch conn.Category {
	case domain.CategorySQL:
		d, err := m.factory.CreateSQLDriver(ctx, conn)
		if err != nil {
			return fmt.Errorf("create SQL driver: %w", err)
		}
		defer d.Close()
		return d.Ping(ctx)
	case domain.CategoryNoSQL:
		d, err := m.factory.CreateNoSQLDriver(ctx, conn)
		if err != nil {
			return fmt.Errorf("create NoSQL driver: %w", err)
		}
		defer d.Close()
		return d.Ping(ctx)
	case domain.CategoryKV:
		d, err := m.factory.CreateKVDriver(ctx, conn)
		if err != nil {
			return fmt.Errorf("create KV driver: %w", err)
		}
		defer d.Close()
		return d.Ping(ctx)
	default:
		return fmt.Errorf("unknown category %s", conn.Category)
	}
}

func (m *ConnectionManager) Get(ctx context.Context, id string) (connection.Connection, error) {
	return m.store.Get(ctx, id)
}

func (m *ConnectionManager) List(ctx context.Context) ([]connection.Connection, error) {
	return m.store.List(ctx)
}

func (m *ConnectionManager) GetSQLDriver(connID string) (SQLDriver, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.drivers[connID]
	if !ok {
		return nil, fmt.Errorf("connection %s not found", connID)
	}
	sql, ok := d.(SQLDriver)
	if !ok {
		return nil, fmt.Errorf("connection %s is not a SQL connection", connID)
	}
	return sql, nil
}

func (m *ConnectionManager) GetNoSQLDriver(connID string) (NoSQLDriver, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.drivers[connID]
	if !ok {
		return nil, fmt.Errorf("connection %s not found", connID)
	}
	nosql, ok := d.(NoSQLDriver)
	if !ok {
		return nil, fmt.Errorf("connection %s is not a NoSQL connection", connID)
	}
	return nosql, nil
}

func (m *ConnectionManager) GetKVDriver(connID string) (KVDriver, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.drivers[connID]
	if !ok {
		return nil, fmt.Errorf("connection %s not found", connID)
	}
	kv, ok := d.(KVDriver)
	if !ok {
		return nil, fmt.Errorf("connection %s is not a KV connection", connID)
	}
	return kv, nil
}

func (m *ConnectionManager) setErrorStatus(ctx context.Context, conn connection.Connection, err error) error {
	conn.Status = domain.StatusError
	_ = m.store.Save(ctx, conn)
	return err
}
