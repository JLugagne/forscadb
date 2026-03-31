package mysql

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlintrospect"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlquery"
)

type Driver struct {
	db *sql.DB
}

func NewMySQLDriver(db *sql.DB) *Driver {
	return &Driver{db: db}
}

func (d *Driver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

func (d *Driver) Close() error {
	return d.db.Close()
}

func (d *Driver) GetTables(ctx context.Context) ([]sqlintrospect.Table, error) {
	q := `
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			COALESCE(TABLE_ROWS, 0),
			COALESCE(CONCAT(ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2), ' MB'), '0 MB')
		FROM information_schema.TABLES
		WHERE TABLE_TYPE = 'BASE TABLE'
			AND TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY TABLE_SCHEMA, TABLE_NAME`

	rows, err := d.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("mysql: GetTables: %w", err)
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
			return nil, fmt.Errorf("mysql: GetTables: scan: %w", err)
		}
		tableRows = append(tableRows, tr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql: GetTables: rows: %w", err)
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
			COLUMN_NAME,
			DATA_TYPE,
			IS_NULLABLE = 'YES',
			COLUMN_DEFAULT,
			COLUMN_KEY = 'PRI'
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`

	rows, err := d.db.QueryContext(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("mysql: getColumns: %w", err)
	}
	defer rows.Close()

	fkMap, err := d.getForeignKeys(ctx, schema, table)
	if err != nil {
		return nil, err
	}

	var columns []sqlintrospect.Column
	for rows.Next() {
		var col sqlintrospect.Column
		var defaultVal sql.NullString
		var nullable, isPK bool
		if err := rows.Scan(&col.Name, &col.Type, &nullable, &defaultVal, &isPK); err != nil {
			return nil, fmt.Errorf("mysql: getColumns: scan: %w", err)
		}
		col.Nullable = nullable
		col.PrimaryKey = isPK
		if defaultVal.Valid {
			col.DefaultValue = &defaultVal.String
		}
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
			kcu.COLUMN_NAME,
			kcu.REFERENCED_TABLE_NAME,
			kcu.REFERENCED_COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE kcu
		JOIN information_schema.TABLE_CONSTRAINTS tc
			ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
			AND tc.TABLE_SCHEMA = kcu.TABLE_SCHEMA
			AND tc.TABLE_NAME = kcu.TABLE_NAME
		WHERE tc.CONSTRAINT_TYPE = 'FOREIGN KEY'
			AND kcu.TABLE_SCHEMA = ?
			AND kcu.TABLE_NAME = ?
			AND kcu.REFERENCED_TABLE_NAME IS NOT NULL`

	rows, err := d.db.QueryContext(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("mysql: getForeignKeys: %w", err)
	}
	defer rows.Close()

	result := make(map[string]sqlintrospect.ForeignKey)
	for rows.Next() {
		var col, ftable, fcol string
		if err := rows.Scan(&col, &ftable, &fcol); err != nil {
			return nil, fmt.Errorf("mysql: getForeignKeys: scan: %w", err)
		}
		result[col] = sqlintrospect.ForeignKey{Table: ftable, Column: fcol}
	}
	return result, rows.Err()
}

func (d *Driver) getIndexes(ctx context.Context, schema, table string) ([]sqlintrospect.Index, error) {
	q := `
		SELECT
			INDEX_NAME,
			NON_UNIQUE = 0 AS is_unique,
			INDEX_TYPE,
			GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) AS columns
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		GROUP BY INDEX_NAME, NON_UNIQUE, INDEX_TYPE
		ORDER BY INDEX_NAME`

	rows, err := d.db.QueryContext(ctx, q, schema, table)
	if err != nil {
		return nil, fmt.Errorf("mysql: getIndexes: %w", err)
	}
	defer rows.Close()

	var indexes []sqlintrospect.Index
	for rows.Next() {
		var idx sqlintrospect.Index
		var colStr string
		if err := rows.Scan(&idx.Name, &idx.Unique, &idx.Type, &colStr); err != nil {
			return nil, fmt.Errorf("mysql: getIndexes: scan: %w", err)
		}
		idx.Columns = strings.Split(colStr, ",")
		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

func (d *Driver) GetViews(ctx context.Context) ([]sqlintrospect.View, error) {
	q := `
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			VIEW_DEFINITION
		FROM information_schema.VIEWS
		WHERE TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY TABLE_SCHEMA, TABLE_NAME`

	rows, err := d.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("mysql: GetViews: %w", err)
	}
	defer rows.Close()

	var views []sqlintrospect.View
	for rows.Next() {
		var v sqlintrospect.View
		if err := rows.Scan(&v.Schema, &v.Name, &v.Definition); err != nil {
			return nil, fmt.Errorf("mysql: GetViews: scan: %w", err)
		}
		columns, err := d.getColumns(ctx, v.Schema, v.Name)
		if err != nil {
			columns = []sqlintrospect.Column{}
		}
		v.Columns = columns
		views = append(views, v)
	}
	return views, rows.Err()
}

func (d *Driver) GetFunctions(ctx context.Context) ([]sqlintrospect.Function, error) {
	q := `
		SELECT
			ROUTINE_SCHEMA,
			ROUTINE_NAME,
			ROUTINE_BODY,
			DTD_IDENTIFIER,
			IS_DETERMINISTIC,
			ROUTINE_DEFINITION
		FROM information_schema.ROUTINES
		WHERE ROUTINE_TYPE = 'FUNCTION'
			AND ROUTINE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY ROUTINE_SCHEMA, ROUTINE_NAME`

	rows, err := d.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("mysql: GetFunctions: %w", err)
	}
	defer rows.Close()

	var functions []sqlintrospect.Function
	for rows.Next() {
		var f sqlintrospect.Function
		var isDeterministic string
		var definition sql.NullString
		if err := rows.Scan(&f.Schema, &f.Name, &f.Language, &f.ReturnType, &isDeterministic, &definition); err != nil {
			return nil, fmt.Errorf("mysql: GetFunctions: scan: %w", err)
		}
		if isDeterministic == "YES" {
			f.Volatility = "DETERMINISTIC"
		} else {
			f.Volatility = "NOT DETERMINISTIC"
		}
		if definition.Valid {
			f.Definition = definition.String
		}
		functions = append(functions, f)
	}
	return functions, rows.Err()
}

func (d *Driver) GetTriggers(ctx context.Context) ([]sqlintrospect.Trigger, error) {
	q := `
		SELECT
			TRIGGER_SCHEMA,
			TRIGGER_NAME,
			EVENT_OBJECT_TABLE,
			EVENT_MANIPULATION,
			ACTION_TIMING,
			ACTION_ORIENTATION,
			ACTION_STATEMENT
		FROM information_schema.TRIGGERS
		WHERE TRIGGER_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY TRIGGER_SCHEMA, TRIGGER_NAME`

	rows, err := d.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("mysql: GetTriggers: %w", err)
	}
	defer rows.Close()

	var triggers []sqlintrospect.Trigger
	for rows.Next() {
		var t sqlintrospect.Trigger
		if err := rows.Scan(
			&t.Schema, &t.Name, &t.Table,
			&t.Event, &t.Timing, &t.ForEach, &t.Definition,
		); err != nil {
			return nil, fmt.Errorf("mysql: GetTriggers: scan: %w", err)
		}
		t.Enabled = true
		triggers = append(triggers, t)
	}
	return triggers, rows.Err()
}

