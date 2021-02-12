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

// queryLogger wraps sql/sqlx methods with loggers.
type queryLogger struct {
	sqext
}

func logt1(qstr string, a ...interface{}) time.Time {
	t := time.Now()
	log.QueryStart(qstr, a...)
	return t
}

// Exec .
func (p *queryLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	t := logt1(query, args...)
	defer log.QueryTime(t, query, args...)
	return p.sqext.Exec(query, args...)
}

// Query .
func (p *queryLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	t := logt1(query, args...)
	defer log.QueryTime(t, query, args...)
	return p.sqext.Query(query, args...)
}

// QueryRow .
func (p *queryLogger) QueryRow(query string, args ...interface{}) *sql.Row {
	t := logt1(query, args...)
	defer log.QueryTime(t, query, args...)
	return p.sqext.QueryRow(query, args...)
}

// Queryx .
func (p *queryLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	t := logt1(query, args...)
	defer log.QueryTime(t, query, args...)
	return p.sqext.Queryx(query, args...)
}

// QueryRowx .
func (p *queryLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	t := logt1(query, args...)
	defer log.QueryTime(t, query, args...)
	return p.sqext.QueryRowx(query, args...)
}

func (p *queryLogger) Beginx() (*sqlx.Tx, error) {
	if a, ok := p.sqext.(*sqlx.Tx); ok {
		return a, nil
	}
	if a, ok := p.sqext.(canBeginx); ok {
		return a.Beginx()
	}
	return nil, errors.New("cannot start tx")
}
