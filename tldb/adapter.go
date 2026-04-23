package tldb

import (
	"context"
	"strings"

	sq "github.com/irees/squirrel"
)

var adapterFactories = map[string]func(string) Adapter{}

func RegisterAdapter(name string, fn func(string) Adapter) {
	adapterFactories[name] = fn
}

// newAdapter returns a Adapter for the given dburl.
// Uses plain string splitting rather than url.Parse so that driver-specific
// forms like "sqlite3://:memory:" (where ":memory:" is not a valid host:port)
// are accepted.
func newAdapter(dburl string) Adapter {
	scheme, _, ok := strings.Cut(dburl, ":")
	if !ok {
		return nil
	}
	fn, ok := adapterFactories[scheme]
	if !ok {
		return nil
	}
	return fn(dburl)
}

// Adapter provides an interface for connecting to various kinds of database backends.
type Adapter interface {
	Open() error
	Close() error
	Create() error
	DBX() Ext
	Tx(func(Adapter) error) error
	Sqrl() sq.StatementBuilderType
	TableExists(string) (bool, error)
	Insert(context.Context, interface{}) (int, error)
	Update(context.Context, interface{}, ...string) error
	Find(context.Context, interface{}) error
	Get(context.Context, interface{}, string, ...interface{}) error
	Select(context.Context, interface{}, string, ...interface{}) error
	MultiInsert(context.Context, []interface{}) ([]int, error)
	SupportsSpatialFunctions() bool
}
