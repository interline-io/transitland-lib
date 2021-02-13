package log

import (
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gookit/color"
)

// Level values
const (
	FATAL = 100
	ERROR = 40
	INFO  = 20
	DEBUG = 10
	TRACE = 5
)

// STRINGLEVEL is the reverse mapping
var STRINGLEVEL = map[int]string{
	FATAL: "FATAL",
	ERROR: "ERROR",
	INFO:  "INFO",
	DEBUG: "DEBUG",
	TRACE: "TRACE",
}

// Level is the log level.
var Level = ERROR

// LogQuery enables query logging regardless of level.
var LogQuery = false

// QueryStart logs database query beginnings; requires TRACE.
func QueryStart(qstr string, a ...interface{}) {
	if Level > TRACE {
		return
	}
	sts := []string{}
	for i, val := range a {
		q := qval{strconv.Itoa(i + 1), val}
		sts = append(sts, q.String())
	}
	log.Printf("[QUERY] %s -- %s [start]", color.Blue.Render(qstr), color.Gray.Render(strings.Join(sts, " ")))
}

// QueryTime logs database queries and time relative to start; requires LogQuery or TRACE.
func QueryTime(t time.Time, qstr string, a ...interface{}) {
	if !(LogQuery || Level <= TRACE) {
		return
	}
	t2 := float64(time.Now().UnixNano()-t.UnixNano()) / 1e6
	sts := []string{}
	for i, val := range a {
		q := qval{strconv.Itoa(i + 1), val}
		sts = append(sts, q.String())
	}
	log.Printf("[QUERY] %s -- %s [time: %0.2f ms]", color.Blue.Render(qstr), color.Gray.Render(strings.Join(sts, " ")), t2)
}

// Error for notable errors.
func Error(fmts string, a ...interface{}) {
	logLog(ERROR, fmts, a...)
}

// Info for regular messages.
func Info(fmts string, a ...interface{}) {
	logLog(INFO, fmts, a...)
}

// Debug for debugging messages.
func Debug(fmts string, a ...interface{}) {
	logLog(DEBUG, fmts, a...)
}

// Fatal for fatal, unrecoverable errors.
func Fatal(fmts string, a ...interface{}) {
	logLog(FATAL, fmts, a...)
	panic(fmt.Sprintf(fmts, a...))
}

// Exit with an error message.
func Exit(fmts string, args ...interface{}) {
	Print(fmts, args...)
	os.Exit(1)
}

// Print - simple print, without timestamp, without regard to log level.
func Print(fmts string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmts+"\n", args...)
}

func logLog(level int, msg string, a ...interface{}) {
	if msg == "" {
		return
	}
	strlevel, _ := STRINGLEVEL[level]
	if level >= Level {
		log.Printf("["+strlevel+"] "+msg, a...)
	}
}

// SetLevel sets the log level.
func SetLevel(lvalue int) {
	if lstr, ok := STRINGLEVEL[lvalue]; ok {
		Level = lvalue
		logLog(DEBUG, "Setting log level to '%s'", strings.ToUpper(lstr))
	} else {
		logLog(ERROR, "Unknown log level '%d'", lvalue)
	}
}

// SetLevelByName sets the log level by string name.
func SetLevelByName(lstr string) {
	var levelstrings = map[string]int{}
	for k, v := range STRINGLEVEL {
		levelstrings[strings.ToUpper(v)] = k
	}
	if lvalue, ok := levelstrings[strings.ToUpper(lstr)]; ok {
		SetLevel(lvalue)
	} else {
		logLog(ERROR, "Unknown log level string '%s'", lstr)
	}
}

// SetQueryLog enables or disables query logging.
func SetQueryLog(v bool) {
	if v {
		log.Printf("[INFO] Enabling SQL logging")
		LogQuery = true
	} else {
		LogQuery = false
	}
}

func init() {
	log.SetOutput(os.Stdout)
	if v := os.Getenv("TRANSITLAND_LOG_MICROSECONDS"); v == "true" {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	}
	if v := os.Getenv("TRANSITLAND_LOGLEVEL"); v != "" {
		SetLevelByName(v)
	}
	if v := os.Getenv("TRANSITLAND_LOG_SQL"); v == "true" {
		SetQueryLog(true)
	}
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
			z = "<binary>"
		}
		s = fmt.Sprintf("%v", z)
	} else {
		s = fmt.Sprintf("%v", q.Value)
	}
	return fmt.Sprintf("{%s:%s}", q.Name, s)
}
