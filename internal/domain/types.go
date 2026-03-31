package domain

type DatabaseEngine string

const (
	// SQL
	EnginePostgreSQL  DatabaseEngine = "postgresql"
	EngineMySQL       DatabaseEngine = "mysql"
	EngineMariaDB     DatabaseEngine = "mariadb"
	EngineSQLite      DatabaseEngine = "sqlite"
	EngineCockroachDB DatabaseEngine = "cockroachdb"
	EngineSQLServer   DatabaseEngine = "sqlserver"
	// NoSQL
	EngineMongoDB    DatabaseEngine = "mongodb"
	EngineDocumentDB DatabaseEngine = "documentdb"
	// KV
	EngineRedis     DatabaseEngine = "redis"
	EngineValkey    DatabaseEngine = "valkey"
	EngineKeyDB     DatabaseEngine = "keydb"
	EngineDragonfly DatabaseEngine = "dragonfly"
	EngineMemcached DatabaseEngine = "memcached"
)

type DatabaseCategory string

const (
	CategorySQL   DatabaseCategory = "sql"
	CategoryNoSQL DatabaseCategory = "nosql"
	CategoryKV    DatabaseCategory = "kv"
)

type ConnectionStatus string

const (
	StatusConnected    ConnectionStatus = "connected"
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusError        ConnectionStatus = "error"
)
