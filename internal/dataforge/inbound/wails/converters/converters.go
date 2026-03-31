package converters

import (
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/connection"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/kv"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/nosql"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlintrospect"
	"github.com/JLugagne/forscadb/internal/dataforge/domain/service/sqlquery"
	"github.com/JLugagne/forscadb/internal/domain"
)

type PublicConnection struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Engine     string `json:"engine"`
	Category   string `json:"category"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password,omitempty"`
	Database   string `json:"database"`
	SSLMode    string `json:"sslMode,omitempty"`
	Status     string `json:"status"`
	Color      string `json:"color"`
	LastAccess string `json:"lastAccess"`
}

type PublicForeignKey struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

type PublicSQLColumn struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Nullable     bool              `json:"nullable"`
	PrimaryKey   bool              `json:"primaryKey"`
	DefaultValue *string           `json:"defaultValue"`
	ForeignKey   *PublicForeignKey `json:"foreignKey"`
}

type PublicSQLIndex struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Type    string   `json:"type"`
}

type PublicSQLTable struct {
	Name     string           `json:"name"`
	Schema   string           `json:"schema"`
	Columns  []PublicSQLColumn `json:"columns"`
	RowCount int64            `json:"rowCount"`
	Size     string           `json:"size"`
	Indexes  []PublicSQLIndex `json:"indexes"`
}

type PublicSQLView struct {
	Name         string           `json:"name"`
	Schema       string           `json:"schema"`
	Definition   string           `json:"definition"`
	Columns      []PublicSQLColumn `json:"columns"`
	Materialized bool             `json:"materialized"`
}

type PublicFunctionArg struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Mode string `json:"mode"`
}

type PublicSQLFunction struct {
	Name       string              `json:"name"`
	Schema     string              `json:"schema"`
	Language   string              `json:"language"`
	ReturnType string              `json:"returnType"`
	Args       []PublicFunctionArg `json:"args"`
	Volatility string              `json:"volatility"`
	Definition string              `json:"definition"`
}

type PublicSQLTrigger struct {
	Name       string `json:"name"`
	Schema     string `json:"schema"`
	Table      string `json:"table"`
	Event      string `json:"event"`
	Timing     string `json:"timing"`
	ForEach    string `json:"forEach"`
	Function   string `json:"function"`
	Enabled    bool   `json:"enabled"`
	Definition string `json:"definition"`
}

type PublicSQLSequence struct {
	Name         string  `json:"name"`
	Schema       string  `json:"schema"`
	DataType     string  `json:"dataType"`
	StartValue   int64   `json:"startValue"`
	Increment    int64   `json:"increment"`
	MinValue     int64   `json:"minValue"`
	MaxValue     int64   `json:"maxValue"`
	CurrentValue int64   `json:"currentValue"`
	CacheSize    int64   `json:"cacheSize"`
	Cycle        bool    `json:"cycle"`
	OwnedBy      *string `json:"ownedBy"`
}

type PublicSQLEnum struct {
	Name   string   `json:"name"`
	Schema string   `json:"schema"`
	Values []string `json:"values"`
}

type PublicQueryResult struct {
	Columns       []string         `json:"columns"`
	Rows          []map[string]any `json:"rows"`
	RowCount      int64            `json:"rowCount"`
	ExecutionTime float64          `json:"executionTime"`
	AffectedRows  *int64           `json:"affectedRows"`
}

type PublicHistoryEntry struct {
	ID          string  `json:"id"`
	Query       string  `json:"query"`
	ExecutedAt  string  `json:"executedAt"`
	Duration    float64 `json:"duration"`
	RowCount    int64   `json:"rowCount"`
	Status      string  `json:"status"`
	Error       *string `json:"error"`
}

type PublicNoSQLIndex struct {
	Name   string         `json:"name"`
	Keys   map[string]int `json:"keys"`
	Unique bool           `json:"unique"`
}

type PublicNoSQLCollection struct {
	Name          string             `json:"name"`
	DocumentCount int64              `json:"documentCount"`
	AvgDocSize    string             `json:"avgDocSize"`
	TotalSize     string             `json:"totalSize"`
	Indexes       []PublicNoSQLIndex `json:"indexes"`
}

type PublicKVEntry struct {
	Key      string  `json:"key"`
	Value    string  `json:"value"`
	Type     string  `json:"type"`
	TTL      *int64  `json:"ttl"`
	Size     string  `json:"size"`
	Encoding string  `json:"encoding"`
}

type PublicKVStats struct {
	TotalKeys        int64   `json:"totalKeys"`
	MemoryUsed       string  `json:"memoryUsed"`
	MemoryPeak       string  `json:"memoryPeak"`
	ConnectedClients int64   `json:"connectedClients"`
	OpsPerSec        int64   `json:"opsPerSec"`
	HitRate          float64 `json:"hitRate"`
	UptimeDays       int64   `json:"uptimeDays"`
	KeyspaceHits     int64   `json:"keyspaceHits"`
	KeyspaceMisses   int64   `json:"keyspaceMisses"`
}

