package tlsql

import (
	// Driver
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type canBeginx interface {
	Beginx() (*sqlx.Tx, error)
}

type canClose interface {
	Close() error
}

// ext is for wrapped sqlx to be used in squirrel.
type sqext interface {
	sqlx.Ext
	// These are required for squirrel.. :(
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}
