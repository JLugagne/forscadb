package pg

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlintrospect"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlquery"
)

type Driver struct {
	pool *pgxpool.Pool
}

func NewPostgresDriver(pool *pgxpool.Pool) *Driver {
	return &Driver{pool: pool}
}

func (d *Driver) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

func (d *Driver) Close() error {
	d.pool.Close()
	return nil
}

func (d *Driver) GetTables(ctx context.Context) ([]sqlintrospect.Table, error) {
	tableQuery := `
		SELECT
			t.table_schema,
			t.table_name,
			COALESCE(s.n_live_tup, 0) AS row_count,
			COALESCE(pg_size_pretty(pg_total_relation_size(quote_ident(t.table_schema) || '.' || quote_ident(t.table_name))), '0 bytes') AS size
		FROM information_schema.tables t
		LEFT JOIN pg_stat_user_tables s
			ON s.schemaname = t.table_schema AND s.relname = t.table_name
		WHERE t.table_type = 'BASE TABLE'
			AND t.table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY t.table_schema, t.table_name`

	rows, err := d.pool.Query(ctx, tableQuery)
	if err != nil {
		return nil, fmt.Errorf("postgres: GetTables: %w", err)
	}
	defer rows.Close()

	type tableRow struct {
		schema   string
		name     string
		rowCount int64
		size     string
	}
	var tableRows []tableRow
	for rows.Next() {
		var tr tableRow
		if err := rows.Scan(&tr.schema, &tr.name, &tr.rowCount, &tr.size); err != nil {
			return nil, fmt.Errorf("postgres: GetTables: scan: %w", err)
		}
		tableRows = append(tableRows, tr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: GetTables: rows: %w", err)
	}

	var tables []sqlintrospect.Table
	for _, tr := range tableRows {
		columns, err := d.getColumns(ctx, tr.schema, tr.name)
		if err != nil {
			return nil, err
		}
		indexes, err := d.getIndexes(ctx, tr.schema, tr.name)
		if err != nil {
			return nil, err
		}
		tables = append(tables, sqlintrospect.Table{
			Name:     tr.name,
			Schema:   tr.schema,
			Columns:  columns,
			RowCount: tr.rowCount,
			Size:     tr.size,
			Indexes:  indexes,
		})
	}
	return tables, nil
}

func (d *Driver) getColumns(ctx context.Context, schema, table string) ([]sqlintrospect.Column, error) {
	q := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' AS nullable,
			c.column_default,
			CASE WHEN tc.constraint_type = 'PRIMARY KEY' THEN true ELSE false END AS is_pk
		FROM information_schema.columns c
		LEFT JOIN information_schema.key_column_usage kcu
			ON kcu.table_schema = c.table_schema
			AND kcu.table_name = c.table_name
			AND kcu.column_name = c.column_name
		LEFT JOIN information_schema.table_constraints tc
			ON tc.constraint_name = kcu.constraint_name
			AND tc.constraint_type = 'PRIMARY KEY'
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position`

	rows, err := d.pool.Query(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("postgres: getColumns: %w", err)
	}
	defer rows.Close()

	fkMap, err := d.getForeignKeys(ctx, schema, table)
	if err != nil {
		return nil, err
	}

	var columns []sqlintrospect.Column
	for rows.Next() {
		var col sqlintrospect.Column
		var defaultVal *string
		if err := rows.Scan(&col.Name, &col.Type, &col.Nullable, &defaultVal, &col.PrimaryKey); err != nil {
			return nil, fmt.Errorf("postgres: getColumns: scan: %w", err)
		}
		col.DefaultValue = defaultVal
		if fk, ok := fkMap[col.Name]; ok {
			col.ForeignKey = &fk
		}
		columns = append(columns, col)
	}
	return columns, rows.Err()
}

func (d *Driver) getForeignKeys(ctx context.Context, schema, table string) (map[string]sqlintrospect.ForeignKey, error) {
	q := `
		SELECT
			kcu.column_name,
			ccu.table_name AS foreign_table,
			ccu.column_name AS foreign_column
		FROM information_schema.key_column_usage kcu
		JOIN information_schema.table_constraints tc
			ON tc.constraint_name = kcu.constraint_name
			AND tc.constraint_schema = kcu.constraint_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND kcu.table_schema = $1
			AND kcu.table_name = $2`

	rows, err := d.pool.Query(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("postgres: getForeignKeys: %w", err)
	}
	defer rows.Close()

	result := make(map[string]sqlintrospect.ForeignKey)
	for rows.Next() {
		var col, ftable, fcol string
		if err := rows.Scan(&col, &ftable, &fcol); err != nil {
			return nil, fmt.Errorf("postgres: getForeignKeys: scan: %w", err)
		}
		result[col] = sqlintrospect.ForeignKey{Table: ftable, Column: fcol}
	}
	return result, rows.Err()
}

func (d *Driver) getIndexes(ctx context.Context, schema, table string) ([]sqlintrospect.Index, error) {
	q := `
		SELECT
			i.relname AS index_name,
			ix.indisunique AS is_unique,
			am.amname AS index_type,
			array_agg(a.attname ORDER BY k.n) AS columns
		FROM pg_class t
		JOIN pg_index ix ON ix.indrelid = t.oid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_am am ON am.oid = i.relam
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS k(attnum, n) ON true
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
		WHERE n.nspname = $1 AND t.relname = $2
		GROUP BY i.relname, ix.indisunique, am.amname
		ORDER BY i.relname`

	rows, err := d.pool.Query(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("postgres: getIndexes: %w", err)
	}
	defer rows.Close()

	var indexes []sqlintrospect.Index
	for rows.Next() {
		var idx sqlintrospect.Index
		var cols []string
		if err := rows.Scan(&idx.Name, &idx.Unique, &idx.Type, &cols); err != nil {
			return nil, fmt.Errorf("postgres: getIndexes: scan: %w", err)
		}
		idx.Columns = cols
		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

func (d *Driver) GetViews(ctx context.Context) ([]sqlintrospect.View, error) {
	q := `
		SELECT
			v.table_schema,
			v.table_name,
			v.view_definition,
			false AS materialized
		FROM information_schema.views v
		WHERE v.table_schema NOT IN ('pg_catalog', 'information_schema')
		UNION ALL
		SELECT
			schemaname,
			matviewname,
			definition,
			true
		FROM pg_matviews
		ORDER BY 1, 2`

	rows, err := d.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres: GetViews: %w", err)
	}
	defer rows.Close()

	type viewRow struct {
		schema       string
		name         string
		definition   string
		materialized bool
	}
	var viewRows []viewRow
	for rows.Next() {
		var vr viewRow
		if err := rows.Scan(&vr.schema, &vr.name, &vr.definition, &vr.materialized); err != nil {
			return nil, fmt.Errorf("postgres: GetViews: scan: %w", err)
		}
		viewRows = append(viewRows, vr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: GetViews: rows: %w", err)
	}

	var views []sqlintrospect.View
	for _, vr := range viewRows {
		columns, err := d.getColumns(ctx, vr.schema, vr.name)
		if err != nil {
			columns = []sqlintrospect.Column{}
		}
		views = append(views, sqlintrospect.View{
			Name:         vr.name,
			Schema:       vr.schema,
			Definition:   vr.definition,
			Columns:      columns,
			Materialized: vr.materialized,
		})
	}
	return views, nil
}

func (d *Driver) GetFunctions(ctx context.Context) ([]sqlintrospect.Function, error) {
	q := `
		SELECT
			n.nspname AS schema,
			p.proname AS name,
			l.lanname AS language,
			pg_get_function_result(p.oid) AS return_type,
			p.provolatile::text AS volatility,
			pg_get_functiondef(p.oid) AS definition
		FROM pg_proc p
		JOIN pg_namespace n ON n.oid = p.pronamespace
		JOIN pg_language l ON l.oid = p.prolang
		WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
			AND p.prokind = 'f'
		ORDER BY n.nspname, p.proname`

	rows, err := d.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres: GetFunctions: %w", err)
	}
	defer rows.Close()

	type funcRow struct {
		schema     string
		name       string
		language   string
		returnType string
		volatility string
		definition string
	}
	var funcRows []funcRow
	for rows.Next() {
		var fr funcRow
		var vol string
		if err := rows.Scan(&fr.schema, &fr.name, &fr.language, &fr.returnType, &vol, &fr.definition); err != nil {
			return nil, fmt.Errorf("postgres: GetFunctions: scan: %w", err)
		}
		switch vol {
		case "i":
			fr.volatility = "IMMUTABLE"
		case "s":
			fr.volatility = "STABLE"
		default:
			fr.volatility = "VOLATILE"
		}
		funcRows = append(funcRows, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: GetFunctions: rows: %w", err)
	}

	var functions []sqlintrospect.Function
	for _, fr := range funcRows {
		args, err := d.getFunctionArgs(ctx, fr.schema, fr.name)
		if err != nil {
			args = []sqlintrospect.FunctionArg{}
		}
		functions = append(functions, sqlintrospect.Function{
			Name:       fr.name,
			Schema:     fr.schema,
			Language:   fr.language,
			ReturnType: fr.returnType,
			Args:       args,
			Volatility: fr.volatility,
			Definition: fr.definition,
		})
	}
	return functions, nil
}

func (d *Driver) getFunctionArgs(ctx context.Context, schema, name string) ([]sqlintrospect.FunctionArg, error) {
	q := `
		SELECT
			COALESCE(p.parameter_name, '') AS arg_name,
			p.data_type AS arg_type,
			p.parameter_mode AS arg_mode
		FROM information_schema.routines r
		JOIN information_schema.parameters p
			ON p.specific_schema = r.specific_schema
			AND p.specific_name = r.specific_name
		WHERE r.routine_schema = $1 AND r.routine_name = $2
		ORDER BY p.ordinal_position`

	rows, err := d.pool.Query(ctx, q, schema, name)
	if err != nil {
		return nil, fmt.Errorf("postgres: getFunctionArgs: %w", err)
	}
	defer rows.Close()

	var args []sqlintrospect.FunctionArg
	for rows.Next() {
		var arg sqlintrospect.FunctionArg
		if err := rows.Scan(&arg.Name, &arg.Type, &arg.Mode); err != nil {
			return nil, fmt.Errorf("postgres: getFunctionArgs: scan: %w", err)
		}
		args = append(args, arg)
	}
	return args, rows.Err()
}

func (d *Driver) GetTriggers(ctx context.Context) ([]sqlintrospect.Trigger, error) {
	q := `
		SELECT
			t.trigger_schema,
			t.trigger_name,
			t.event_object_table,
			t.event_manipulation,
			t.action_timing,
			t.action_orientation,
			t.action_statement,
			COALESCE(pg_t.tgenabled != 'D', true) AS enabled
		FROM information_schema.triggers t
		LEFT JOIN pg_trigger pg_t
			ON pg_t.tgname = t.trigger_name
		WHERE t.trigger_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY t.trigger_schema, t.trigger_name`

	rows, err := d.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres: GetTriggers: %w", err)
	}
	defer rows.Close()

	var triggers []sqlintrospect.Trigger
	for rows.Next() {
		var trig sqlintrospect.Trigger
		if err := rows.Scan(
			&trig.Schema,
			&trig.Name,
			&trig.Table,
			&trig.Event,
			&trig.Timing,
			&trig.ForEach,
			&trig.Definition,
			&trig.Enabled,
		); err != nil {
			return nil, fmt.Errorf("postgres: GetTriggers: scan: %w", err)
		}
		triggers = append(triggers, trig)
	}
	return triggers, rows.Err()
}

func (d *Driver) GetSequences(ctx context.Context) ([]sqlintrospect.Sequence, error) {
	q := `
		SELECT
			ps.schemaname,
			ps.sequencename,
			ps.data_type,
			ps.start_value,
			ps.increment_by,
			ps.min_value,
			ps.max_value,
			COALESCE(ps.last_value, ps.start_value) AS current_value,
			ps.cache_size,
			ps.cycle
		FROM pg_sequences ps
		WHERE ps.schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY ps.schemaname, ps.sequencename`

	rows, err := d.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres: GetSequences: %w", err)
	}
	defer rows.Close()

	var sequences []sqlintrospect.Sequence
	for rows.Next() {
		var seq sqlintrospect.Sequence
		if err := rows.Scan(
			&seq.Schema,
			&seq.Name,
			&seq.DataType,
			&seq.StartValue,
			&seq.Increment,
			&seq.MinValue,
			&seq.MaxValue,
			&seq.CurrentValue,
			&seq.CacheSize,
			&seq.Cycle,
		); err != nil {
			return nil, fmt.Errorf("postgres: GetSequences: scan: %w", err)
		}
		sequences = append(sequences, seq)
	}
	return sequences, rows.Err()
}

func (d *Driver) GetEnums(ctx context.Context) ([]sqlintrospect.Enum, error) {
	q := `
		SELECT
			n.nspname AS schema,
			t.typname AS name,
			array_agg(e.enumlabel ORDER BY e.enumsortorder) AS values
		FROM pg_type t
		JOIN pg_namespace n ON n.oid = t.typnamespace
		JOIN pg_enum e ON e.enumtypid = t.oid
		WHERE t.typtype = 'e'
			AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		GROUP BY n.nspname, t.typname
		ORDER BY n.nspname, t.typname`

	rows, err := d.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres: GetEnums: %w", err)
	}
	defer rows.Close()

	var enums []sqlintrospect.Enum
	for rows.Next() {
		var enum sqlintrospect.Enum
		var values []string
		if err := rows.Scan(&enum.Schema, &enum.Name, &values); err != nil {
			return nil, fmt.Errorf("postgres: GetEnums: scan: %w", err)
		}
		enum.Values = values
		enums = append(enums, enum)
	}
	return enums, rows.Err()
}

func (d *Driver) ExecuteQuery(ctx context.Context, query string) (sqlquery.QueryResult, error) {
	start := time.Now()
	trimmed := strings.TrimSpace(strings.ToUpper(query))

	isSelect := strings.HasPrefix(trimmed, "SELECT") ||
		strings.HasPrefix(trimmed, "WITH") ||
		strings.HasPrefix(trimmed, "SHOW") ||
		strings.HasPrefix(trimmed, "EXPLAIN")

	if isSelect {
		rows, err := d.pool.Query(ctx, query)
		if err != nil {
			return sqlquery.QueryResult{}, fmt.Errorf("postgres: ExecuteQuery: %w", err)
		}
		defer rows.Close()

		result, err := scanRows(rows)
		if err != nil {
			return sqlquery.QueryResult{}, fmt.Errorf("postgres: ExecuteQuery: %w", err)
		}
		result.ExecutionTime = float64(time.Since(start).Milliseconds())
		return result, nil
	}

	tag, err := d.pool.Exec(ctx, query)
	if err != nil {
		return sqlquery.QueryResult{}, fmt.Errorf("postgres: ExecuteQuery: %w", err)
	}
	affected := tag.RowsAffected()
	return sqlquery.QueryResult{
		Columns:       []string{},
		Rows:          []map[string]any{},
		RowCount:      0,
		ExecutionTime: float64(time.Since(start).Milliseconds()),
		AffectedRows:  &affected,
	}, nil
}

func (d *Driver) DropTable(ctx context.Context, schema, table string) error {
	query := fmt.Sprintf("DROP TABLE %s.%s",
		pgx.Identifier{schema}.Sanitize(),
		pgx.Identifier{table}.Sanitize(),
	)
	_, err := d.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("postgres: DropTable: %w", err)
	}
	return nil
}

func (d *Driver) AddColumn(ctx context.Context, schema, table, name, colType string, nullable bool, defaultVal string) error {
	col := fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s %s",
		pgx.Identifier{schema}.Sanitize(),
		pgx.Identifier{table}.Sanitize(),
		pgx.Identifier{name}.Sanitize(),
		colType,
	)
	if !nullable {
		col += " NOT NULL"
	}
	if defaultVal != "" {
		col += " DEFAULT " + defaultVal
	}
	_, err := d.pool.Exec(ctx, col)
	if err != nil {
		return fmt.Errorf("postgres: AddColumn: %w", err)
	}
	return nil
}

func (d *Driver) RefreshMaterializedView(ctx context.Context, schema, name string) error {
	query := fmt.Sprintf("REFRESH MATERIALIZED VIEW %s.%s",
		pgx.Identifier{schema}.Sanitize(),
		pgx.Identifier{name}.Sanitize(),
	)
	_, err := d.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("postgres: RefreshMaterializedView: %w", err)
	}
	return nil
}

func (d *Driver) RenameColumn(ctx context.Context, schema, table, oldName, newName string) error {
	q := fmt.Sprintf("ALTER TABLE %s.%s RENAME COLUMN %s TO %s",
		pgx.Identifier{schema}.Sanitize(), pgx.Identifier{table}.Sanitize(),
		pgx.Identifier{oldName}.Sanitize(), pgx.Identifier{newName}.Sanitize())
	_, err := d.pool.Exec(ctx, q)
	if err != nil {
		return fmt.Errorf("postgres: RenameColumn: %w", err)
	}
	return nil
}

func (d *Driver) AlterColumnType(ctx context.Context, schema, table, column, newType string) error {
	q := fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s TYPE %s",
		pgx.Identifier{schema}.Sanitize(), pgx.Identifier{table}.Sanitize(),
		pgx.Identifier{column}.Sanitize(), newType)
	_, err := d.pool.Exec(ctx, q)
	if err != nil {
		return fmt.Errorf("postgres: AlterColumnType: %w", err)
	}
	return nil
}

func (d *Driver) DropColumn(ctx context.Context, schema, table, column string) error {
	q := fmt.Sprintf("ALTER TABLE %s.%s DROP COLUMN %s",
		pgx.Identifier{schema}.Sanitize(), pgx.Identifier{table}.Sanitize(),
		pgx.Identifier{column}.Sanitize())
	_, err := d.pool.Exec(ctx, q)
	if err != nil {
		return fmt.Errorf("postgres: DropColumn: %w", err)
	}
	return nil
}

func (d *Driver) SetColumnNullable(ctx context.Context, schema, table, column string, nullable bool) error {
	action := "SET NOT NULL"
	if nullable {
		action = "DROP NOT NULL"
	}
	q := fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s %s",
		pgx.Identifier{schema}.Sanitize(), pgx.Identifier{table}.Sanitize(),
		pgx.Identifier{column}.Sanitize(), action)
	_, err := d.pool.Exec(ctx, q)
	if err != nil {
		return fmt.Errorf("postgres: SetColumnNullable: %w", err)
	}
	return nil
}

func (d *Driver) SetColumnDefault(ctx context.Context, schema, table, column, defaultVal string) error {
	var q string
	if defaultVal == "" {
		q = fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s DROP DEFAULT",
			pgx.Identifier{schema}.Sanitize(), pgx.Identifier{table}.Sanitize(),
			pgx.Identifier{column}.Sanitize())
	} else {
		q = fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s SET DEFAULT %s",
			pgx.Identifier{schema}.Sanitize(), pgx.Identifier{table}.Sanitize(),
			pgx.Identifier{column}.Sanitize(), defaultVal)
	}
	_, err := d.pool.Exec(ctx, q)
	if err != nil {
		return fmt.Errorf("postgres: SetColumnDefault: %w", err)
	}
	return nil
}

func (d *Driver) GetTableData(ctx context.Context, schema, table string, limit, offset int) (sqlquery.QueryResult, error) {
	start := time.Now()
	query := fmt.Sprintf("SELECT * FROM %s.%s LIMIT $1 OFFSET $2",
		pgx.Identifier{schema}.Sanitize(),
		pgx.Identifier{table}.Sanitize(),
	)

	rows, err := d.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return sqlquery.QueryResult{}, fmt.Errorf("postgres: GetTableData: %w", err)
	}
	defer rows.Close()

	result, err := scanRows(rows)
	if err != nil {
		return sqlquery.QueryResult{}, fmt.Errorf("postgres: GetTableData: %w", err)
	}
	result.ExecutionTime = float64(time.Since(start).Milliseconds())
	return result, nil
}

func (d *Driver) ExplainQuery(ctx context.Context, query string, analyze bool) (sqlquery.ExplainResult, error) {
	prefix := "EXPLAIN (FORMAT TEXT)"
	if analyze {
		prefix = "EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)"
	}
	explainSQL := prefix + " " + query

	rows, err := d.pool.Query(ctx, explainSQL)
	if err != nil {
		return sqlquery.ExplainResult{}, fmt.Errorf("postgres: ExplainQuery: %w", err)
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return sqlquery.ExplainResult{}, fmt.Errorf("postgres: ExplainQuery: scan: %w", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return sqlquery.ExplainResult{}, fmt.Errorf("postgres: ExplainQuery: rows: %w", err)
	}

	fullPlan := strings.Join(lines, "\n")
	planRows := make([]sqlquery.ExplainRow, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		level := (len(line) - len(trimmed)) / 2
		isNode := strings.Contains(trimmed, "->") || i == 0
		if isNode {
			trimmed = strings.TrimPrefix(trimmed, "->  ")
			trimmed = strings.TrimPrefix(trimmed, "-> ")
		}
		planRows[i] = sqlquery.ExplainRow{
			Text:   trimmed,
			Level:  level,
			IsNode: isNode,
		}
	}

	return sqlquery.ExplainResult{
		Plan:      fullPlan,
		Format:    "text",
		QueryText: query,
		PlanRows:  planRows,
	}, nil
}

func scanRows(rows pgx.Rows) (sqlquery.QueryResult, error) {
	fields := rows.FieldDescriptions()
	columns := make([]string, len(fields))
	for i, f := range fields {
		columns[i] = string(f.Name)
	}

	var resultRows []map[string]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return sqlquery.QueryResult{}, fmt.Errorf("scan values: %w", err)
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			row[col] = convertValue(vals[i])
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return sqlquery.QueryResult{}, err
	}
	if resultRows == nil {
		resultRows = []map[string]any{}
	}
	return sqlquery.QueryResult{
		Columns:  columns,
		Rows:     resultRows,
		RowCount: int64(len(resultRows)),
	}, nil
}

func convertValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case time.Time:
		return val.Format(time.RFC3339)
	case []byte:
		return base64.StdEncoding.EncodeToString(val)
	case pgtype.Numeric:
		if val.Valid {
			text, err := val.MarshalJSON()
			if err == nil {
				return strings.Trim(string(text), `"`)
			}
		}
		return nil
	case pgtype.UUID:
		if val.Valid {
			return fmt.Sprintf("%x-%x-%x-%x-%x",
				val.Bytes[0:4], val.Bytes[4:6], val.Bytes[6:8], val.Bytes[8:10], val.Bytes[10:16])
		}
		return nil
	case [16]byte:
		return fmt.Sprintf("%x-%x-%x-%x-%x",
			val[0:4], val[4:6], val[6:8], val[8:10], val[10:16])
	case int8, int16, int32, int64,
		uint8, uint16, uint32, uint64,
		float32, float64, bool, string, int:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}
