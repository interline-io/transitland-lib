package tldb

import (
	"context"
	"net/url"

	sq "github.com/Masterminds/squirrel"
)

var adapterFactories = map[string]func(string) Adapter{}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// newAdapter returns a Adapter for the given dburl.
func newAdapter(dburl string) Adapter {
	u, err := url.Parse(dburl)
	if err != nil {
		return nil
	}
	fn, ok := adapterFactories[u.Scheme]
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
	//////////
	Insert(interface{}) (int, error)
	Update(interface{}, ...string) error
	Find(interface{}) error
	Get(interface{}, string, ...interface{}) error
	Select(interface{}, string, ...interface{}) error
	MultiInsert([]interface{}) ([]int, error)
	//////////
	InsertContext(context.Context, interface{}) (int, error)
	UpdateContext(context.Context, interface{}, ...string) error
	FindContext(context.Context, interface{}) error
	GetContext(context.Context, interface{}, string, ...interface{}) error
	SelectContext(context.Context, interface{}, string, ...interface{}) error
	MultiInsertContext(context.Context, []interface{}) ([]int, error)
}
