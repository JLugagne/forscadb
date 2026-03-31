//go:build e2e

package e2eapi_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/JLugagne/forscadb/internal/dataforge/app"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound/memory"
	"github.com/JLugagne/forscadb/internal/domain"
)

// mysqlSeedSQL is the DDL + DML used to populate the shared MySQL test
// database.  It is executed once in TestMain (main_test.go).
const mysqlSeedSQL = `
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(100) NOT NULL UNIQUE,
    role ENUM('admin', 'user', 'viewer') NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE orders (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX orders_user_id_idx ON orders(user_id);
CREATE INDEX orders_status_idx ON orders(status);

CREATE VIEW active_users AS
SELECT id, email, username, role FROM users WHERE is_active = true;

CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users
FOR EACH ROW SET NEW.updated_at = NOW();

INSERT INTO users (email, username, role) VALUES
('alice@test.com', 'alice', 'admin'),
('bob@test.com', 'bob', 'user'),
('carol@test.com', 'carol', 'user');

INSERT INTO orders (user_id, status, total_amount)
VALUES (1, 'pending', 99.99), (2, 'pending', 149.50);
`

// newMySQLStack builds an isolated ConnectionManager + SQLService backed by
// the shared MySQL testcontainer.  Each caller gets its own in-memory stores
// so tests remain independent.
func newMySQLStack(t *testing.T, host string, port int) (*app.ConnectionManager, *app.SQLService, string) {
	t.Helper()
	ctx := context.Background()

	connStore := memory.NewConnStore()
	historyStore := memory.NewQueryHistory()
	factory := outbound.NewFactory()

	manager := app.NewConnectionManager(connStore, factory)
	svc := app.NewSQLService(manager, historyStore)

	connSpec := connection.Connection{
		Name:     t.Name(),
		Engine:   domain.EngineMySQL,
		Category: domain.CategorySQL,
		Host:     host,
		Port:     port,
		User:     "root",
		Password: "rootpass",
		Database: "testdb",
	}
	created, err := manager.Create(ctx, connSpec)
	require.NoError(t, err)

	err = manager.Connect(ctx, created.ID)
	require.NoError(t, err)
	t.Cleanup(func() { manager.Disconnect(ctx, created.ID) }) //nolint:errcheck

	return manager, svc, created.ID
}

func TestMySQLConnection(t *testing.T) {
	ctx := context.Background()

	// Retrieve the existing shared connection's current host/port by re-using
	// the shared manager that already has the container coordinates wired in.
	got, err := mysqlManager.Get(ctx, mysqlConnID)
	require.NoError(t, err)

	// Use the same coordinates to create a fresh, isolated lifecycle test.
	manager, _, connID := newMySQLStack(t, got.Host, got.Port)

	// verify connected after newMySQLStack
	conn, err := manager.Get(ctx, connID)
	require.NoError(t, err)
	if conn.Status != domain.StatusConnected {
		t.Errorf("expected status %q after connect, got %q", domain.StatusConnected, conn.Status)
	}

	// disconnect
	err = manager.Disconnect(ctx, connID)
	require.NoError(t, err)

	conn, err = manager.Get(ctx, connID)
	require.NoError(t, err)
	if conn.Status != domain.StatusDisconnected {
		t.Errorf("expected status %q after disconnect, got %q", domain.StatusDisconnected, conn.Status)
	}
}