func (d *Driver) GetSequences(_ context.Context) ([]sqlintrospect.Sequence, error) {
	return []sqlintrospect.Sequence{}, nil
}

func (d *Driver) GetEnums(ctx context.Context) ([]sqlintrospect.Enum, error) {
	q := `
		SELECT
			TABLE_SCHEMA,
			COLUMN_NAME,
			COLUMN_TYPE
		FROM information_schema.COLUMNS
		WHERE DATA_TYPE = 'enum'
			AND TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY TABLE_SCHEMA, COLUMN_NAME`

	rows, err := d.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("mysql: GetEnums: %w", err)
	}
	defer rows.Close()

	enumMap := make(map[string]*sqlintrospect.Enum)
	var order []string
	for rows.Next() {
		var schema, colName, colType string
		if err := rows.Scan(&schema, &colName, &colType); err != nil {
			return nil, fmt.Errorf("mysql: GetEnums: scan: %w", err)
		}
		values := parseEnumValues(colType)
		key := schema + "." + colName
		if _, exists := enumMap[key]; !exists {
			enumMap[key] = &sqlintrospect.Enum{
				Name:   colName,
				Schema: schema,
				Values: values,
			}
			order = append(order, key)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql: GetEnums: rows: %w", err)
	}

	var enums []sqlintrospect.Enum
	for _, k := range order {
		enums = append(enums, *enumMap[k])
	}
	return enums, nil
}

