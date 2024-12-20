package tldb

import (
	"net/url"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
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
	DBX() sqlx.Ext
	Tx(func(Adapter) error) error
	Sqrl() sq.StatementBuilderType
	TableExists(string) (bool, error)
	Insert(interface{}) (int, error)
	Update(interface{}, ...string) error
	Find(interface{}) error
	Get(interface{}, string, ...interface{}) error
	Select(interface{}, string, ...interface{}) error
	MultiInsert([]interface{}) ([]int, error)
}