func TestMySQLTables(t *testing.T) {
	ctx := context.Background()

	tables, err := mysqlService.GetTables(ctx, mysqlConnID)
	require.NoError(t, err)

	tableByName := make(map[string]int)
	for i, tbl := range tables {
		tableByName[tbl.Name] = i
	}

	if _, ok := tableByName["users"]; !ok {
		t.Error("expected table 'users' to exist")
	}
	if _, ok := tableByName["orders"]; !ok {
		t.Error("expected table 'orders' to exist")
	}

	// verify columns for users table
	for _, tbl := range tables {
		if tbl.Name != "users" {
			continue
		}

		colByName := make(map[string]int)
		for i, col := range tbl.Columns {
			colByName[col.Name] = i
		}

		for _, want := range []string{"id", "email", "role"} {
			if _, ok := colByName[want]; !ok {
				t.Errorf("users table missing column %q", want)
			}
		}

		for _, col := range tbl.Columns {
			switch col.Name {
			case "id":
				if !col.PrimaryKey {
					t.Error("expected 'id' column to be a primary key")
				}
				if col.Type != "int" {
					t.Errorf("expected 'id' column type 'int', got %q", col.Type)
				}
			case "email":
				if col.Type != "varchar" {
					t.Errorf("expected 'email' column type 'varchar', got %q", col.Type)
				}
			case "role":
				if col.Type != "enum" {
					t.Errorf("expected 'role' column type 'enum', got %q", col.Type)
				}
			}
		}

		foundPrimary := false
		for _, idx := range tbl.Indexes {
			if idx.Name == "PRIMARY" {
				foundPrimary = true
			}
		}
		if !foundPrimary {
			t.Error("users table missing PRIMARY index")
		}
	}

	// verify orders table indexes and foreign keys
	for _, tbl := range tables {
		if tbl.Name != "orders" {
			continue
		}

		indexByName := make(map[string]bool)
		for _, idx := range tbl.Indexes {
			indexByName[idx.Name] = true
		}

		if !indexByName["orders_user_id_idx"] {
			t.Error("orders table missing 'orders_user_id_idx' index")
		}
		if !indexByName["orders_status_idx"] {
			t.Error("orders table missing 'orders_status_idx' index")
		}

		for _, col := range tbl.Columns {
			if col.Name != "user_id" {
				continue
			}
			if col.ForeignKey == nil {
				t.Error("orders.user_id should have a foreign key")
			} else {
				if col.ForeignKey.Table != "users" {
					t.Errorf("orders.user_id FK should reference 'users', got %q", col.ForeignKey.Table)
				}
				if col.ForeignKey.Column != "id" {
					t.Errorf("orders.user_id FK column should be 'id', got %q", col.ForeignKey.Column)
				}
			}
		}
	}
}

func TestMySQLViews(t *testing.T) {
	ctx := context.Background()

	views, err := mysqlService.GetViews(ctx, mysqlConnID)
	require.NoError(t, err)

	found := false
	for _, v := range views {
		if v.Name == "active_users" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected view 'active_users' to exist, got %d views", len(views))
	}
}

func TestMySQLTriggers(t *testing.T) {
	ctx := context.Background()

	triggers, err := mysqlService.GetTriggers(ctx, mysqlConnID)
	require.NoError(t, err)

	found := false
	for _, tr := range triggers {
		if tr.Name != "trg_users_updated_at" {
			continue
		}
		found = true
		if tr.Timing != "BEFORE" {
			t.Errorf("expected trigger timing 'BEFORE', got %q", tr.Timing)
		}
		if tr.Event != "UPDATE" {
			t.Errorf("expected trigger event 'UPDATE', got %q", tr.Event)
		}
	}
	if !found {
		t.Errorf("expected trigger 'trg_users_updated_at' to exist, got %d triggers", len(triggers))
	}
}

func TestMySQLEnums(t *testing.T) {
	ctx := context.Background()

	enums, err := mysqlService.GetEnums(ctx, mysqlConnID)
	require.NoError(t, err)

	if len(enums) == 0 {
		t.Fatal("expected at least one enum, got none")
	}

	found := false
	for _, e := range enums {
		if e.Name != "role" {
			continue
		}
		found = true
		valSet := make(map[string]bool, len(e.Values))
		for _, v := range e.Values {
			valSet[v] = true
		}
		for _, want := range []string{"admin", "user", "viewer"} {
			if !valSet[want] {
				t.Errorf("expected enum 'role' to contain value %q", want)
			}
		}
	}
	if !found {
		t.Errorf("expected enum named 'role', got enums: %v", enums)
	}
}

