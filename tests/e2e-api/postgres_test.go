//go:build e2e

package e2eapi_test

import (
	"context"
	"strings"
	"testing"

	"github.com/JLugagne/forscadb/internal/dataforge/app"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound"
	"github.com/JLugagne/forscadb/internal/dataforge/outbound/memory"
	"github.com/JLugagne/forscadb/internal/domain"
)

// postgresSeedSQL is the DDL + DML executed once in TestMain (main_test.go) to
// populate the shared PostgreSQL test database.
const postgresSeedSQL = `
CREATE SCHEMA IF NOT EXISTS public;

CREATE TYPE public.order_status AS ENUM ('pending', 'processing', 'shipped', 'delivered', 'cancelled');

CREATE TABLE public.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(100) NOT NULL UNIQUE,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE public.orders (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES public.users(id),
    status public.order_status NOT NULL DEFAULT 'pending',
    total_amount NUMERIC(12,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX orders_user_id_idx ON public.orders(user_id);
CREATE INDEX orders_status_idx ON public.orders(status);

CREATE SEQUENCE public.custom_seq START 100 INCREMENT 5;

CREATE OR REPLACE VIEW public.active_users AS
SELECT id, email, username, role FROM public.users WHERE is_active = true;

CREATE OR REPLACE FUNCTION public.update_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END; $$;

CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON public.users
FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

INSERT INTO public.users (email, username, role) VALUES
('alice@test.com', 'alice', 'admin'),
('bob@test.com', 'bob', 'user'),
('carol@test.com', 'carol', 'user');

INSERT INTO public.orders (user_id, status, total_amount)
SELECT u.id, 'pending'::public.order_status, 99.99 FROM public.users u WHERE u.username = 'alice'
UNION ALL
SELECT u.id, 'shipped'::public.order_status, 149.50 FROM public.users u WHERE u.username = 'bob';
`

// newPGStack creates a fresh ConnectionManager + SQLService backed by in-memory
// stores and the real outbound factory. Each test gets its own isolated stack.
func newPGStack() (*app.ConnectionManager, *app.SQLService) {
	store := memory.NewConnStore()
	history := memory.NewQueryHistory()
	factory := outbound.NewFactory()
	mgr := app.NewConnectionManager(store, factory)
	svc := app.NewSQLService(mgr, history)
	return mgr, svc
}

// pgConnect creates and connects a PostgreSQL connection through the app layer.
// It returns the connection ID; callers should defer mgr.Disconnect.
func pgConnect(t *testing.T, mgr *app.ConnectionManager) string {
	t.Helper()
	ctx := context.Background()
	conn, err := mgr.Create(ctx, postgresConnConfig)
	if err != nil {
		t.Fatalf("Create connection: %v", err)
	}
	if err := mgr.Connect(ctx, conn.ID); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return conn.ID
}

