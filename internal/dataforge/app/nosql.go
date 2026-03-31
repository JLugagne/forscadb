package app

import (
	"context"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/nosql"
)

type NoSQLService struct {
	manager *ConnectionManager
}

func NewNoSQLService(manager *ConnectionManager) *NoSQLService {
	return &NoSQLService{manager: manager}
}

func (s *NoSQLService) GetCollections(ctx context.Context, connID string) ([]nosql.Collection, error) {
	d, err := s.manager.GetNoSQLDriver(connID)
	if err != nil {
		return nil, err
	}
	return d.GetCollections(ctx)
}

func (s *NoSQLService) GetDocuments(ctx context.Context, connID string, collection string, filter string, limit int) ([]nosql.Document, error) {
	d, err := s.manager.GetNoSQLDriver(connID)
	if err != nil {
		return nil, err
	}
	return d.GetDocuments(ctx, collection, filter, limit)
}

func (s *NoSQLService) InsertDocument(ctx context.Context, connID string, collection string, doc nosql.Document) (nosql.Document, error) {
	d, err := s.manager.GetNoSQLDriver(connID)
	if err != nil {
		return nil, err
	}
	return d.InsertDocument(ctx, collection, doc)
}

func (s *NoSQLService) UpdateDocument(ctx context.Context, connID string, collection string, id string, doc nosql.Document) (nosql.Document, error) {
	d, err := s.manager.GetNoSQLDriver(connID)
	if err != nil {
		return nil, err
	}
	return d.UpdateDocument(ctx, collection, id, doc)
}

func (s *NoSQLService) DeleteDocument(ctx context.Context, connID string, collection string, id string) error {
	d, err := s.manager.GetNoSQLDriver(connID)
	if err != nil {
		return err
	}
	return d.DeleteDocument(ctx, collection, id)
}

func (s *NoSQLService) CreateCollection(ctx context.Context, connID string, name string) error {
	d, err := s.manager.GetNoSQLDriver(connID)
	if err != nil {
		return err
	}
	return d.CreateCollection(ctx, name)
}

func (s *NoSQLService) DropCollection(ctx context.Context, connID string, name string) error {
	d, err := s.manager.GetNoSQLDriver(connID)
	if err != nil {
		return err
	}
	return d.DropCollection(ctx, name)
}