func parseEnumValues(colType string) []string {
	colType = strings.TrimPrefix(colType, "enum(")
	colType = strings.TrimSuffix(colType, ")")
	parts := strings.Split(colType, ",")
	values := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "'")
		values = append(values, p)
	}
	return values
}

func (d *Driver) ExecuteQuery(ctx context.Context, query string) (sqlquery.QueryResult, error) {
	start := time.Now()
	trimmed := strings.TrimSpace(strings.ToUpper(query))

	isSelect := strings.HasPrefix(trimmed, "SELECT") ||
		strings.HasPrefix(trimmed, "WITH") ||
		strings.HasPrefix(trimmed, "SHOW") ||
		strings.HasPrefix(trimmed, "EXPLAIN")

	if isSelect {
		rows, err := d.db.QueryContext(ctx, query)
		if err != nil {
			return sqlquery.QueryResult{}, fmt.Errorf("mysql: ExecuteQuery: %w", err)
		}
		defer rows.Close()

		result, err := scanSQLRows(rows)
		if err != nil {
			return sqlquery.QueryResult{}, fmt.Errorf("mysql: ExecuteQuery: %w", err)
		}
		result.ExecutionTime = float64(time.Since(start).Milliseconds())
		return result, nil
	}

	res, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return sqlquery.QueryResult{}, fmt.Errorf("mysql: ExecuteQuery: %w", err)
	}
	affected, _ := res.RowsAffected()
	return sqlquery.QueryResult{
		Columns:       []string{},
		Rows:          []map[string]any{},
		RowCount:      0,
		ExecutionTime: float64(time.Since(start).Milliseconds()),
		AffectedRows:  &affected,
	}, nil
}

func (d *Driver) DropTable(ctx context.Context, schema, table string) error {
	query := fmt.Sprintf("DROP TABLE `%s`.`%s`",
		strings.ReplaceAll(schema, "`", ""),
		strings.ReplaceAll(table, "`", ""),
	)
	_, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("mysql: DropTable: %w", err)
	}
	return nil
}

func (d *Driver) AddColumn(ctx context.Context, schema, table, name, colType string, nullable bool, defaultVal string) error {
	col := fmt.Sprintf("ALTER TABLE `%s`.`%s` ADD COLUMN `%s` %s",
		strings.ReplaceAll(schema, "`", ""),
		strings.ReplaceAll(table, "`", ""),
		strings.ReplaceAll(name, "`", ""),
		colType,
	)
	if !nullable {
		col += " NOT NULL"
	}
	if defaultVal != "" {
		col += " DEFAULT " + defaultVal
	}
	_, err := d.db.ExecContext(ctx, col)
	if err != nil {
		return fmt.Errorf("mysql: AddColumn: %w", err)
	}
	return nil
}

