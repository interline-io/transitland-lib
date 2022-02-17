package tldb

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/interline-io/transitland-lib/log"
	"github.com/jmoiron/sqlx"
)

var queryCounter = uint64(0)

type sqrlExt interface {
	QueryRow(string, ...interface{}) *sql.Row
	Prepare(query string) (*sql.Stmt, error)
}

// QueryLogger wraps sql/sqlx methods with loggers.
type QueryLogger struct {
	sqlx.Ext
	Trace bool
}

// Exec .
func (p *QueryLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	t, rid := p.queryId()
	if p.Trace {
		logt1(rid, query, args...)
	}
	defer queryTime(rid, t, query, args...)
	return p.Ext.Exec(query, args...)
}

// Query .
func (p *QueryLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	t, rid := p.queryId()
	if p.Trace {
		logt1(rid, query, args...)
	}
	defer queryTime(rid, t, query, args...)
	return p.Ext.Query(query, args...)
}

// Queryx .
func (p *QueryLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	t, rid := p.queryId()
	if p.Trace {
		logt1(rid, query, args...)
	}
	defer queryTime(rid, t, query, args...)
	return p.Ext.Queryx(query, args...)
}

// QueryRow .
func (p *QueryLogger) QueryRow(query string, args ...interface{}) *sql.Row {
	t, rid := p.queryId()
	if p.Trace {
		logt1(rid, query, args...)
	}
	defer queryTime(rid, t, query, args...)
	if v, ok := p.Ext.(sqrlExt); ok {
		return v.QueryRow(query, args...)
	}
	return nil
}

// Prepare
func (p *QueryLogger) Prepare(query string) (*sql.Stmt, error) {
	t, rid := p.queryId()
	if p.Trace {
		logt1(rid, query)
	}
	defer queryTime(rid, t, query)
	if v, ok := p.Ext.(sqrlExt); ok {
		return v.Prepare(query)
	}
	return nil, errors.New("not Preparer")
}

// QueryRowx .
func (p *QueryLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	t, rid := p.queryId()
	if p.Trace {
		logt1(rid, query, args...)
	}
	defer queryTime(rid, t, query, args...)
	return p.Ext.QueryRowx(query, args...)
}

// Beginx .
func (p *QueryLogger) Beginx() (*sqlx.Tx, error) {
	if a, ok := p.Ext.(canBeginx); ok {
		return a.Beginx()
	}
	return nil, errors.New("not Beginxer")
}

func (p *QueryLogger) queryId() (time.Time, int) {
	t := time.Now()
	a := atomic.AddUint64(&queryCounter, 1)
	return t, int(a)
}

//////

func logt1(rid int, qstr string, a ...interface{}) time.Time {
	t := time.Now()
	queryStart(rid, qstr, a...)
	return t
}

// QueryStart logs database query beginnings; requires TRACE.
func queryStart(rid int, qstr string, a ...interface{}) {
	sts := []string{}
	for i, val := range a {
		q := qval{strconv.Itoa(i + 1), val}
		sts = append(sts, q.String())
	}
	log.Trace().Int("queryId", rid).Str("query", qstr).Strs("queryArgs", sts).Msg("begin")
}

// QueryTime logs database queries and time relative to start; requires LogQuery or TRACE.
func queryTime(rid int, t time.Time, qstr string, a ...interface{}) {
	t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e6
	sts := []string{}
	for i, val := range a {
		q := qval{strconv.Itoa(i + 1), val}
		sts = append(sts, q.String())
	}
	log.Trace().Int("queryId", rid).Str("query", qstr).Strs("queryArgs", sts).Float64("queryTime", t2).Msg("complete")
}

// Some helpers

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
			z = "(binary)"
		} else if z == nil {
			z = "(nil)"
		}
		s = fmt.Sprintf("%v", z)
	} else if q.Value == nil {
		s = "(nil)"
	} else {
		s = fmt.Sprintf("%v", q.Value)
	}
	return s
}