// pgHas reports whether s is in slice.
func pgHas(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// ---- tests ------------------------------------------------------------------

func TestPostgresConnection(t *testing.T) {
	ctx := context.Background()
	mgr, _ := newPGStack()

	// Create connection — should be disconnected initially.
	conn, err := mgr.Create(ctx, postgresConnConfig)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if conn.ID == "" {
		t.Error("expected non-empty connection ID")
	}
	if conn.Status != domain.StatusDisconnected {
		t.Errorf("initial status: expected %q, got %q", domain.StatusDisconnected, conn.Status)
	}

	// Connect.
	if err := mgr.Connect(ctx, conn.ID); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Verify status is connected.
	got, err := mgr.Get(ctx, conn.ID)
	if err != nil {
		t.Fatalf("Get after connect: %v", err)
	}
	if got.Status != domain.StatusConnected {
		t.Errorf("after connect: expected status %q, got %q", domain.StatusConnected, got.Status)
	}

	// List connections — ours must be present.
	list, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, c := range list {
		if c.ID == conn.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("connection %s not found in list", conn.ID)
	}

	// Disconnect.
	if err := mgr.Disconnect(ctx, conn.ID); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	// Verify status is disconnected.
	got, err = mgr.Get(ctx, conn.ID)
	if err != nil {
		t.Fatalf("Get after disconnect: %v", err)
	}
	if got.Status != domain.StatusDisconnected {
		t.Errorf("after disconnect: expected status %q, got %q", domain.StatusDisconnected, got.Status)
	}
}

func TestPostgresTestConnection(t *testing.T) {
	ctx := context.Background()
	mgr, _ := newPGStack()

	t.Run("valid config returns no error", func(t *testing.T) {
		if err := mgr.TestConnection(ctx, postgresConnConfig); err != nil {
			t.Errorf("unexpected error with valid config: %v", err)
		}
	})

	t.Run("wrong port returns error", func(t *testing.T) {
		bad := postgresConnConfig
		bad.Port = 9 // nothing should be listening here
		if err := mgr.TestConnection(ctx, bad); err == nil {
			t.Error("expected error with wrong port, got nil")
		}
	})

	t.Run("wrong password returns error", func(t *testing.T) {
		bad := postgresConnConfig
		bad.Password = "definitelywrongpassword"
		if err := mgr.TestConnection(ctx, bad); err == nil {
			t.Error("expected error with wrong password, got nil")
		}
	})
}

func TestPostgresTables(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	tables, err := svc.GetTables(ctx, connID)
	if err != nil {
		t.Fatalf("GetTables: %v", err)
	}

	byName := make(map[string]int, len(tables))
	for i, tbl := range tables {
		byName[tbl.Name] = i
	}

	t.Run("users table exists with correct structure", func(t *testing.T) {
		idx, ok := byName["users"]
		if !ok {
			t.Fatal("users table not found")
		}
		tbl := tables[idx]

		if tbl.RowCount <= 0 {
			t.Errorf("users row count: expected > 0, got %d", tbl.RowCount)
		}

		colByName := make(map[string]int, len(tbl.Columns))
		for i, col := range tbl.Columns {
			colByName[col.Name] = i
		}

		// id: uuid, primary key
		if i, ok := colByName["id"]; ok {
			col := tbl.Columns[i]
			if !col.PrimaryKey {
				t.Error("users.id: expected PrimaryKey=true")
			}
			if !strings.Contains(strings.ToLower(col.Type), "uuid") {
				t.Errorf("users.id type: expected uuid, got %q", col.Type)
			}
		} else {
			t.Error("users.id column not found")
		}

		// email: not null
		if i, ok := colByName["email"]; ok {
			if tbl.Columns[i].Nullable {
				t.Error("users.email: expected Nullable=false")
			}
		} else {
			t.Error("users.email column not found")
		}

		// username must exist
		if _, ok := colByName["username"]; !ok {
			t.Error("users.username column not found")
		}

		// is_active: boolean
		if i, ok := colByName["is_active"]; ok {
			col := tbl.Columns[i]
			if !strings.Contains(strings.ToLower(col.Type), "boolean") {
				t.Errorf("users.is_active type: expected boolean, got %q", col.Type)
			}
		} else {
			t.Error("users.is_active column not found")
		}

		// users_pkey index must exist
		idxNames := make([]string, 0, len(tbl.Indexes))
		for _, ix := range tbl.Indexes {
			idxNames = append(idxNames, ix.Name)
		}
		if !pgHas(idxNames, "users_pkey") {
			t.Errorf("users_pkey index not found; got: %v", idxNames)
		}
	})

	t.Run("orders table exists with correct structure", func(t *testing.T) {
		idx, ok := byName["orders"]
		if !ok {
			t.Fatal("orders table not found")
		}
		tbl := tables[idx]

		if tbl.RowCount <= 0 {
			t.Errorf("orders row count: expected > 0, got %d", tbl.RowCount)
		}

		// orders_user_id_idx and orders_status_idx must exist
		idxNames := make([]string, 0, len(tbl.Indexes))
		for _, ix := range tbl.Indexes {
			idxNames = append(idxNames, ix.Name)
		}
		if !pgHas(idxNames, "orders_user_id_idx") {
			t.Errorf("orders_user_id_idx not found; got: %v", idxNames)
		}
		if !pgHas(idxNames, "orders_status_idx") {
			t.Errorf("orders_status_idx not found; got: %v", idxNames)
		}

		// user_id foreign key → users.id
		colByName := make(map[string]int, len(tbl.Columns))
		for i, col := range tbl.Columns {
			colByName[col.Name] = i
		}
		if i, ok := colByName["user_id"]; ok {
			col := tbl.Columns[i]
			if col.ForeignKey == nil {
				t.Error("orders.user_id: expected ForeignKey to be set")
			} else {
				if col.ForeignKey.Table != "users" {
					t.Errorf("orders.user_id FK table: expected %q, got %q", "users", col.ForeignKey.Table)
				}
				if col.ForeignKey.Column != "id" {
					t.Errorf("orders.user_id FK column: expected %q, got %q", "id", col.ForeignKey.Column)
				}
			}
		} else {
			t.Error("orders.user_id column not found")
		}
	})
}

func TestPostgresViews(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	views, err := svc.GetViews(ctx, connID)
	if err != nil {
		t.Fatalf("GetViews: %v", err)
	}

	var viewIdx int
	var found bool
	for i, v := range views {
		if v.Name == "active_users" {
			found = true
			viewIdx = i
			break
		}
	}
	if !found {
		names := make([]string, 0, len(views))
		for _, v := range views {
			names = append(names, v.Name)
		}
		t.Fatalf("active_users view not found; got: %v", names)
	}

	view := views[viewIdx]

	t.Run("definition contains is_active", func(t *testing.T) {
		if !strings.Contains(view.Definition, "is_active") {
			t.Errorf("definition does not contain 'is_active'; got: %s", view.Definition)
		}
	})

	t.Run("has expected columns", func(t *testing.T) {
		colNames := make([]string, 0, len(view.Columns))
		for _, col := range view.Columns {
			colNames = append(colNames, col.Name)
		}
		for _, expected := range []string{"id", "email", "username", "role"} {
			if !pgHas(colNames, expected) {
				t.Errorf("column %q not found; got: %v", expected, colNames)
			}
		}
	})

	t.Run("is not materialized", func(t *testing.T) {
		if view.Materialized {
			t.Error("expected Materialized=false")
		}
	})
}

func TestPostgresFunctions(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	functions, err := svc.GetFunctions(ctx, connID)
	if err != nil {
		t.Fatalf("GetFunctions: %v", err)
	}

	var fnIdx int
	var found bool
	for i, fn := range functions {
		if fn.Name == "update_updated_at" {
			found = true
			fnIdx = i
			break
		}
	}
	if !found {
		names := make([]string, 0, len(functions))
		for _, fn := range functions {
			names = append(names, fn.Name)
		}
		t.Fatalf("update_updated_at not found; got: %v", names)
	}

	fn := functions[fnIdx]

	t.Run("language is plpgsql", func(t *testing.T) {
		if fn.Language != "plpgsql" {
			t.Errorf("expected language %q, got %q", "plpgsql", fn.Language)
		}
	})

	t.Run("return type contains trigger", func(t *testing.T) {
		if !strings.Contains(strings.ToLower(fn.ReturnType), "trigger") {
			t.Errorf("expected ReturnType to contain 'trigger', got %q", fn.ReturnType)
		}
	})

	t.Run("schema is public", func(t *testing.T) {
		if fn.Schema != "public" {
			t.Errorf("expected schema %q, got %q", "public", fn.Schema)
		}
	})
}

func TestPostgresTriggers(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	triggers, err := svc.GetTriggers(ctx, connID)
	if err != nil {
		t.Fatalf("GetTriggers: %v", err)
	}

	var trigIdx int
	var found bool
	for i, trig := range triggers {
		if trig.Name == "trg_users_updated_at" {
			found = true
			trigIdx = i
			break
		}
	}
	if !found {
		names := make([]string, 0, len(triggers))
		for _, trig := range triggers {
			names = append(names, trig.Name)
		}
		t.Fatalf("trg_users_updated_at not found; got: %v", names)
	}

	trig := triggers[trigIdx]

	t.Run("timing is BEFORE", func(t *testing.T) {
		if !strings.EqualFold(trig.Timing, "BEFORE") {
			t.Errorf("expected timing BEFORE, got %q", trig.Timing)
		}
	})

	t.Run("event is UPDATE", func(t *testing.T) {
		if !strings.EqualFold(trig.Event, "UPDATE") {
			t.Errorf("expected event UPDATE, got %q", trig.Event)
		}
	})

	t.Run("enabled is true", func(t *testing.T) {
		if !trig.Enabled {
			t.Error("expected trigger to be enabled")
		}
	})

	t.Run("table is users", func(t *testing.T) {
		if trig.Table != "users" {
			t.Errorf("expected table %q, got %q", "users", trig.Table)
		}
	})
}

func TestPostgresSequences(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	sequences, err := svc.GetSequences(ctx, connID)
	if err != nil {
		t.Fatalf("GetSequences: %v", err)
	}

	var seqIdx int
	var found bool
	for i, seq := range sequences {
		if seq.Name == "custom_seq" {
			found = true
			seqIdx = i
			break
		}
	}
	if !found {
		names := make([]string, 0, len(sequences))
		for _, seq := range sequences {
			names = append(names, seq.Name)
		}
		t.Fatalf("custom_seq not found; got: %v", names)
	}

	seq := sequences[seqIdx]

	t.Run("start value is 100", func(t *testing.T) {
		if seq.StartValue != 100 {
			t.Errorf("expected StartValue=100, got %d", seq.StartValue)
		}
	})

	t.Run("increment is 5", func(t *testing.T) {
		if seq.Increment != 5 {
			t.Errorf("expected Increment=5, got %d", seq.Increment)
		}
	})

	t.Run("schema is public", func(t *testing.T) {
		if seq.Schema != "public" {
			t.Errorf("expected schema %q, got %q", "public", seq.Schema)
		}
	})
}

func TestPostgresEnums(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	enums, err := svc.GetEnums(ctx, connID)
	if err != nil {
		t.Fatalf("GetEnums: %v", err)
	}

	var enumIdx int
	var found bool
	for i, e := range enums {
		if e.Name == "order_status" {
			found = true
			enumIdx = i
			break
		}
	}
	if !found {
		names := make([]string, 0, len(enums))
		for _, e := range enums {
			names = append(names, e.Name)
		}
		t.Fatalf("order_status not found; got: %v", names)
	}

	enum := enums[enumIdx]

	t.Run("contains all expected values", func(t *testing.T) {
		for _, expected := range []string{"pending", "processing", "shipped", "delivered", "cancelled"} {
			if !pgHas(enum.Values, expected) {
				t.Errorf("value %q not found; got: %v", expected, enum.Values)
			}
		}
	})

	t.Run("schema is public", func(t *testing.T) {
		if enum.Schema != "public" {
			t.Errorf("expected schema %q, got %q", "public", enum.Schema)
		}
	})
}

func TestPostgresExecuteQuery(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	t.Run("SELECT returns columns rows and executionTime", func(t *testing.T) {
		result, err := svc.Execute(ctx, connID, "SELECT id, email, username FROM public.users ORDER BY username")
		if err != nil {
			t.Fatalf("Execute SELECT: %v", err)
		}
		if len(result.Columns) == 0 {
			t.Error("expected non-empty Columns")
		}
		if result.RowCount == 0 {
			t.Error("expected RowCount > 0")
		}
		if len(result.Rows) == 0 {
			t.Error("expected non-empty Rows")
		}
		if result.ExecutionTime <= 0 {
			t.Error("expected ExecutionTime > 0")
		}
		for _, col := range []string{"id", "email", "username"} {
			if !pgHas(result.Columns, col) {
				t.Errorf("column %q not in result; got: %v", col, result.Columns)
			}
		}
		// 3 seed users: alice, bob, carol
		if result.RowCount != 3 {
			t.Errorf("expected 3 rows, got %d", result.RowCount)
		}
	})

	t.Run("INSERT returns affected rows", func(t *testing.T) {
		result, err := svc.Execute(ctx, connID,
			"INSERT INTO public.users (email, username, role) VALUES ('dave@test.com', 'dave', 'user')")
		if err != nil {
			t.Fatalf("Execute INSERT: %v", err)
		}
		if result.AffectedRows == nil {
			t.Fatal("expected AffectedRows to be set")
		}
		if *result.AffectedRows != 1 {
			t.Errorf("expected 1 affected row, got %d", *result.AffectedRows)
		}
	})

	t.Run("UPDATE returns affected rows", func(t *testing.T) {
		result, err := svc.Execute(ctx, connID,
			"UPDATE public.users SET role = 'moderator' WHERE username = 'dave'")
		if err != nil {
			t.Fatalf("Execute UPDATE: %v", err)
		}
		if result.AffectedRows == nil {
			t.Fatal("expected AffectedRows to be set")
		}
		if *result.AffectedRows != 1 {
			t.Errorf("expected 1 affected row, got %d", *result.AffectedRows)
		}
	})

	t.Run("query on nonexistent table returns error", func(t *testing.T) {
		if _, err := svc.Execute(ctx, connID, "SELECT * FROM nonexistent_table_xyz_abc"); err == nil {
			t.Error("expected error for nonexistent table, got nil")
		}
	})

	t.Run("syntax error returns error", func(t *testing.T) {
		if _, err := svc.Execute(ctx, connID, "THIS IS NOT VALID SQL !!!"); err == nil {
			t.Error("expected error for invalid SQL, got nil")
		}
	})
}

func TestPostgresGetTableData(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	t.Run("returns correct columns for users", func(t *testing.T) {
		result, err := svc.GetTableData(ctx, connID, "public", "users", 100, 0)
		if err != nil {
			t.Fatalf("GetTableData users: %v", err)
		}
		for _, col := range []string{"id", "email", "username", "role", "is_active", "created_at", "updated_at"} {
			if !pgHas(result.Columns, col) {
				t.Errorf("column %q not found; got: %v", col, result.Columns)
			}
		}
		if result.RowCount < 3 {
			t.Errorf("expected at least 3 rows, got %d", result.RowCount)
		}
	})

	t.Run("limit=1 returns exactly 1 row", func(t *testing.T) {
		result, err := svc.GetTableData(ctx, connID, "public", "users", 1, 0)
		if err != nil {
			t.Fatalf("GetTableData limit=1: %v", err)
		}
		if result.RowCount != 1 {
			t.Errorf("expected RowCount=1, got %d", result.RowCount)
		}
		if len(result.Rows) != 1 {
			t.Errorf("expected 1 row in Rows, got %d", len(result.Rows))
		}
	})

	t.Run("offset skips rows", func(t *testing.T) {
		all, err := svc.GetTableData(ctx, connID, "public", "users", 100, 0)
		if err != nil {
			t.Fatalf("GetTableData all: %v", err)
		}
		if all.RowCount < 2 {
			t.Skip("need at least 2 rows to test offset")
		}

		offset1, err := svc.GetTableData(ctx, connID, "public", "users", 100, 1)
		if err != nil {
			t.Fatalf("GetTableData offset=1: %v", err)
		}
		if offset1.RowCount != all.RowCount-1 {
			t.Errorf("expected %d rows with offset=1, got %d", all.RowCount-1, offset1.RowCount)
		}
	})

	t.Run("returns correct columns for orders", func(t *testing.T) {
		result, err := svc.GetTableData(ctx, connID, "public", "orders", 100, 0)
		if err != nil {
			t.Fatalf("GetTableData orders: %v", err)
		}
		for _, col := range []string{"id", "user_id", "status", "total_amount", "created_at"} {
			if !pgHas(result.Columns, col) {
				t.Errorf("column %q not found; got: %v", col, result.Columns)
			}
		}
		if result.RowCount < 2 {
			t.Errorf("expected at least 2 orders, got %d", result.RowCount)
		}
	})
}

func TestPostgresDropTable(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	// Create the temporary table.
	if _, err := svc.Execute(ctx, connID, "CREATE TABLE public.temp_drop_test (id serial PRIMARY KEY)"); err != nil {
		t.Fatalf("CREATE TABLE temp_drop_test: %v", err)
	}

	// Verify the table exists.
	tables, err := svc.GetTables(ctx, connID)
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
	if err := svc.DropTable(ctx, connID, "public", "temp_drop_test"); err != nil {
		t.Fatalf("DropTable: %v", err)
	}

	// Verify the table is gone.
	tables, err = svc.GetTables(ctx, connID)
	if err != nil {
		t.Fatalf("GetTables after drop: %v", err)
	}
	for _, tbl := range tables {
		if tbl.Name == "temp_drop_test" {
			t.Error("temp_drop_test still exists after DropTable")
		}
	}
}

func TestPostgresAddColumn(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	// Create the temporary table.
	if _, err := svc.Execute(ctx, connID, "CREATE TABLE public.temp_col_test (id serial PRIMARY KEY)"); err != nil {
		t.Fatalf("CREATE TABLE temp_col_test: %v", err)
	}
	defer svc.Execute(ctx, connID, "DROP TABLE public.temp_col_test") //nolint:errcheck

	// Add a new column.
	if err := svc.AddColumn(ctx, connID, "public", "temp_col_test", "email", "varchar(255)", false, "'test@test.com'"); err != nil {
		t.Fatalf("AddColumn: %v", err)
	}

	// Retrieve tables and find temp_col_test.
	tables, err := svc.GetTables(ctx, connID)
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
			t.Errorf("expected 2 columns (id + email), got %d", len(tbl.Columns))
		}
		hasEmail := false
		for _, col := range tbl.Columns {
			if col.Name == "email" {
				hasEmail = true
				break
			}
		}
		if !hasEmail {
			t.Error("column 'email' not found after AddColumn")
		}
		break
	}
	if !found {
		t.Fatal("temp_col_test table not found after AddColumn")
	}
}