func ToPublicConnection(c connection.Connection) PublicConnection {
	return PublicConnection{
		ID:       c.ID,
		Name:     c.Name,
		Engine:   string(c.Engine),
		Category: string(c.Category),
		Host:     c.Host,
		Port:     c.Port,
		User:     c.User,
		Database: c.Database,
		SSLMode:  c.SSLMode,
		Status:   string(c.Status),
		Color:    c.Color,
	}
}

func ToDomainConnection(p PublicConnection) connection.Connection {
	return connection.Connection{
		ID:       p.ID,
		Name:     p.Name,
		Engine:   domain.DatabaseEngine(p.Engine),
		Category: domain.DatabaseCategory(p.Category),
		Host:     p.Host,
		Port:     p.Port,
		User:     p.User,
		Password: p.Password,
		Database: p.Database,
		SSLMode:  p.SSLMode,
		Status:   domain.ConnectionStatus(p.Status),
		Color:    p.Color,
	}
}

func toPublicColumn(c sqlintrospect.Column) PublicSQLColumn {
	pub := PublicSQLColumn{
		Name:         c.Name,
		Type:         c.Type,
		Nullable:     c.Nullable,
		PrimaryKey:   c.PrimaryKey,
		DefaultValue: c.DefaultValue,
	}
	if c.ForeignKey != nil {
		pub.ForeignKey = &PublicForeignKey{
			Table:  c.ForeignKey.Table,
			Column: c.ForeignKey.Column,
		}
	}
	return pub
}

func toPublicColumns(cols []sqlintrospect.Column) []PublicSQLColumn {
	out := make([]PublicSQLColumn, len(cols))
	for i, c := range cols {
		out[i] = toPublicColumn(c)
	}
	return out
}

func toPublicIndexes(idxs []sqlintrospect.Index) []PublicSQLIndex {
	out := make([]PublicSQLIndex, len(idxs))
	for i, idx := range idxs {
		out[i] = PublicSQLIndex{
			Name:    idx.Name,
			Columns: idx.Columns,
			Unique:  idx.Unique,
			Type:    idx.Type,
		}
	}
	return out
}

func ToPublicTable(t sqlintrospect.Table) PublicSQLTable {
	return PublicSQLTable{
		Name:     t.Name,
		Schema:   t.Schema,
		Columns:  toPublicColumns(t.Columns),
		RowCount: t.RowCount,
		Size:     t.Size,
		Indexes:  toPublicIndexes(t.Indexes),
	}
}

func ToPublicTables(tables []sqlintrospect.Table) []PublicSQLTable {
	out := make([]PublicSQLTable, len(tables))
	for i, t := range tables {
		out[i] = ToPublicTable(t)
	}
	return out
}

func ToPublicView(v sqlintrospect.View) PublicSQLView {
	return PublicSQLView{
		Name:         v.Name,
		Schema:       v.Schema,
		Definition:   v.Definition,
		Columns:      toPublicColumns(v.Columns),
		Materialized: v.Materialized,
	}
}

func ToPublicViews(views []sqlintrospect.View) []PublicSQLView {
	out := make([]PublicSQLView, len(views))
	for i, v := range views {
		out[i] = ToPublicView(v)
	}
	return out
}

func ToPublicFunction(f sqlintrospect.Function) PublicSQLFunction {
	args := make([]PublicFunctionArg, len(f.Args))
	for i, a := range f.Args {
		args[i] = PublicFunctionArg{Name: a.Name, Type: a.Type, Mode: a.Mode}
	}
	return PublicSQLFunction{
		Name:       f.Name,
		Schema:     f.Schema,
		Language:   f.Language,
		ReturnType: f.ReturnType,
		Args:       args,
		Volatility: f.Volatility,
		Definition: f.Definition,
	}
}

func ToPublicFunctions(fns []sqlintrospect.Function) []PublicSQLFunction {
	out := make([]PublicSQLFunction, len(fns))
	for i, f := range fns {
		out[i] = ToPublicFunction(f)
	}
	return out
}

func ToPublicTrigger(t sqlintrospect.Trigger) PublicSQLTrigger {
	return PublicSQLTrigger{
		Name:       t.Name,
		Schema:     t.Schema,
		Table:      t.Table,
		Event:      t.Event,
		Timing:     t.Timing,
		ForEach:    t.ForEach,
		Function:   t.Function,
		Enabled:    t.Enabled,
		Definition: t.Definition,
	}
}

func ToPublicTriggers(triggers []sqlintrospect.Trigger) []PublicSQLTrigger {
	out := make([]PublicSQLTrigger, len(triggers))
	for i, t := range triggers {
		out[i] = ToPublicTrigger(t)
	}
	return out
}

