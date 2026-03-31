package connection

import (
	"context"

	"github.com/JLugagne/forscadb/internal/domain"
)

type Connection struct {
	ID       string
	Name     string
	Engine   domain.DatabaseEngine
	Category domain.DatabaseCategory
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string // disable, require, verify-ca, verify-full (PG); true, false, skip-verify (MySQL); etc.
	Status   domain.ConnectionStatus
	Color    string
}

type Commands interface {
	Create(ctx context.Context, conn Connection) (Connection, error)
	Update(ctx context.Context, conn Connection) (Connection, error)
	Delete(ctx context.Context, id string) error
	Connect(ctx context.Context, id string) error
	Disconnect(ctx context.Context, id string) error
}

type Queries interface {
	Get(ctx context.Context, id string) (Connection, error)
	List(ctx context.Context) ([]Connection, error)
	TestConnection(ctx context.Context, conn Connection) error
}