func TestPostgresRefreshMaterializedView(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	// Create a temporary materialized view.
	if _, err := svc.Execute(ctx, connID, "CREATE MATERIALIZED VIEW public.temp_matview AS SELECT 1 AS n"); err != nil {
		t.Fatalf("CREATE MATERIALIZED VIEW: %v", err)
	}
	defer svc.Execute(ctx, connID, "DROP MATERIALIZED VIEW public.temp_matview") //nolint:errcheck

	// Refresh should succeed without error.
	if err := svc.RefreshMaterializedView(ctx, connID, "public", "temp_matview"); err != nil {
		t.Fatalf("RefreshMaterializedView: %v", err)
	}
}

func TestPostgresQueryHistory(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	type querySpec struct {
		sql     string
		wantErr bool
	}
	specs := []querySpec{
		{"SELECT 1", false},
		{"SELECT email FROM public.users WHERE username = 'alice'", false},
		{"SELECT * FROM nonexistent_table_history_test", true},
		{"SELECT count(*) FROM public.users", false},
	}

	for _, s := range specs {
		_, _ = svc.Execute(ctx, connID, s.sql)
	}

	history, err := svc.GetHistory(ctx, connID)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}

	if len(history) < len(specs) {
		t.Errorf("expected at least %d history entries, got %d", len(specs), len(history))
	}

	t.Run("all entries have required fields", func(t *testing.T) {
		for i, entry := range history {
			if entry.ID == "" {
				t.Errorf("entry[%d]: empty ID", i)
			}
			if entry.ConnectionID != connID {
				t.Errorf("entry[%d]: ConnectionID: expected %q, got %q", i, connID, entry.ConnectionID)
			}
			if entry.Query == "" {
				t.Errorf("entry[%d]: empty Query", i)
			}
			if entry.ExecutedAt == "" {
				t.Errorf("entry[%d]: empty ExecutedAt", i)
			}
			if entry.Status != "success" && entry.Status != "error" {
				t.Errorf("entry[%d]: unexpected Status %q", i, entry.Status)
			}
			if entry.Duration < 0 {
				t.Errorf("entry[%d]: negative Duration %f", i, entry.Duration)
			}
		}
	})

	t.Run("failed query is recorded with error status and error message", func(t *testing.T) {
		found := false
		for _, entry := range history {
			if entry.Status == "error" {
				found = true
				if entry.Error == nil {
					t.Error("error-status entry has nil Error field")
				}
				break
			}
		}
		if !found {
			t.Error("no error entry found despite executing a bad query")
		}
	})

	t.Run("successful queries have success status", func(t *testing.T) {
		successCount := 0
		for _, entry := range history {
			if entry.Status == "success" {
				successCount++
			}
		}
		expectedSuccessCount := 0
		for _, s := range specs {
			if !s.wantErr {
				expectedSuccessCount++
			}
		}
		if successCount < expectedSuccessCount {
			t.Errorf("expected at least %d success entries, got %d", expectedSuccessCount, successCount)
		}
	})

	t.Run("history is scoped to connection", func(t *testing.T) {
		for _, entry := range history {
			if entry.ConnectionID != connID {
				t.Errorf("entry belongs to wrong connection: %q", entry.ConnectionID)
			}
		}
	})
}

