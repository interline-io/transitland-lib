package tldb

import (
	"net/url"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var adapters = map[string]func(string) Adapter{}

// newAdapter returns a Adapter for the given dburl.
func newAdapter(dburl string) Adapter {
	u, err := url.Parse(dburl)
	if err != nil {
		return nil
	}
	fn, ok := adapters[u.Scheme]
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
	Insert(interface{}) (int, error)
	Update(interface{}, ...string) error
	Find(interface{}, ...interface{}) error
	Get(interface{}, string, ...interface{}) error
	Select(interface{}, string, ...interface{}) error
	MultiInsert([]interface{}) error
	CopyInsert([]interface{}) error
}
