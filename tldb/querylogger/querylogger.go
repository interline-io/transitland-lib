package querylogger

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/interline-io/log"
	"github.com/jmoiron/sqlx"
)

var queryCounter = uint64(0)

type Ext interface {
	sqlx.Ext
	sqlx.QueryerContext
	sqlx.ExecerContext
	// QueryRowContext is missing from sqlx.QueryerContext, despite having QueryRowContext
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func init() {
	var _ Ext = &QueryLogger{}
}

// QueryLogger wraps sql/sqlx methods with loggers.
type QueryLogger struct {
	LongQueryDuration time.Duration
	Trace             bool
	Ext
}

// Exec .
func (p *QueryLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	return p.ExecContext(context.Background(), query, args...)
}

func (p *QueryLogger) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	rid := p.queryId()
	if p.Trace {
		queryStart(ctx, rid, query, args...)
	}

	t := time.Now()
	result, err := p.Ext.ExecContext(ctx, query, args...)
	p.queryTime(ctx, rid, t, query, args...)

	return result, err
}

// Query .
func (p *QueryLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return p.QueryContext(context.Background(), query, args...)
}

func (p *QueryLogger) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	rid := p.queryId()
	if p.Trace {
		queryStart(ctx, rid, query, args...)
	}

	t := time.Now()
	result, err := p.Ext.QueryContext(ctx, query, args...)
	p.queryTime(ctx, rid, t, query, args...)

	return result, err
}

// Queryx .
func (p *QueryLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	return p.QueryxContext(context.Background(), query, args...)
}

func (p *QueryLogger) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	rid := p.queryId()
	if p.Trace {
		queryStart(ctx, rid, query, args...)
	}

	t := time.Now()
	result, err := p.Ext.QueryxContext(ctx, query, args...)
	p.queryTime(ctx, rid, t, query, args...)

	return result, err
}

// QueryRow .
func (p *QueryLogger) QueryRow(query string, args ...interface{}) *sql.Row {
	return p.QueryRowContext(context.Background(), query, args...)
}

func (p *QueryLogger) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	rid := p.queryId()
	if p.Trace {
		queryStart(ctx, rid, query, args...)
	}

	t := time.Now()
	result := p.Ext.QueryRowContext(ctx, query, args...)
	p.queryTime(ctx, rid, t, query, args...)

	return result
}

// QueryRowx .
func (p *QueryLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	return p.QueryRowxContext(context.Background(), query, args...)
}

func (p *QueryLogger) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	rid := p.queryId()
	if p.Trace {
		queryStart(ctx, rid, query, args...)
	}

	t := time.Now()
	result := p.Ext.QueryRowxContext(ctx, query, args...)
	p.queryTime(ctx, rid, t, query, args...)

	return result
}

type canBeginx interface {
	Beginx() (*sqlx.Tx, error)
}

// Beginx .
func (p *QueryLogger) Beginx() (*sqlx.Tx, error) {
	if a, ok := p.Ext.(canBeginx); ok {
		return a.Beginx()
	}
	return nil, errors.New("not Beginxer")
}

func (p *QueryLogger) queryId() int {
	a := atomic.AddUint64(&queryCounter, 1)
	return int(a)
}

//////

var qstrRex = regexp.MustCompile(`[\s]+`)

// QueryStart logs database query beginnings; requires TRACE.
func queryStart(ctx context.Context, rid int, qstr string, a ...interface{}) {
	log.TraceCheck(func() {
		sts := []string{}
		for i, val := range a {
			q := qval{strconv.Itoa(i + 1), val}
			sts = append(sts, q.String())
		}
		qstr = qstrRex.ReplaceAllString(qstr, " ")
		log.For(ctx).Trace().Int("query_id", rid).Str("query", qstr).Strs("query_args", sts).Msg("query: begin")
	})
}

// QueryTime logs database queries and time relative to start; requires LogQuery or TRACE.
func (p *QueryLogger) queryTime(ctx context.Context, rid int, t time.Time, qstr string, a ...interface{}) {
	duration := time.Since(t)

	// Only log if duration exceeds threshold (or threshold is zero for all queries)
	if p.LongQueryDuration > 0 && duration < p.LongQueryDuration {
		return
	}

	log.TraceCheck(func() {
		t2 := float64(duration.Nanoseconds()) / 1e6
		sts := []string{}
		for i, val := range a {
			q := qval{strconv.Itoa(i + 1), val}
			sts = append(sts, q.String())
		}
		qstr = qstrRex.ReplaceAllString(qstr, " ")
		log.For(ctx).Trace().Int("query_id", rid).Str("query", qstr).Strs("query_args", sts).Float64("queryTime", t2).Msg("query: complete")
	})
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
