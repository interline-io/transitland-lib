package gtdb

import (
	"net/url"

	sq "github.com/Masterminds/squirrel"
	tl "github.com/interline-io/transitland-lib"
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

// Adapter implements details specific to each backend.
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
	BatchInsert([]tl.Entity) error
}