func TestPostgresRenameColumn(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	if _, err := svc.Execute(ctx, connID,
		"CREATE TABLE public.temp_rename_test (id serial PRIMARY KEY, old_name varchar(100))"); err != nil {
		t.Fatalf("CREATE TABLE temp_rename_test: %v", err)
	}
	defer svc.Execute(ctx, connID, "DROP TABLE IF EXISTS public.temp_rename_test") //nolint:errcheck

	if err := svc.RenameColumn(ctx, connID, "public", "temp_rename_test", "old_name", "new_name"); err != nil {
		t.Fatalf("RenameColumn: %v", err)
	}

	tables, err := svc.GetTables(ctx, connID)
	if err != nil {
		t.Fatalf("GetTables after RenameColumn: %v", err)
	}

	var tbl *struct{ cols []string }
	for _, tt := range tables {
		if tt.Name != "temp_rename_test" {
			continue
		}
		cols := make([]string, 0, len(tt.Columns))
		for _, c := range tt.Columns {
			cols = append(cols, c.Name)
		}
		tbl = &struct{ cols []string }{cols: cols}
		break
	}
	if tbl == nil {
		t.Fatal("temp_rename_test table not found after RenameColumn")
	}
	if pgHas(tbl.cols, "old_name") {
		t.Error("column 'old_name' still exists after RenameColumn")
	}
	if !pgHas(tbl.cols, "new_name") {
		t.Errorf("column 'new_name' not found after RenameColumn; got: %v", tbl.cols)
	}
}

