package outbound

import (
	"context"
	"database/sql"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JLugagne/forscadb/internal/dataforge/app"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	mongodriver "github.com/JLugagne/forscadb/internal/dataforge/outbound/mongo"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound/mysql"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound/pg"
	redisdriver "github.com/JLugagne/forscadb/internal/dataforge/outbound/redis"
	"github.com/JLugagne/forscadb/internal/domain"
)

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) CreateSQLDriver(ctx context.Context, conn connection.Connection) (app.SQLDriver, error) {
	switch conn.Engine {
	case domain.EnginePostgreSQL, domain.EngineCockroachDB:
		dsn := buildPostgresDSN(conn)
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return nil, fmt.Errorf("outbound: CreateSQLDriver: %s: %w", conn.Engine, err)
		}
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return nil, fmt.Errorf("outbound: CreateSQLDriver: %s ping: %w", conn.Engine, err)
		}
		return pg.NewPostgresDriver(pool), nil

	case domain.EngineMySQL, domain.EngineMariaDB:
		dsn := buildMySQLDSN(conn)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, fmt.Errorf("outbound: CreateSQLDriver: %s: %w", conn.Engine, err)
		}
		if err := db.PingContext(ctx); err != nil {
			db.Close()
			return nil, fmt.Errorf("outbound: CreateSQLDriver: %s ping: %w", conn.Engine, err)
		}
		return mysql.NewMySQLDriver(db), nil

	case domain.EngineSQLServer:
		return nil, fmt.Errorf("outbound: CreateSQLDriver: SQL Server support coming soon")

	case domain.EngineSQLite:
		return nil, fmt.Errorf("outbound: CreateSQLDriver: SQLite support coming soon")

	default:
		return nil, fmt.Errorf("outbound: CreateSQLDriver: unsupported engine: %s", conn.Engine)
	}
}

func (f *Factory) CreateNoSQLDriver(ctx context.Context, conn connection.Connection) (app.NoSQLDriver, error) {
	switch conn.Engine {
	case domain.EngineMongoDB, domain.EngineDocumentDB:
		uri := buildMongoURI(conn)
		client, err := mongo.Connect(options.Client().ApplyURI(uri))
		if err != nil {
			return nil, fmt.Errorf("outbound: CreateNoSQLDriver: mongo connect: %w", err)
		}
		if err := client.Ping(ctx, nil); err != nil {
			client.Disconnect(ctx)
			return nil, fmt.Errorf("outbound: CreateNoSQLDriver: mongo ping: %w", err)
		}
		return mongodriver.NewMongoDriver(client, conn.Database), nil

	default:
		return nil, fmt.Errorf("outbound: CreateNoSQLDriver: unsupported engine: %s", conn.Engine)
	}
}

func (f *Factory) CreateKVDriver(ctx context.Context, conn connection.Connection) (app.KVDriver, error) {
	switch conn.Engine {
	case domain.EngineRedis, domain.EngineValkey, domain.EngineKeyDB, domain.EngineDragonfly:
		opts := buildRedisOptions(conn)
		client := goredis.NewClient(opts)
		if err := client.Ping(ctx).Err(); err != nil {
			client.Close()
			return nil, fmt.Errorf("outbound: CreateKVDriver: %s ping: %w", conn.Engine, err)
		}
		return redisdriver.NewRedisDriver(client), nil

	case domain.EngineMemcached:
		return nil, fmt.Errorf("outbound: CreateKVDriver: Memcached support coming soon")

	default:
		return nil, fmt.Errorf("outbound: CreateKVDriver: unsupported engine: %s", conn.Engine)
	}
}

func buildPostgresDSN(conn connection.Connection) string {
	host := conn.Host
	if host == "" {
		host = "localhost"
	}
	port := conn.Port
	if port == 0 {
		port = 5432
	}
	sslMode := conn.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		host, port, conn.User, conn.Database, sslMode)
	if conn.Password != "" {
		dsn += fmt.Sprintf(" password=%s", conn.Password)
	}
	return dsn
}

func buildMySQLDSN(conn connection.Connection) string {
	host := conn.Host
	if host == "" {
		host = "localhost"
	}
	port := conn.Port
	if port == 0 {
		port = 3306
	}
	auth := conn.User
	if conn.Password != "" {
		auth = fmt.Sprintf("%s:%s", conn.User, conn.Password)
	}
	dsn := fmt.Sprintf("%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
		auth, host, port, conn.Database)
	if conn.SSLMode != "" && conn.SSLMode != "disable" {
		dsn += "&tls=" + conn.SSLMode
	}
	return dsn
}

func buildMongoURI(conn connection.Connection) string {
	host := conn.Host
	if host == "" {
		host = "localhost"
	}
	port := conn.Port
	if port == 0 {
		port = 27017
	}
	var uri string
	if conn.User != "" && conn.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
			conn.User, conn.Password, host, port, conn.Database)
	} else if conn.User != "" {
		uri = fmt.Sprintf("mongodb://%s@%s:%d/%s",
			conn.User, host, port, conn.Database)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%d/%s", host, port, conn.Database)
	}
	if conn.SSLMode == "require" || conn.SSLMode == "true" {
		uri += "?tls=true"
	}
	return uri
}

func buildRedisOptions(conn connection.Connection) *goredis.Options {
	host := conn.Host
	if host == "" {
		host = "localhost"
	}
	port := conn.Port
	if port == 0 {
		port = 6379
	}
	opts := &goredis.Options{
		Addr: fmt.Sprintf("%s:%d", host, port),
		DB:   0,
	}
	if conn.Password != "" {
		opts.Password = conn.Password
	}
	if conn.User != "" {
		opts.Username = conn.User
	}
	return opts
}
