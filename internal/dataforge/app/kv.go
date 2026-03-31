package app

import (
	"context"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/kv"
)

type KVService struct {
	manager *ConnectionManager
}

func NewKVService(manager *ConnectionManager) *KVService {
	return &KVService{manager: manager}
}

func (s *KVService) GetStats(ctx context.Context, connID string) (kv.Stats, error) {
	d, err := s.manager.GetKVDriver(connID)
	if err != nil {
		return kv.Stats{}, err
	}
	return d.GetStats(ctx)
}

func (s *KVService) GetKeys(ctx context.Context, connID string, pattern string, limit int) ([]kv.Entry, error) {
	d, err := s.manager.GetKVDriver(connID)
	if err != nil {
		return nil, err
	}
	return d.GetKeys(ctx, pattern, limit)
}

func (s *KVService) Get(ctx context.Context, connID string, key string) (kv.Entry, error) {
	d, err := s.manager.GetKVDriver(connID)
	if err != nil {
		return kv.Entry{}, err
	}
	return d.Get(ctx, key)
}

func (s *KVService) Set(ctx context.Context, connID string, key string, value string, ttlSeconds *int64) error {
	d, err := s.manager.GetKVDriver(connID)
	if err != nil {
		return err
	}
	return d.Set(ctx, key, value, ttlSeconds)
}

func (s *KVService) Delete(ctx context.Context, connID string, key string) error {
	d, err := s.manager.GetKVDriver(connID)
	if err != nil {
		return err
	}
	return d.Delete(ctx, key)
}