func TestPostgresAlterColumnType(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	if _, err := svc.Execute(ctx, connID,
		"CREATE TABLE public.temp_type_test (id serial PRIMARY KEY, val varchar(50))"); err != nil {
		t.Fatalf("CREATE TABLE temp_type_test: %v", err)
	}
	defer svc.Execute(ctx, connID, "DROP TABLE IF EXISTS public.temp_type_test") //nolint:errcheck

	if err := svc.AlterColumnType(ctx, connID, "public", "temp_type_test", "val", "text"); err != nil {
		t.Fatalf("AlterColumnType: %v", err)
	}

	tables, err := svc.GetTables(ctx, connID)
	if err != nil {
		t.Fatalf("GetTables after AlterColumnType: %v", err)
	}

	for _, tt := range tables {
		if tt.Name != "temp_type_test" {
			continue
		}
		for _, col := range tt.Columns {
			if col.Name != "val" {
				continue
			}
			if !strings.Contains(strings.ToLower(col.Type), "text") {
				t.Errorf("val column type: expected 'text', got %q", col.Type)
			}
			return
		}
		t.Fatal("column 'val' not found in temp_type_test")
	}
	t.Fatal("temp_type_test table not found after AlterColumnType")
}

func TestPostgresDropColumn(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	if _, err := svc.Execute(ctx, connID,
		"CREATE TABLE public.temp_dropcol_test (id serial PRIMARY KEY, to_remove varchar(100))"); err != nil {
		t.Fatalf("CREATE TABLE temp_dropcol_test: %v", err)
	}
	defer svc.Execute(ctx, connID, "DROP TABLE IF EXISTS public.temp_dropcol_test") //nolint:errcheck

	// Verify 2 columns exist before drop.
	tables, err := svc.GetTables(ctx, connID)
	if err != nil {
		t.Fatalf("GetTables before DropColumn: %v", err)
	}
	for _, tt := range tables {
		if tt.Name == "temp_dropcol_test" {
			if len(tt.Columns) != 2 {
				t.Fatalf("expected 2 columns before DropColumn, got %d", len(tt.Columns))
			}
			break
		}
	}

	if err := svc.DropColumn(ctx, connID, "public", "temp_dropcol_test", "to_remove"); err != nil {
		t.Fatalf("DropColumn: %v", err)
	}

	tables, err = svc.GetTables(ctx, connID)
	if err != nil {
		t.Fatalf("GetTables after DropColumn: %v", err)
	}
	for _, tt := range tables {
		if tt.Name != "temp_dropcol_test" {
			continue
		}
		if len(tt.Columns) != 1 {
			t.Errorf("expected 1 column after DropColumn, got %d", len(tt.Columns))
		}
		if tt.Columns[0].Name != "id" {
			t.Errorf("expected remaining column to be 'id', got %q", tt.Columns[0].Name)
		}
		return
	}
	t.Fatal("temp_dropcol_test table not found after DropColumn")
}

