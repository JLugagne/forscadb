package nosql

import "context"

type Index struct {
	Name   string
	Keys   map[string]int
	Unique bool
}

type Collection struct {
	Name          string
	DocumentCount int64
	AvgDocSize    string
	TotalSize     string
	Indexes       []Index
}

type Document = map[string]any

type Commands interface {
	InsertDocument(ctx context.Context, connID string, collection string, doc Document) (Document, error)
	UpdateDocument(ctx context.Context, connID string, collection string, id string, doc Document) (Document, error)
	DeleteDocument(ctx context.Context, connID string, collection string, id string) error
	CreateCollection(ctx context.Context, connID string, name string) error
	DropCollection(ctx context.Context, connID string, name string) error
}

type Queries interface {
	GetCollections(ctx context.Context, connID string) ([]Collection, error)
	GetDocuments(ctx context.Context, connID string, collection string, filter string, limit int) ([]Document, error)
}