func TestMySQLExecuteQuery(t *testing.T) {
	ctx := context.Background()

	t.Run("SELECT returns columns and rows", func(t *testing.T) {
		result, err := mysqlService.Execute(ctx, mysqlConnID, "SELECT id, email, username FROM users ORDER BY id")
		require.NoError(t, err)

		if len(result.Columns) == 0 {
			t.Error("expected columns in result")
		}
		if result.RowCount < 3 {
			t.Errorf("expected at least 3 rows, got %d", result.RowCount)
		}
		if result.ExecutionTime < 0 {
			t.Error("expected executionTime >= 0")
		}

		colSet := make(map[string]bool, len(result.Columns))
		for _, c := range result.Columns {
			colSet[c] = true
		}
		if !colSet["id"] {
			t.Error("expected column 'id' in result")
		}
		if !colSet["email"] {
			t.Error("expected column 'email' in result")
		}
	})

	t.Run("INSERT returns affectedRows=1", func(t *testing.T) {
		result, err := mysqlService.Execute(ctx, mysqlConnID,
			"INSERT INTO users (email, username, role) VALUES ('dave@test.com', 'dave', 'viewer')")
		require.NoError(t, err)

		if result.AffectedRows == nil {
			t.Fatal("expected AffectedRows to be set for INSERT")
		}
		if *result.AffectedRows != 1 {
			t.Errorf("expected affectedRows=1, got %d", *result.AffectedRows)
		}
	})

	t.Run("bad SQL returns error", func(t *testing.T) {
		_, err := mysqlService.Execute(ctx, mysqlConnID, "SELECT * FROM nonexistent_table_xyz")
		if err == nil {
			t.Error("expected error for invalid SQL, got nil")
		}
	})
}

func TestMySQLGetTableData(t *testing.T) {
	ctx := context.Background()

	t.Run("returns seed data", func(t *testing.T) {
		result, err := mysqlService.GetTableData(ctx, mysqlConnID, "testdb", "users", 10, 0)
		require.NoError(t, err)

		if result.RowCount < 3 {
			t.Errorf("expected at least 3 rows, got %d", result.RowCount)
		}
		if len(result.Columns) == 0 {
			t.Error("expected columns to be populated")
		}
	})

	t.Run("limit and offset", func(t *testing.T) {
		result, err := mysqlService.GetTableData(ctx, mysqlConnID, "testdb", "users", 1, 0)
		require.NoError(t, err)

		if result.RowCount != 1 {
			t.Errorf("expected 1 row with limit=1, got %d", result.RowCount)
		}

		result2, err := mysqlService.GetTableData(ctx, mysqlConnID, "testdb", "users", 1, 1)
		require.NoError(t, err)

		if result2.RowCount != 1 {
			t.Errorf("expected 1 row with limit=1 offset=1, got %d", result2.RowCount)
		}

		if len(result.Rows) > 0 && len(result2.Rows) > 0 {
			id1 := result.Rows[0]["id"]
			id2 := result2.Rows[0]["id"]
			if id1 == id2 {
				t.Errorf("expected different rows at offset 0 vs 1, but got same id: %v", id1)
			}
		}
	})
}

func TestMySQLSequences(t *testing.T) {
	ctx := context.Background()

	seqs, err := mysqlService.GetSequences(ctx, mysqlConnID)
	require.NoError(t, err)

	if len(seqs) != 0 {
		t.Errorf("expected no sequences in MySQL, got %d", len(seqs))
	}
}

