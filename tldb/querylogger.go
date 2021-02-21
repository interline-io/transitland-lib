package tldb

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/jmoiron/sqlx"
)

type canBeginx interface {
	Beginx() (*sqlx.Tx, error)
}

type canClose interface {
	Close() error
}

// canToSQL is the squirrel interface
type canToSQL interface {
	ToSql() (string, []interface{}, error)
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

// QueryLogger wraps sql/sqlx methods with loggers.
type QueryLogger struct {
	sqext
}

// NewQueryLogger returns the db wrapped into a QueryLogger.
func NewQueryLogger(db *sqlx.DB) *QueryLogger {
	return &QueryLogger{sqext: db.Unsafe()}
}

func logt1(qstr string, a ...interface{}) time.Time {
	t := time.Now()
	log.QueryStart(qstr, a...)
	return t
}

// Exec .
func (p *QueryLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	t := logt1(query, args...)
	defer log.QueryTime(t, query, args...)
	return p.sqext.Exec(query, args...)
}

// Query .
func (p *QueryLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	t := logt1(query, args...)
	defer log.QueryTime(t, query, args...)
	return p.sqext.Query(query, args...)
}

// QueryRow .
func (p *QueryLogger) QueryRow(query string, args ...interface{}) *sql.Row {
	t := logt1(query, args...)
	defer log.QueryTime(t, query, args...)
	return p.sqext.QueryRow(query, args...)
}

// Queryx .
func (p *QueryLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	t := logt1(query, args...)
	defer log.QueryTime(t, query, args...)
	return p.sqext.Queryx(query, args...)
}

// QueryRowx .
func (p *QueryLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	t := logt1(query, args...)
	defer log.QueryTime(t, query, args...)
	return p.sqext.QueryRowx(query, args...)
}

func (p *QueryLogger) Beginx() (*sqlx.Tx, error) {
	if a, ok := p.sqext.(*sqlx.Tx); ok {
		return a, nil
	}
	if a, ok := p.sqext.(canBeginx); ok {
		return a.Beginx()
	}
	return nil, errors.New("cannot start tx")
}