func (d *Driver) RefreshMaterializedView(_ context.Context, _, _ string) error {
	return fmt.Errorf("mysql: materialized views not supported")
}

func (d *Driver) RenameColumn(ctx context.Context, schema, table, oldName, newName string) error {
	q := fmt.Sprintf("ALTER TABLE `%s`.`%s` RENAME COLUMN `%s` TO `%s`",
		strings.ReplaceAll(schema, "`", ""), strings.ReplaceAll(table, "`", ""),
		strings.ReplaceAll(oldName, "`", ""), strings.ReplaceAll(newName, "`", ""))
	_, err := d.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("mysql: RenameColumn: %w", err)
	}
	return nil
}

func (d *Driver) AlterColumnType(ctx context.Context, schema, table, column, newType string) error {
	q := fmt.Sprintf("ALTER TABLE `%s`.`%s` MODIFY COLUMN `%s` %s",
		strings.ReplaceAll(schema, "`", ""), strings.ReplaceAll(table, "`", ""),
		strings.ReplaceAll(column, "`", ""), newType)
	_, err := d.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("mysql: AlterColumnType: %w", err)
	}
	return nil
}

func (d *Driver) DropColumn(ctx context.Context, schema, table, column string) error {
	q := fmt.Sprintf("ALTER TABLE `%s`.`%s` DROP COLUMN `%s`",
		strings.ReplaceAll(schema, "`", ""), strings.ReplaceAll(table, "`", ""),
		strings.ReplaceAll(column, "`", ""))
	_, err := d.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("mysql: DropColumn: %w", err)
	}
	return nil
}

func (d *Driver) SetColumnNullable(ctx context.Context, schema, table, column string, nullable bool) error {
	var colType string
	q := "SELECT COLUMN_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND COLUMN_NAME = ?"
	err := d.db.QueryRowContext(ctx, q, schema, table, column).Scan(&colType)
	if err != nil {
		return fmt.Errorf("mysql: SetColumnNullable: get type: %w", err)
	}
	nullClause := "NOT NULL"
	if nullable {
		nullClause = "NULL"
	}
	alter := fmt.Sprintf("ALTER TABLE `%s`.`%s` MODIFY COLUMN `%s` %s %s",
		strings.ReplaceAll(schema, "`", ""), strings.ReplaceAll(table, "`", ""),
		strings.ReplaceAll(column, "`", ""), colType, nullClause)
	_, err = d.db.ExecContext(ctx, alter)
	if err != nil {
		return fmt.Errorf("mysql: SetColumnNullable: %w", err)
	}
	return nil
}

func (d *Driver) SetColumnDefault(ctx context.Context, schema, table, column, defaultVal string) error {
	var q string
	if defaultVal == "" {
		q = fmt.Sprintf("ALTER TABLE `%s`.`%s` ALTER COLUMN `%s` DROP DEFAULT",
			strings.ReplaceAll(schema, "`", ""), strings.ReplaceAll(table, "`", ""),
			strings.ReplaceAll(column, "`", ""))
	} else {
		q = fmt.Sprintf("ALTER TABLE `%s`.`%s` ALTER COLUMN `%s` SET DEFAULT %s",
			strings.ReplaceAll(schema, "`", ""), strings.ReplaceAll(table, "`", ""),
			strings.ReplaceAll(column, "`", ""), defaultVal)
	}
	_, err := d.db.ExecContext(ctx, q)
	if err != nil {
		return fmt.Errorf("mysql: SetColumnDefault: %w", err)
	}
	return nil
}

