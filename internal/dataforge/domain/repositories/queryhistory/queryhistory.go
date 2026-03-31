package queryhistory

import (
	"context"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlquery"
)

type Repository interface {
	Save(ctx context.Context, entry sqlquery.HistoryEntry) error
	List(ctx context.Context, connID string) ([]sqlquery.HistoryEntry, error)
}
