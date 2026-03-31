package connstore

import (
	"context"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
)

type Repository interface {
	Save(ctx context.Context, conn connection.Connection) error
	Get(ctx context.Context, id string) (connection.Connection, error)
	List(ctx context.Context) ([]connection.Connection, error)
	Delete(ctx context.Context, id string) error
}