func TestPostgresSetColumnNullable(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	if _, err := svc.Execute(ctx, connID,
		"CREATE TABLE public.temp_null_test (id serial PRIMARY KEY, val varchar(100) NOT NULL)"); err != nil {
		t.Fatalf("CREATE TABLE temp_null_test: %v", err)
	}
	defer svc.Execute(ctx, connID, "DROP TABLE IF EXISTS public.temp_null_test") //nolint:errcheck

	// Helper to find val column nullable state.
	findValNullable := func(expectTable bool) (bool, bool) {
		tables, err := svc.GetTables(ctx, connID)
		if err != nil {
			t.Fatalf("GetTables: %v", err)
		}
		for _, tt := range tables {
			if tt.Name != "temp_null_test" {
				continue
			}
			for _, col := range tt.Columns {
				if col.Name == "val" {
					return col.Nullable, true
				}
			}
			t.Fatal("column 'val' not found in temp_null_test")
		}
		if expectTable {
			t.Fatal("temp_null_test table not found")
		}
		return false, false
	}

	// Initially val must be NOT NULL.
	nullable, found := findValNullable(true)
	if !found {
		t.Fatal("val column not found before SetColumnNullable")
	}
	if nullable {
		t.Error("val: expected Nullable=false initially (NOT NULL constraint)")
	}

	// Make nullable.
	if err := svc.SetColumnNullable(ctx, connID, "public", "temp_null_test", "val", true); err != nil {
		t.Fatalf("SetColumnNullable(true): %v", err)
	}
	nullable, _ = findValNullable(true)
	if !nullable {
		t.Error("val: expected Nullable=true after SetColumnNullable(true)")
	}

	// Set back to NOT NULL.
	if err := svc.SetColumnNullable(ctx, connID, "public", "temp_null_test", "val", false); err != nil {
		t.Fatalf("SetColumnNullable(false): %v", err)
	}
	nullable, _ = findValNullable(true)
	if nullable {
		t.Error("val: expected Nullable=false after SetColumnNullable(false)")
	}
}

