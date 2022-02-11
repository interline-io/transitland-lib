package log

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/jmoiron/sqlx"
)

// QueryLogger wraps sql/sqlx methods with loggers.
type QueryLogger struct {
	Trace bool
	sqlx.Ext
}

// Exec .
func (p *QueryLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	t := logt1(query, args...)
	if p.Trace {
		defer QueryTime(t, query, args...)
	}
	return p.Ext.Exec(query, args...)
}

// Query .
func (p *QueryLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	t := logt1(query, args...)
	if p.Trace {
		defer QueryTime(t, query, args...)
	}
	return p.Ext.Query(query, args...)
}

// Queryx .
func (p *QueryLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	t := logt1(query, args...)
	if p.Trace {
		defer QueryTime(t, query, args...)
	}
	return p.Ext.Queryx(query, args...)
}

// QueryRowx .
func (p *QueryLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	t := logt1(query, args...)
	if p.Trace {
		defer QueryTime(t, query, args...)
	}
	return p.Ext.QueryRowx(query, args...)
}

// Beginx .
func (p *QueryLogger) Beginx() (*sqlx.Tx, error) {
	if a, ok := p.Ext.(*sqlx.Tx); ok {
		return a, nil
	}
	if a, ok := p.Ext.(canBeginx); ok {
		return a.Beginx()
	}
	return nil, errors.New("cannot start tx")
}

//////

func logt1(qstr string, a ...interface{}) time.Time {
	t := time.Now()
	QueryStart(qstr, a...)
	return t
}

// QueryStart logs database query beginnings; requires TRACE.
func QueryStart(qstr string, a ...interface{}) {
	sts := []string{}
	for i, val := range a {
		q := qval{strconv.Itoa(i + 1), val}
		sts = append(sts, q.String())
	}
	Tracef("%s -- %s [start]", color.Blue.Render(qstr), color.Gray.Render(strings.Join(sts, " ")))
}

// QueryTime logs database queries and time relative to start; requires LogQuery or TRACE.
func QueryTime(t time.Time, qstr string, a ...interface{}) {
	t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e6
	sts := []string{}
	for i, val := range a {
		q := qval{strconv.Itoa(i + 1), val}
		sts = append(sts, q.String())
	}
	Tracef("[%s -- %s [time: %0.2f ms]", color.Blue.Render(qstr), color.Gray.Render(strings.Join(sts, " ")), t2)
}

// Some helpers

type canBeginx interface {
	Beginx() (*sqlx.Tx, error)
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
