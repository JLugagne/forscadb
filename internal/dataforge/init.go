package dataforge

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/JLugagne/forscadb/internal/dataforge/app"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/kv"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/nosql"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlintrospect"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlquery"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound/filecache"
)

type Services struct {
	ConnCommands  connection.Commands
	ConnQueries   connection.Queries
	SQLIntrospect sqlintrospect.Queries
	SQLCommands   sqlquery.Commands
	SQLQueries    sqlquery.Queries
	NoSQLCommands nosql.Commands
	NoSQLQueries  nosql.Queries
	KVCommands    kv.Commands
	KVQueries     kv.Queries
}

func cacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".cache", "forscadb")
}

func Init() Services {
	dir := cacheDir()

	connStore, err := filecache.NewConnStore(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to init connection store: %v, falling back to in-memory\n", err)
		panic(err)
	}

	queryHistory, err := filecache.NewQueryHistory(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to init query history: %v, falling back to in-memory\n", err)
		panic(err)
	}

	factory := outbound.NewFactory()

	connManager := app.NewConnectionManager(connStore, factory)
	sqlService := app.NewSQLService(connManager, queryHistory)
	nosqlService := app.NewNoSQLService(connManager)
	kvService := app.NewKVService(connManager)

	return Services{
		ConnCommands:  connManager,
		ConnQueries:   connManager,
		SQLIntrospect: sqlService,
		SQLCommands:   sqlService,
		SQLQueries:    sqlService,
		NoSQLCommands: nosqlService,
		NoSQLQueries:  nosqlService,
		KVCommands:    kvService,
		KVQueries:     kvService,
	}
}