func TestPostgresSetColumnDefault(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	if _, err := svc.Execute(ctx, connID,
		"CREATE TABLE public.temp_default_test (id serial PRIMARY KEY, val varchar(100))"); err != nil {
		t.Fatalf("CREATE TABLE temp_default_test: %v", err)
	}
	defer svc.Execute(ctx, connID, "DROP TABLE IF EXISTS public.temp_default_test") //nolint:errcheck

	// Helper to find the DefaultValue of the val column.
	findValDefault := func() *string {
		tables, err := svc.GetTables(ctx, connID)
		if err != nil {
			t.Fatalf("GetTables: %v", err)
		}
		for _, tt := range tables {
			if tt.Name != "temp_default_test" {
				continue
			}
			for _, col := range tt.Columns {
				if col.Name == "val" {
					return col.DefaultValue
				}
			}
			t.Fatal("column 'val' not found in temp_default_test")
		}
		t.Fatal("temp_default_test table not found")
		return nil
	}

	// Initially no default.
	if def := findValDefault(); def != nil {
		t.Errorf("val: expected no default initially, got %q", *def)
	}

	// Set a default.
	if err := svc.SetColumnDefault(ctx, connID, "public", "temp_default_test", "val", "'hello'"); err != nil {
		t.Fatalf("SetColumnDefault('hello'): %v", err)
	}
	def := findValDefault()
	if def == nil {
		t.Fatal("val: expected default to be set after SetColumnDefault, got nil")
	}
	if !strings.Contains(*def, "hello") {
		t.Errorf("val: expected default to contain 'hello', got %q", *def)
	}

	// Drop the default.
	if err := svc.SetColumnDefault(ctx, connID, "public", "temp_default_test", "val", ""); err != nil {
		t.Fatalf("SetColumnDefault(''): %v", err)
	}
	if def := findValDefault(); def != nil {
		t.Errorf("val: expected no default after dropping it, got %q", *def)
	}
}

