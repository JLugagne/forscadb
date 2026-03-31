package sqlintrospect

import "context"

type Column struct {
	Name         string
	Type         string
	Nullable     bool
	PrimaryKey   bool
	DefaultValue *string
	ForeignKey   *ForeignKey
}

type ForeignKey struct {
	Table  string
	Column string
}

type Index struct {
	Name    string
	Columns []string
	Unique  bool
	Type    string
}

type Table struct {
	Name     string
	Schema   string
	Columns  []Column
	RowCount int64
	Size     string
	Indexes  []Index
}

type View struct {
	Name         string
	Schema       string
	Definition   string
	Columns      []Column
	Materialized bool
}

type FunctionArg struct {
	Name string
	Type string
	Mode string
}

type Function struct {
	Name       string
	Schema     string
	Language   string
	ReturnType string
	Args       []FunctionArg
	Volatility string
	Definition string
}

type Trigger struct {
	Name       string
	Schema     string
	Table      string
	Event      string
	Timing     string
	ForEach    string
	Function   string
	Enabled    bool
	Definition string
}

type Sequence struct {
	Name         string
	Schema       string
	DataType     string
	StartValue   int64
	Increment    int64
	MinValue     int64
	MaxValue     int64
	CurrentValue int64
	CacheSize    int64
	Cycle        bool
	OwnedBy      *string
}

type Enum struct {
	Name   string
	Schema string
	Values []string
}

type Queries interface {
	GetTables(ctx context.Context, connID string) ([]Table, error)
	GetViews(ctx context.Context, connID string) ([]View, error)
	GetFunctions(ctx context.Context, connID string) ([]Function, error)
	GetTriggers(ctx context.Context, connID string) ([]Trigger, error)
	GetSequences(ctx context.Context, connID string) ([]Sequence, error)
	GetEnums(ctx context.Context, connID string) ([]Enum, error)
}