func (d *Driver) GetTableData(ctx context.Context, schema, table string, limit, offset int) (sqlquery.QueryResult, error) {
	start := time.Now()
	query := fmt.Sprintf("SELECT * FROM `%s`.`%s` LIMIT ? OFFSET ?",
		strings.ReplaceAll(schema, "`", ""),
		strings.ReplaceAll(table, "`", ""),
	)

	rows, err := d.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return sqlquery.QueryResult{}, fmt.Errorf("mysql: GetTableData: %w", err)
	}
	defer rows.Close()

	result, err := scanSQLRows(rows)
	if err != nil {
		return sqlquery.QueryResult{}, fmt.Errorf("mysql: GetTableData: %w", err)
	}
	result.ExecutionTime = float64(time.Since(start).Milliseconds())
	return result, nil
}

func (d *Driver) ExplainQuery(ctx context.Context, query string, analyze bool) (sqlquery.ExplainResult, error) {
	prefix := "EXPLAIN"
	if analyze {
		prefix = "EXPLAIN ANALYZE"
	}
	explainSQL := prefix + " " + query

	rows, err := d.db.QueryContext(ctx, explainSQL)
	if err != nil {
		return sqlquery.ExplainResult{}, fmt.Errorf("mysql: ExplainQuery: %w", err)
	}
	defer rows.Close()

	if analyze {
		var lines []string
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				return sqlquery.ExplainResult{}, fmt.Errorf("mysql: ExplainQuery: scan: %w", err)
			}
			lines = append(lines, line)
		}
		fullPlan := strings.Join(lines, "\n")
		planRows := make([]sqlquery.ExplainRow, len(lines))
		for i, line := range lines {
			trimmed := strings.TrimLeft(line, " ")
			level := (len(line) - len(trimmed)) / 4
			planRows[i] = sqlquery.ExplainRow{Text: trimmed, Level: level, IsNode: strings.Contains(trimmed, "->") || i == 0}
		}
		return sqlquery.ExplainResult{Plan: fullPlan, Format: "text", QueryText: query, PlanRows: planRows}, nil
	}

	result, err := scanSQLRows(rows)
	if err != nil {
		return sqlquery.ExplainResult{}, fmt.Errorf("mysql: ExplainQuery: %w", err)
	}
	var lines []string
	if len(result.Columns) > 0 {
		lines = append(lines, strings.Join(result.Columns, "\t"))
	}
	for _, row := range result.Rows {
		var vals []string
		for _, col := range result.Columns {
			vals = append(vals, fmt.Sprintf("%v", row[col]))
		}
		lines = append(lines, strings.Join(vals, "\t"))
	}
	fullPlan := strings.Join(lines, "\n")
	planRows := make([]sqlquery.ExplainRow, len(lines))
	for i, line := range lines {
		planRows[i] = sqlquery.ExplainRow{Text: line, Level: 0, IsNode: i > 0}
	}
	return sqlquery.ExplainResult{Plan: fullPlan, Format: "text", QueryText: query, PlanRows: planRows}, nil
}

func scanSQLRows(rows *sql.Rows) (sqlquery.QueryResult, error) {
	cols, err := rows.Columns()
	if err != nil {
		return sqlquery.QueryResult{}, fmt.Errorf("columns: %w", err)
	}

	var resultRows []map[string]any
	for rows.Next() {
		scanVals := make([]any, len(cols))
		scanPtrs := make([]any, len(cols))
		for i := range scanVals {
			scanPtrs[i] = &scanVals[i]
		}
		if err := rows.Scan(scanPtrs...); err != nil {
			return sqlquery.QueryResult{}, fmt.Errorf("scan: %w", err)
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			row[col] = convertMySQLValue(scanVals[i])
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
		Columns:  cols,
		Rows:     resultRows,
		RowCount: int64(len(resultRows)),
	}, nil
}

func convertMySQLValue(v any) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case time.Time:
		return val.Format(time.RFC3339)
	case []byte:
		s := string(val)
		if isValidUTF8String(s) {
			return s
		}
		return base64.StdEncoding.EncodeToString(val)
	default:
		return val
	}
}

func isValidUTF8String(s string) bool {
	for _, r := range s {
		if r == '\uFFFD' {
			return false
		}
	}
	return true
}
