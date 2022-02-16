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
}

// QueryLogger wraps sql/sqlx methods with loggers.
type QueryLogger struct {
	sqlx.Ext
}

// Exec .
func (p *QueryLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	rid := p.queryId()
	t := logt1(rid, query, args...)
	defer queryTime(rid, t)
	return p.Ext.Exec(query, args...)
}

// Query .
func (p *QueryLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	rid := p.queryId()
	t := logt1(rid, query, args...)
	defer queryTime(rid, t)
	return p.Ext.Query(query, args...)
}

// Queryx .
func (p *QueryLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	rid := p.queryId()
	t := logt1(rid, query, args...)
	defer queryTime(rid, t)
	return p.Ext.Queryx(query, args...)
}

// QueryRow .
func (p *QueryLogger) QueryRow(query string, args ...interface{}) *sql.Row {
	rid := p.queryId()
	t := logt1(rid, query, args...)
	defer queryTime(rid, t)
	if v, ok := p.Ext.(sqrlExt); ok {
		return v.QueryRow(query, args...)
	}
	return nil
}

// QueryRowx .
func (p *QueryLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	rid := p.queryId()
	t := logt1(rid, query, args...)
	defer queryTime(rid, t)
	return p.Ext.QueryRowx(query, args...)
}

// Beginx .
func (p *QueryLogger) Beginx() (*sqlx.Tx, error) {
	if a, ok := p.Ext.(*sqlx.Tx); ok {
		fmt.Println("ql already in txn")
		return a, nil
	}
	if a, ok := p.Ext.(canBeginx); ok {
		fmt.Println("ql starting txn")
		return a.Beginx()
	}
	return nil, errors.New("cannot start tx")
}

func (p *QueryLogger) queryId() int {
	a := atomic.AddUint64(&queryCounter, 1)
	return int(a)
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
	log.Info().Int("queryId", rid).Str("query", qstr).Strs("queryArgs", sts).Msg("begin")
}

// QueryTime logs database queries and time relative to start; requires LogQuery or TRACE.
func queryTime(rid int, t time.Time) {
	t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e6
	log.Info().Int("queryId", rid).Float64("queryTime", t2).Msgf("complete")
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
