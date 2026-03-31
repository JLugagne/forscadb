package app

import (
	"context"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/kv"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/nosql"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlintrospect"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlquery"
)

type SQLDriver interface {
	GetTables(ctx context.Context) ([]sqlintrospect.Table, error)
	GetViews(ctx context.Context) ([]sqlintrospect.View, error)
	GetFunctions(ctx context.Context) ([]sqlintrospect.Function, error)
	GetTriggers(ctx context.Context) ([]sqlintrospect.Trigger, error)
	GetSequences(ctx context.Context) ([]sqlintrospect.Sequence, error)
	GetEnums(ctx context.Context) ([]sqlintrospect.Enum, error)
	ExecuteQuery(ctx context.Context, query string) (sqlquery.QueryResult, error)
	GetTableData(ctx context.Context, schema, table string, limit, offset int) (sqlquery.QueryResult, error)
	DropTable(ctx context.Context, schema, table string) error
	AddColumn(ctx context.Context, schema, table, name, colType string, nullable bool, defaultVal string) error
	RefreshMaterializedView(ctx context.Context, schema, name string) error
	RenameColumn(ctx context.Context, schema, table, oldName, newName string) error
	AlterColumnType(ctx context.Context, schema, table, column, newType string) error
	DropColumn(ctx context.Context, schema, table, column string) error
	SetColumnNullable(ctx context.Context, schema, table, column string, nullable bool) error
	SetColumnDefault(ctx context.Context, schema, table, column, defaultVal string) error
	ExplainQuery(ctx context.Context, query string, analyze bool) (sqlquery.ExplainResult, error)
	Close() error
	Ping(ctx context.Context) error
}

type NoSQLDriver interface {
	GetCollections(ctx context.Context) ([]nosql.Collection, error)
	GetDocuments(ctx context.Context, collection string, filter string, limit int) ([]nosql.Document, error)
	InsertDocument(ctx context.Context, collection string, doc nosql.Document) (nosql.Document, error)
	UpdateDocument(ctx context.Context, collection string, id string, doc nosql.Document) (nosql.Document, error)
	DeleteDocument(ctx context.Context, collection string, id string) error
	CreateCollection(ctx context.Context, name string) error
	DropCollection(ctx context.Context, name string) error
	Close() error
	Ping(ctx context.Context) error
}

type KVDriver interface {
	GetStats(ctx context.Context) (kv.Stats, error)
	GetKeys(ctx context.Context, pattern string, limit int) ([]kv.Entry, error)
	Get(ctx context.Context, key string) (kv.Entry, error)
	Set(ctx context.Context, key string, value string, ttlSeconds *int64) error
	Delete(ctx context.Context, key string) error
	Close() error
	Ping(ctx context.Context) error
}

type DriverFactory interface {
	CreateSQLDriver(ctx context.Context, conn connection.Connection) (SQLDriver, error)
	CreateNoSQLDriver(ctx context.Context, conn connection.Connection) (NoSQLDriver, error)
	CreateKVDriver(ctx context.Context, conn connection.Connection) (KVDriver, error)
}