func ToPublicSequence(s sqlintrospect.Sequence) PublicSQLSequence {
	return PublicSQLSequence{
		Name:         s.Name,
		Schema:       s.Schema,
		DataType:     s.DataType,
		StartValue:   s.StartValue,
		Increment:    s.Increment,
		MinValue:     s.MinValue,
		MaxValue:     s.MaxValue,
		CurrentValue: s.CurrentValue,
		CacheSize:    s.CacheSize,
		Cycle:        s.Cycle,
		OwnedBy:      s.OwnedBy,
	}
}

func ToPublicSequences(seqs []sqlintrospect.Sequence) []PublicSQLSequence {
	out := make([]PublicSQLSequence, len(seqs))
	for i, s := range seqs {
		out[i] = ToPublicSequence(s)
	}
	return out
}

func ToPublicEnum(e sqlintrospect.Enum) PublicSQLEnum {
	return PublicSQLEnum{
		Name:   e.Name,
		Schema: e.Schema,
		Values: e.Values,
	}
}

func ToPublicEnums(enums []sqlintrospect.Enum) []PublicSQLEnum {
	out := make([]PublicSQLEnum, len(enums))
	for i, e := range enums {
		out[i] = ToPublicEnum(e)
	}
	return out
}

func ToPublicQueryResult(r sqlquery.QueryResult) PublicQueryResult {
	return PublicQueryResult{
		Columns:       r.Columns,
		Rows:          r.Rows,
		RowCount:      r.RowCount,
		ExecutionTime: r.ExecutionTime,
		AffectedRows:  r.AffectedRows,
	}
}

func ToPublicHistoryEntry(e sqlquery.HistoryEntry) PublicHistoryEntry {
	return PublicHistoryEntry{
		ID:         e.ID,
		Query:      e.Query,
		ExecutedAt: e.ExecutedAt,
		Duration:   e.Duration,
		RowCount:   e.RowCount,
		Status:     e.Status,
		Error:      e.Error,
	}
}

type PublicExplainResult struct {
	Plan      string             `json:"plan"`
	Format    string             `json:"format"`
	QueryText string             `json:"queryText"`
	PlanRows  []PublicExplainRow `json:"planRows"`
}

type PublicExplainRow struct {
	Text   string `json:"text"`
	Level  int    `json:"level"`
	IsNode bool   `json:"isNode"`
}

func ToPublicExplainResult(r sqlquery.ExplainResult) PublicExplainResult {
	rows := make([]PublicExplainRow, len(r.PlanRows))
	for i, row := range r.PlanRows {
		rows[i] = PublicExplainRow{Text: row.Text, Level: row.Level, IsNode: row.IsNode}
	}
	return PublicExplainResult{
		Plan:      r.Plan,
		Format:    r.Format,
		QueryText: r.QueryText,
		PlanRows:  rows,
	}
}

func ToPublicHistoryEntries(entries []sqlquery.HistoryEntry) []PublicHistoryEntry {
	out := make([]PublicHistoryEntry, len(entries))
	for i, e := range entries {
		out[i] = ToPublicHistoryEntry(e)
	}
	return out
}

func ToPublicCollection(c nosql.Collection) PublicNoSQLCollection {
	idxs := make([]PublicNoSQLIndex, len(c.Indexes))
	for i, idx := range c.Indexes {
		idxs[i] = PublicNoSQLIndex{Name: idx.Name, Keys: idx.Keys, Unique: idx.Unique}
	}
	return PublicNoSQLCollection{
		Name:          c.Name,
		DocumentCount: c.DocumentCount,
		AvgDocSize:    c.AvgDocSize,
		TotalSize:     c.TotalSize,
		Indexes:       idxs,
	}
}

func ToPublicCollections(cols []nosql.Collection) []PublicNoSQLCollection {
	out := make([]PublicNoSQLCollection, len(cols))
	for i, c := range cols {
		out[i] = ToPublicCollection(c)
	}
	return out
}

func ToPublicKVEntry(e kv.Entry) PublicKVEntry {
	return PublicKVEntry{
		Key:      e.Key,
		Value:    e.Value,
		Type:     e.Type,
		TTL:      e.TTL,
		Size:     e.Size,
		Encoding: e.Encoding,
	}
}

func ToPublicKVEntries(entries []kv.Entry) []PublicKVEntry {
	out := make([]PublicKVEntry, len(entries))
	for i, e := range entries {
		out[i] = ToPublicKVEntry(e)
	}
	return out
}

func ToPublicKVStats(s kv.Stats) PublicKVStats {
	return PublicKVStats{
		TotalKeys:        s.TotalKeys,
		MemoryUsed:       s.MemoryUsed,
		MemoryPeak:       s.MemoryPeak,
		ConnectedClients: s.ConnectedClients,
		OpsPerSec:        s.OpsPerSec,
		HitRate:          s.HitRate,
		UptimeDays:       s.UptimeDays,
		KeyspaceHits:     s.KeyspaceHits,
		KeyspaceMisses:   s.KeyspaceMisses,
	}
}
