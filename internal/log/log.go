package log

import (
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gookit/color"
)

// Level values
const (
	FATAL    = 100
	CRITICAL = 50
	ERROR    = 40
	WARNING  = 30
	INFO     = 20
	QUERY    = 11
	DEBUG    = 10
	TRACE    = 5
)

// LEVELSTRINGS provides log level aliases.
var LEVELSTRINGS = map[string]int{
	"CRITICAL": CRITICAL,
	"ERROR":    ERROR,
	"WARNING":  WARNING,
	"INFO":     INFO,
	"DEBUG":    DEBUG,
	"TRACE":    TRACE,
	"QUERY":    QUERY,
}

// STRINGLEVEL is the reverse mapping
var STRINGLEVEL = map[int]string{}

func init() {
	for k, v := range LEVELSTRINGS {
		STRINGLEVEL[v] = k
	}
}

// Level is the log level.
var Level = DEBUG

// Printf is the same as Info.
func Printf(fmt string, a ...interface{}) {
	logLog(INFO, fmt, a...)
}

// Println is for compatibility.
func Println(a ...interface{}) {
	log.Println(a...)
}

// Info for regular messages.
func Info(fmt string, a ...interface{}) {
	logLog(INFO, fmt, a...)
}

// Debug for debugging messages.
func Debug(fmt string, a ...interface{}) {
	logLog(DEBUG, fmt, a...)
}

// Trace for really deep debugging.
func Trace(fmt string, a ...interface{}) {
	logLog(TRACE, fmt, a...)
}

// Fatal for fatal, unrecoverable errors.
func Fatal(fmta string, a ...interface{}) {
	logLog(FATAL, fmta, a...)
	panic(fmt.Sprintf(fmta, a...))
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
		s = fmt.Sprintf("%s", z)
	} else {
		s = fmt.Sprintf("%s", q.Value)
	}
	return fmt.Sprintf("{%s:%s}", q.Name, s)
}

// Query for logging database queries.
func Query(qstr string, a ...interface{}) {
	sts := []string{}
	for i, val := range a {
		q := qval{strconv.Itoa(i + 1), val}
		sts = append(sts, q.String())
	}
	fmta := qstr
	logLog(QUERY, color.Blue.Render(fmta)+" -- "+color.Gray.Render(strings.Join(sts, " ")))
}

// Sq for logging Squirrel Queries; avoids ToSql evaluation unless log level.
func Sq(q canToSQL) {
	level := DEBUG
	if level < Level {
		return
	}
	qstr, qargs, err := q.ToSql()
	if err != nil {
		Query("error building query: %s", err)
		return
	}
	Query(qstr, qargs...)
}

func logLog(level int, fmt string, a ...interface{}) {
	strlevel, _ := STRINGLEVEL[level]
	if level >= Level {
		log.Printf("["+strlevel+"] "+fmt, a...)
	}
}

// SetLevel sets the log level.
func SetLevel(level int) {
	Level = level
}

// SetLevelString uses a string alias to set the log level.
func SetLevelString(lstr string) {
	lvalue, ok := LEVELSTRINGS[strings.ToUpper(lstr)]
	if !ok {
		lvalue = 20
	}
	SetLevel(lvalue)
}

func init() {
	log.SetOutput(os.Stdout)
	SetLevelString(os.Getenv("GTFS_LOGLEVEL"))
}