func TestPostgresExplainQuery(t *testing.T) {
	ctx := context.Background()
	mgr, svc := newPGStack()
	connID := pgConnect(t, mgr)
	defer mgr.Disconnect(ctx, connID) //nolint:errcheck

	t.Run("explain without analyze", func(t *testing.T) {
		result, err := svc.ExplainQuery(ctx, connID, "SELECT * FROM public.users", false)
		if err != nil {
			t.Fatalf("ExplainQuery: %v", err)
		}
		if result.Plan == "" {
			t.Error("expected non-empty plan")
		}
		if result.Format != "text" {
			t.Errorf("expected format 'text', got %q", result.Format)
		}
		if len(result.PlanRows) == 0 {
			t.Error("expected at least one plan row")
		}
		// Plan should mention Seq Scan since there's no WHERE clause
		if !strings.Contains(result.Plan, "Seq Scan") {
			t.Errorf("expected plan to contain 'Seq Scan', got:\n%s", result.Plan)
		}
		// Should NOT contain "actual time" since we didn't analyze
		if strings.Contains(result.Plan, "actual time") {
			t.Error("expected no 'actual time' without analyze")
		}
	})

	t.Run("explain with analyze", func(t *testing.T) {
		result, err := svc.ExplainQuery(ctx, connID, "SELECT * FROM public.users", true)
		if err != nil {
			t.Fatalf("ExplainQuery (analyze): %v", err)
		}
		if result.Plan == "" {
			t.Error("expected non-empty plan")
		}
		// ANALYZE should include actual timing
		if !strings.Contains(result.Plan, "actual time") {
			t.Errorf("expected 'actual time' with analyze, got:\n%s", result.Plan)
		}
		// Should have PlanRows with at least one node
		hasNode := false
		for _, row := range result.PlanRows {
			if row.IsNode {
				hasNode = true
				break
			}
		}
		if !hasNode {
			t.Error("expected at least one plan node marked as IsNode")
		}
	})

	t.Run("explain join query", func(t *testing.T) {
		result, err := svc.ExplainQuery(ctx, connID,
			"SELECT u.email, o.total_amount FROM public.users u JOIN public.orders o ON o.user_id = u.id",
			true)
		if err != nil {
			t.Fatalf("ExplainQuery (join): %v", err)
		}
		// Should show join strategy
		planLower := strings.ToLower(result.Plan)
		if !strings.Contains(planLower, "join") && !strings.Contains(planLower, "loop") && !strings.Contains(planLower, "hash") {
			t.Errorf("expected plan to mention a join strategy, got:\n%s", result.Plan)
		}
	})

	t.Run("explain bad query returns error", func(t *testing.T) {
		_, err := svc.ExplainQuery(ctx, connID, "SELECT * FROM nonexistent_table", false)
		if err == nil {
			t.Error("expected error for bad query, got nil")
		}
	})
}
