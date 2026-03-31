package main

import (
	"context"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/kv"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/nosql"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlintrospect"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlquery"
	"github.com/JLugagne/forscadb/internal/dataforge/inbound/wails/converters"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound/filecache"
)

type App struct {
	ctx      context.Context
	cacheDir string

	connCommands  connection.Commands
	connQueries   connection.Queries
	sqlIntrospect sqlintrospect.Queries
	sqlCommands   sqlquery.Commands
	sqlQueries    sqlquery.Queries
	nosqlCommands nosql.Commands
	nosqlQueries  nosql.Queries
	kvCommands    kv.Commands
	kvQueries     kv.Queries
}

func NewApp(
	cacheDir string,
	connCommands connection.Commands,
	connQueries connection.Queries,
	sqlIntrospect sqlintrospect.Queries,
	sqlCommands sqlquery.Commands,
	sqlQueries sqlquery.Queries,
	nosqlCommands nosql.Commands,
	nosqlQueries nosql.Queries,
	kvCommands kv.Commands,
	kvQueries kv.Queries,
) *App {
	return &App{
		cacheDir:      cacheDir,
		connCommands:  connCommands,
		connQueries:   connQueries,
		sqlIntrospect: sqlIntrospect,
		sqlCommands:   sqlCommands,
		sqlQueries:    sqlQueries,
		nosqlCommands: nosqlCommands,
		nosqlQueries:  nosqlQueries,
		kvCommands:    kvCommands,
		kvQueries:     kvQueries,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) CreateConnection(input converters.PublicConnection) (converters.PublicConnection, error) {
	result, err := a.connCommands.Create(a.ctx, converters.ToDomainConnection(input))
	if err != nil {
		return converters.PublicConnection{}, err
	}
	return converters.ToPublicConnection(result), nil
}

func (a *App) UpdateConnection(input converters.PublicConnection) (converters.PublicConnection, error) {
	result, err := a.connCommands.Update(a.ctx, converters.ToDomainConnection(input))
	if err != nil {
		return converters.PublicConnection{}, err
	}
	return converters.ToPublicConnection(result), nil
}

func (a *App) DeleteConnection(id string) error {
	return a.connCommands.Delete(a.ctx, id)
}

func (a *App) ConnectToDatabase(id string) error {
	return a.connCommands.Connect(a.ctx, id)
}

func (a *App) DisconnectFromDatabase(id string) error {
	return a.connCommands.Disconnect(a.ctx, id)
}

func (a *App) ListConnections() ([]converters.PublicConnection, error) {
	list, err := a.connQueries.List(a.ctx)
	if err != nil {
		return nil, err
	}
	out := make([]converters.PublicConnection, len(list))
	for i, c := range list {
		out[i] = converters.ToPublicConnection(c)
	}
	return out, nil
}

func (a *App) GetConnection(id string) (converters.PublicConnection, error) {
	c, err := a.connQueries.Get(a.ctx, id)
	if err != nil {
		return converters.PublicConnection{}, err
	}
	return converters.ToPublicConnection(c), nil
}

func (a *App) TestConnection(input converters.PublicConnection) error {
	return a.connQueries.TestConnection(a.ctx, converters.ToDomainConnection(input))
}

func (a *App) GetTables(connID string) ([]converters.PublicSQLTable, error) {
	tables, err := a.sqlIntrospect.GetTables(a.ctx, connID)
	if err != nil {
		return nil, err
	}
	return converters.ToPublicTables(tables), nil
}

func (a *App) GetViews(connID string) ([]converters.PublicSQLView, error) {
	views, err := a.sqlIntrospect.GetViews(a.ctx, connID)
	if err != nil {
		return nil, err
	}
	return converters.ToPublicViews(views), nil
}

func (a *App) GetFunctions(connID string) ([]converters.PublicSQLFunction, error) {
	fns, err := a.sqlIntrospect.GetFunctions(a.ctx, connID)
	if err != nil {
		return nil, err
	}
	return converters.ToPublicFunctions(fns), nil
}

func (a *App) GetTriggers(connID string) ([]converters.PublicSQLTrigger, error) {
	triggers, err := a.sqlIntrospect.GetTriggers(a.ctx, connID)
	if err != nil {
		return nil, err
	}
	return converters.ToPublicTriggers(triggers), nil
}

func (a *App) GetSequences(connID string) ([]converters.PublicSQLSequence, error) {
	seqs, err := a.sqlIntrospect.GetSequences(a.ctx, connID)
	if err != nil {
		return nil, err
	}
	return converters.ToPublicSequences(seqs), nil
}

func (a *App) GetEnums(connID string) ([]converters.PublicSQLEnum, error) {
	enums, err := a.sqlIntrospect.GetEnums(a.ctx, connID)
	if err != nil {
		return nil, err
	}
	return converters.ToPublicEnums(enums), nil
}

func (a *App) ExecuteQuery(connID string, query string) (converters.PublicQueryResult, error) {
	result, err := a.sqlCommands.Execute(a.ctx, connID, query)
	if err != nil {
		return converters.PublicQueryResult{}, err
	}
	return converters.ToPublicQueryResult(result), nil
}

func (a *App) GetTableData(connID string, schema string, table string, limit int, offset int) (converters.PublicQueryResult, error) {
	result, err := a.sqlQueries.GetTableData(a.ctx, connID, schema, table, limit, offset)
	if err != nil {
		return converters.PublicQueryResult{}, err
	}
	return converters.ToPublicQueryResult(result), nil
}

func (a *App) GetQueryHistory(connID string) ([]converters.PublicHistoryEntry, error) {
	entries, err := a.sqlQueries.GetHistory(a.ctx, connID)
	if err != nil {
		return nil, err
	}
	return converters.ToPublicHistoryEntries(entries), nil
}

func (a *App) GetCollections(connID string) ([]converters.PublicNoSQLCollection, error) {
	cols, err := a.nosqlQueries.GetCollections(a.ctx, connID)
	if err != nil {
		return nil, err
	}
	return converters.ToPublicCollections(cols), nil
}

func (a *App) GetDocuments(connID string, collection string, filter string, limit int) ([]map[string]any, error) {
	return a.nosqlQueries.GetDocuments(a.ctx, connID, collection, filter, limit)
}

func (a *App) InsertDocument(connID string, collection string, doc map[string]any) (map[string]any, error) {
	return a.nosqlCommands.InsertDocument(a.ctx, connID, collection, doc)
}

func (a *App) UpdateDocument(connID string, collection string, id string, doc map[string]any) (map[string]any, error) {
	return a.nosqlCommands.UpdateDocument(a.ctx, connID, collection, id, doc)
}

func (a *App) DeleteDocument(connID string, collection string, id string) error {
	return a.nosqlCommands.DeleteDocument(a.ctx, connID, collection, id)
}

func (a *App) GetKVStats(connID string) (converters.PublicKVStats, error) {
	stats, err := a.kvQueries.GetStats(a.ctx, connID)
	if err != nil {
		return converters.PublicKVStats{}, err
	}
	return converters.ToPublicKVStats(stats), nil
}

func (a *App) GetKeys(connID string, pattern string, limit int) ([]converters.PublicKVEntry, error) {
	entries, err := a.kvQueries.GetKeys(a.ctx, connID, pattern, limit)
	if err != nil {
		return nil, err
	}
	return converters.ToPublicKVEntries(entries), nil
}

func (a *App) GetKVEntry(connID string, key string) (converters.PublicKVEntry, error) {
	entry, err := a.kvQueries.Get(a.ctx, connID, key)
	if err != nil {
		return converters.PublicKVEntry{}, err
	}
	return converters.ToPublicKVEntry(entry), nil
}

func (a *App) SetKVEntry(connID string, key string, value string, ttl *int64) error {
	return a.kvCommands.Set(a.ctx, connID, key, value, ttl)
}

func (a *App) DeleteKVEntry(connID string, key string) error {
	return a.kvCommands.Delete(a.ctx, connID, key)
}

func (a *App) DropTable(connID string, schema string, table string) error {
	return a.sqlCommands.DropTable(a.ctx, connID, schema, table)
}

func (a *App) AddColumn(connID string, schema string, table string, name string, colType string, nullable bool, defaultVal string) error {
	return a.sqlCommands.AddColumn(a.ctx, connID, schema, table, name, colType, nullable, defaultVal)
}

func (a *App) RefreshMaterializedView(connID string, schema string, name string) error {
	return a.sqlCommands.RefreshMaterializedView(a.ctx, connID, schema, name)
}

func (a *App) RenameColumn(connID, schema, table, oldName, newName string) error {
	return a.sqlCommands.RenameColumn(a.ctx, connID, schema, table, oldName, newName)
}

func (a *App) AlterColumnType(connID, schema, table, column, newType string) error {
	return a.sqlCommands.AlterColumnType(a.ctx, connID, schema, table, column, newType)
}

func (a *App) DropColumn(connID, schema, table, column string) error {
	return a.sqlCommands.DropColumn(a.ctx, connID, schema, table, column)
}

func (a *App) SetColumnNullable(connID, schema, table, column string, nullable bool) error {
	return a.sqlCommands.SetColumnNullable(a.ctx, connID, schema, table, column, nullable)
}

func (a *App) SetColumnDefault(connID, schema, table, column, defaultVal string) error {
	return a.sqlCommands.SetColumnDefault(a.ctx, connID, schema, table, column, defaultVal)
}

func (a *App) ExplainQuery(connID string, query string, analyze bool) (converters.PublicExplainResult, error) {
	result, err := a.sqlCommands.ExplainQuery(a.ctx, connID, query, analyze)
	if err != nil {
		return converters.PublicExplainResult{}, err
	}
	return converters.ToPublicExplainResult(result), nil
}

func (a *App) CreateCollection(connID string, name string) error {
	return a.nosqlCommands.CreateCollection(a.ctx, connID, name)
}

func (a *App) DropCollection(connID string, name string) error {
	return a.nosqlCommands.DropCollection(a.ctx, connID, name)
}

func (a *App) GetSidebarTree() (string, error) {
	return filecache.LoadSidebarTree(a.cacheDir)
}

func (a *App) SaveSidebarTree(tree string) error {
	return filecache.SaveSidebarTree(a.cacheDir, tree)
}
