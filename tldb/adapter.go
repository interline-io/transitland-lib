package tldb

import (
	"context"
	"net/url"

	sq "github.com/irees/squirrel"
)

var adapterFactories = map[string]func(string) Adapter{}

// sqliteMemoryDBURL is the sqlite in-memory DSN used throughout this library.
// Go 1.26 tightened net/url and now rejects it because ":memory:" after the
// authority marker parses as host:port with a non-numeric port. It is carved
// out explicitly rather than loosening url.Parse handling in general.
const sqliteMemoryDBURL = "sqlite3://:memory:"

func RegisterAdapter(name string, fn func(string) Adapter) {
	adapterFactories[name] = fn
}

// newAdapter returns a Adapter for the given dburl.
func newAdapter(dburl string) Adapter {
	scheme := "sqlite3"
	if dburl != sqliteMemoryDBURL {
		u, err := url.Parse(dburl)
		if err != nil {
			return nil
		}
		scheme = u.Scheme
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
