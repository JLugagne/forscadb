package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/repositories/queryhistory"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlintrospect"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlquery"
)

type SQLService struct {
	manager *ConnectionManager
	history queryhistory.Repository
}

func NewSQLService(manager *ConnectionManager, history queryhistory.Repository) *SQLService {
	return &SQLService{
		manager: manager,
		history: history,
	}
}

func (s *SQLService) GetTables(ctx context.Context, connID string) ([]sqlintrospect.Table, error) {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return nil, err
	}
	return d.GetTables(ctx)
}

func (s *SQLService) GetViews(ctx context.Context, connID string) ([]sqlintrospect.View, error) {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return nil, err
	}
	return d.GetViews(ctx)
}

func (s *SQLService) GetFunctions(ctx context.Context, connID string) ([]sqlintrospect.Function, error) {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return nil, err
	}
	return d.GetFunctions(ctx)
}

func (s *SQLService) GetTriggers(ctx context.Context, connID string) ([]sqlintrospect.Trigger, error) {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return nil, err
	}
	return d.GetTriggers(ctx)
}

func (s *SQLService) GetSequences(ctx context.Context, connID string) ([]sqlintrospect.Sequence, error) {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return nil, err
	}
	return d.GetSequences(ctx)
}

func (s *SQLService) GetEnums(ctx context.Context, connID string) ([]sqlintrospect.Enum, error) {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return nil, err
	}
	return d.GetEnums(ctx)
}

func (s *SQLService) Execute(ctx context.Context, connID string, query string) (sqlquery.QueryResult, error) {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return sqlquery.QueryResult{}, err
	}

	start := time.Now()
	result, execErr := d.ExecuteQuery(ctx, query)
	duration := float64(time.Since(start).Milliseconds())

	entry := sqlquery.HistoryEntry{
		ID:           uuid.New().String(),
		ConnectionID: connID,
		Query:        query,
		ExecutedAt:   time.Now().UTC().Format(time.RFC3339),
		Duration:     duration,
	}

	if execErr != nil {
		errMsg := execErr.Error()
		entry.Status = "error"
		entry.Error = &errMsg
		_ = s.history.Save(ctx, entry)
		return sqlquery.QueryResult{}, execErr
	}

	entry.RowCount = result.RowCount
	entry.Status = "success"
	_ = s.history.Save(ctx, entry)

	return result, nil
}

func (s *SQLService) GetTableData(ctx context.Context, connID string, schema string, table string, limit int, offset int) (sqlquery.QueryResult, error) {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return sqlquery.QueryResult{}, err
	}
	return d.GetTableData(ctx, schema, table, limit, offset)
}

func (s *SQLService) DropTable(ctx context.Context, connID, schema, table string) error {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return err
	}
	return d.DropTable(ctx, schema, table)
}

func (s *SQLService) AddColumn(ctx context.Context, connID, schema, table, name, colType string, nullable bool, defaultVal string) error {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return err
	}
	return d.AddColumn(ctx, schema, table, name, colType, nullable, defaultVal)
}

func (s *SQLService) RefreshMaterializedView(ctx context.Context, connID, schema, name string) error {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return err
	}
	return d.RefreshMaterializedView(ctx, schema, name)
}

func (s *SQLService) RenameColumn(ctx context.Context, connID, schema, table, oldName, newName string) error {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return err
	}
	return d.RenameColumn(ctx, schema, table, oldName, newName)
}

func (s *SQLService) AlterColumnType(ctx context.Context, connID, schema, table, column, newType string) error {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return err
	}
	return d.AlterColumnType(ctx, schema, table, column, newType)
}

func (s *SQLService) DropColumn(ctx context.Context, connID, schema, table, column string) error {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return err
	}
	return d.DropColumn(ctx, schema, table, column)
}

func (s *SQLService) SetColumnNullable(ctx context.Context, connID, schema, table, column string, nullable bool) error {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return err
	}
	return d.SetColumnNullable(ctx, schema, table, column, nullable)
}

func (s *SQLService) SetColumnDefault(ctx context.Context, connID, schema, table, column, defaultVal string) error {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return err
	}
	return d.SetColumnDefault(ctx, schema, table, column, defaultVal)
}

func (s *SQLService) ExplainQuery(ctx context.Context, connID string, query string, analyze bool) (sqlquery.ExplainResult, error) {
	d, err := s.manager.GetSQLDriver(connID)
	if err != nil {
		return sqlquery.ExplainResult{}, err
	}
	return d.ExplainQuery(ctx, query, analyze)
}

func (s *SQLService) GetHistory(ctx context.Context, connID string) ([]sqlquery.HistoryEntry, error) {
	entries, err := s.history.List(ctx, connID)
	if err != nil {
		return nil, fmt.Errorf("list history for connection %s: %w", connID, err)
	}
	return entries, nil
}
