//go:build e2e

package e2eapi_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcmongo "github.com/testcontainers/testcontainers-go/modules/mongodb"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/JLugagne/forscadb/internal/dataforge/app"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound/memory"
	"github.com/JLugagne/forscadb/internal/domain"
)

// Shared connection details populated by TestMain.
var (
	mongoHost string
	mongoPort int

	redisHost string
	redisPort int

	// MySQL shared state
	mysqlConnID  string
	mysqlManager *app.ConnectionManager
	mysqlService *app.SQLService

	// PostgreSQL shared state: raw connection config used by postgres_test.go
	// to create fresh per-test connections through the app layer.
	postgresConnConfig connection.Connection
)

// TestMain starts one container per backend that is shared across all tests in
// this package, then tears everything down after all tests have run.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// --- MongoDB ---
	mongoContainer, err := tcmongo.Run(ctx, "mongo:7")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start MongoDB container: %v\n", err)
		os.Exit(1)
	}
	mongoMappedPort, err := mongoContainer.MappedPort(ctx, "27017")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get MongoDB mapped port: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}
	mongoHostIP, err := mongoContainer.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get MongoDB host: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}
	mongoHost = mongoHostIP
	mongoPort = mongoMappedPort.Int()

	// --- Redis ---
	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start Redis container: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}
	redisMappedPort, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get Redis mapped port: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}
	redisHostIP, err := redisContainer.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get Redis host: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}
	redisHost = redisHostIP
	redisPort = redisMappedPort.Int()

	// --- MySQL ---
	mysqlContainer, err := tcmysql.Run(ctx,
		"mysql:8",
		tcmysql.WithDatabase("testdb"),
		tcmysql.WithUsername("root"),
		tcmysql.WithPassword("rootpass"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start MySQL container: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}
	mysqlHostIP, err := mysqlContainer.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get MySQL host: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		mysqlContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}
	mysqlMappedPort, err := mysqlContainer.MappedPort(ctx, "3306")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get MySQL port: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		mysqlContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}

	connStore := memory.NewConnStore()
	historyStore := memory.NewQueryHistory()
	factory := outbound.NewFactory()

	mysqlManager = app.NewConnectionManager(connStore, factory)
	mysqlService = app.NewSQLService(mysqlManager, historyStore)

	connSpec := connection.Connection{
		Name:     "test-mysql",
		Engine:   domain.EngineMySQL,
		Category: domain.CategorySQL,
		Host:     mysqlHostIP,
		Port:     mysqlMappedPort.Int(),
		User:     "root",
		Password: "rootpass",
		Database: "testdb",
	}
	createdConn, err := mysqlManager.Create(ctx, connSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create MySQL connection: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		mysqlContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}
	mysqlConnID = createdConn.ID

	if err := mysqlManager.Connect(ctx, mysqlConnID); err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to MySQL: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		mysqlContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}

	if _, err := mysqlService.Execute(ctx, mysqlConnID, mysqlSeedSQL); err != nil {
		fmt.Fprintf(os.Stderr, "failed to seed MySQL: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		mysqlContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}

	// --- PostgreSQL ---
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		testcontainers.WithEnv(map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start PostgreSQL container: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		mysqlContainer.Terminate(ctx) //nolint:errcheck
		os.Exit(1)
	}
	pgHostIP, err := pgContainer.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get PostgreSQL host: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		mysqlContainer.Terminate(ctx) //nolint:errcheck
		pgContainer.Terminate(ctx)    //nolint:errcheck
		os.Exit(1)
	}
	pgMappedPort, err := pgContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get PostgreSQL port: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		mysqlContainer.Terminate(ctx) //nolint:errcheck
		pgContainer.Terminate(ctx)    //nolint:errcheck
		os.Exit(1)
	}

	postgresConnConfig = connection.Connection{
		Name:     "e2e-postgres",
		Engine:   domain.EnginePostgreSQL,
		Category: domain.CategorySQL,
		Host:     pgHostIP,
		Port:     pgMappedPort.Int(),
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
	}

	// Seed the PostgreSQL database once.
	pgFactory := outbound.NewFactory()
	pgDriver, err := pgFactory.CreateSQLDriver(ctx, postgresConnConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create PostgreSQL driver for seeding: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		mysqlContainer.Terminate(ctx) //nolint:errcheck
		pgContainer.Terminate(ctx)    //nolint:errcheck
		os.Exit(1)
	}
	if _, err := pgDriver.ExecuteQuery(ctx, postgresSeedSQL); err != nil {
		pgDriver.Close()               //nolint:errcheck
		fmt.Fprintf(os.Stderr, "failed to seed PostgreSQL: %v\n", err)
		mongoContainer.Terminate(ctx) //nolint:errcheck
		redisContainer.Terminate(ctx) //nolint:errcheck
		mysqlContainer.Terminate(ctx) //nolint:errcheck
		pgContainer.Terminate(ctx)    //nolint:errcheck
		os.Exit(1)
	}
	pgDriver.Close() //nolint:errcheck

	code := m.Run()

	mysqlManager.Disconnect(ctx, mysqlConnID) //nolint:errcheck
	mongoContainer.Terminate(ctx)             //nolint:errcheck
	redisContainer.Terminate(ctx)             //nolint:errcheck
	mysqlContainer.Terminate(ctx)             //nolint:errcheck
	pgContainer.Terminate(ctx)                //nolint:errcheck
	os.Exit(code)
}
