package tlsql

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
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
	MultiInsert([]interface{}) ([]int, error)
	CopyInsert([]interface{}) error
}
