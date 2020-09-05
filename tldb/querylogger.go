package tldb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gookit/color"
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

type canValue interface {
	Value() (driver.Value, error)
}

type qval struct {
	Name  string
	Value interface{}
}

func (q qval) String() string {
	s := ""
	if a, ok := q.Value.(canValue); ok {
		z, _ := a.Value()
		if x, ok := z.([]byte); ok {
			_ = x
			z = "<binary>"
		}
		s = fmt.Sprintf("%v", z)
	} else {
		s = fmt.Sprintf("%v", q.Value)
	}
	return fmt.Sprintf("{%s:%s}", q.Name, s)
}

// Query for logging database queries.
func qlog(qstr string, a ...interface{}) {
	if !log.LogQuery {
		return
	}
	sts := []string{}
	for i, val := range a {
		q := qval{strconv.Itoa(i + 1), val}
		sts = append(sts, q.String())
	}
	fmta := qstr
	log.Query(color.Blue.Render(fmta) + " -- " + color.Gray.Render(strings.Join(sts, " ")))
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

// Exec .
func (p *queryLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	qlog(query, args...)
	return p.sqext.Exec(query, args...)
}

// Query .
func (p *queryLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	qlog(query, args...)
	return p.sqext.Query(query, args...)
}

// QueryRow .
func (p *queryLogger) QueryRow(query string, args ...interface{}) *sql.Row {
	qlog(query, args...)
	return p.sqext.QueryRow(query, args...)
}

// Queryx .
func (p *queryLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	qlog(query, args...)
	return p.sqext.Queryx(query, args...)
}

// QueryRowx .
func (p *queryLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	qlog(query, args...)
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
