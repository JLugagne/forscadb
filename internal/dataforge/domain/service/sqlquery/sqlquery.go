package sqlquery

import "context"

type QueryResult struct {
	Columns       []string
	Rows          []map[string]any
	RowCount      int64
	ExecutionTime float64
	AffectedRows  *int64
}

type HistoryEntry struct {
	ID           string
	ConnectionID string
	Query        string
	ExecutedAt   string
	Duration     float64
	RowCount     int64
	Status       string
	Error        *string
}

type ExplainResult struct {
	Plan      string
	Format    string
	QueryText string
	PlanRows  []ExplainRow
}

type ExplainRow struct {
	Text   string
	Level  int
	IsNode bool
}

type Commands interface {
	Execute(ctx context.Context, connID string, query string) (QueryResult, error)
	DropTable(ctx context.Context, connID, schema, table string) error
	AddColumn(ctx context.Context, connID, schema, table, name, colType string, nullable bool, defaultVal string) error
	RefreshMaterializedView(ctx context.Context, connID, schema, name string) error
	RenameColumn(ctx context.Context, connID, schema, table, oldName, newName string) error
	AlterColumnType(ctx context.Context, connID, schema, table, column, newType string) error
	DropColumn(ctx context.Context, connID, schema, table, column string) error
	SetColumnNullable(ctx context.Context, connID, schema, table, column string, nullable bool) error
	SetColumnDefault(ctx context.Context, connID, schema, table, column, defaultVal string) error
	ExplainQuery(ctx context.Context, connID string, query string, analyze bool) (ExplainResult, error)
}

type Queries interface {
	GetTableData(ctx context.Context, connID string, schema string, table string, limit int, offset int) (QueryResult, error)
	GetHistory(ctx context.Context, connID string) ([]HistoryEntry, error)
}