func TestMySQLDropTable(t *testing.T) {
	ctx := context.Background()

	// Create the temporary table.
	if _, err := mysqlService.Execute(ctx, mysqlConnID, "CREATE TABLE temp_drop_test (id INT PRIMARY KEY)"); err != nil {
		t.Fatalf("CREATE TABLE temp_drop_test: %v", err)
	}

	// Verify the table exists.
	tables, err := mysqlService.GetTables(ctx, mysqlConnID)
	if err != nil {
		t.Fatalf("GetTables before drop: %v", err)
	}
	found := false
	for _, tbl := range tables {
		if tbl.Name == "temp_drop_test" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("temp_drop_test should exist before DropTable")
	}

	// Drop the table.
	if err := mysqlService.DropTable(ctx, mysqlConnID, "testdb", "temp_drop_test"); err != nil {
		t.Fatalf("DropTable: %v", err)
	}

	// Verify the table is gone.
	tables, err = mysqlService.GetTables(ctx, mysqlConnID)
	if err != nil {
		t.Fatalf("GetTables after drop: %v", err)
	}
	for _, tbl := range tables {
		if tbl.Name == "temp_drop_test" {
			t.Error("temp_drop_test still exists after DropTable")
		}
	}
}

func TestMySQLAddColumn(t *testing.T) {
	ctx := context.Background()

	// Create the temporary table.
	if _, err := mysqlService.Execute(ctx, mysqlConnID, "CREATE TABLE temp_col_test (id INT PRIMARY KEY)"); err != nil {
		t.Fatalf("CREATE TABLE temp_col_test: %v", err)
	}
	defer mysqlService.Execute(ctx, mysqlConnID, "DROP TABLE temp_col_test") //nolint:errcheck

	// Add a new column.
	if err := mysqlService.AddColumn(ctx, mysqlConnID, "testdb", "temp_col_test", "name", "VARCHAR(100)", true, ""); err != nil {
		t.Fatalf("AddColumn: %v", err)
	}

	// Retrieve tables and find temp_col_test.
	tables, err := mysqlService.GetTables(ctx, mysqlConnID)
	if err != nil {
		t.Fatalf("GetTables after AddColumn: %v", err)
	}

	var found bool
	for _, tbl := range tables {
		if tbl.Name != "temp_col_test" {
			continue
		}
		found = true
		if len(tbl.Columns) != 2 {
			t.Errorf("expected 2 columns (id + name), got %d", len(tbl.Columns))
		}
		hasName := false
		for _, col := range tbl.Columns {
			if col.Name == "name" {
				hasName = true
				break
			}
		}
		if !hasName {
			t.Error("column 'name' not found after AddColumn")
		}
		break
	}
	if !found {
		t.Fatal("temp_col_test table not found after AddColumn")
	}
}

func TestMySQLQueryHistory(t *testing.T) {
	ctx := context.Background()

	// use a dedicated isolated stack with its own history store
	conn, err := mysqlManager.Get(ctx, mysqlConnID)
	require.NoError(t, err)

	_, svc, connID := newMySQLStack(t, conn.Host, conn.Port)

	// history should be empty initially
	history, err := svc.GetHistory(ctx, connID)
	require.NoError(t, err)
	if len(history) != 0 {
		t.Errorf("expected empty history initially, got %d entries", len(history))
	}

	// execute two successful queries
	_, err = svc.Execute(ctx, connID, "SELECT 1")
	require.NoError(t, err)

	_, err = svc.Execute(ctx, connID, "SELECT 2")
	require.NoError(t, err)

	// execute one that should fail to verify error entries are also stored
	_, _ = svc.Execute(ctx, connID, "SELECT * FROM nonexistent_xyz")

	history, err = svc.GetHistory(ctx, connID)
	require.NoError(t, err)

	if len(history) < 3 {
		t.Errorf("expected at least 3 history entries, got %d", len(history))
	}

	for _, entry := range history {
		if entry.ConnectionID != connID {
			t.Errorf("history entry connectionID mismatch: got %q, want %q", entry.ConnectionID, connID)
		}
		if entry.Query == "" {
			t.Error("history entry has empty query")
		}
		if entry.ExecutedAt == "" {
			t.Error("history entry has empty executedAt")
		}
		if entry.Status == "" {
			t.Error("history entry has empty status")
		}
	}

	statusSet := make(map[string]bool, len(history))
	for _, e := range history {
		statusSet[e.Status] = true
	}
	if !statusSet["success"] {
		t.Error("expected at least one 'success' history entry")
	}
	if !statusSet["error"] {
		t.Error("expected at least one 'error' history entry")
	}
}
