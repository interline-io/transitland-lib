package log

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Level values
const (
	FATAL    = 100
	CRITICAL = 50
	ERROR    = 40
	WARNING  = 30
	INFO     = 20
	DEBUG    = 10
	QUERY    = 9
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
var Level = ERROR

// LogQuery is a flag for logging database queries.
var LogQuery = false

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

// Query for really deep debugging.
func Query(fmt string, a ...interface{}) {
	logLog(QUERY, fmt, a...)
}

// Fatal for fatal, unrecoverable errors.
func Fatal(fmta string, a ...interface{}) {
	logLog(FATAL, fmta, a...)
	panic(fmt.Sprintf(fmta, a...))
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
	if ok {
		SetLevel(lvalue)
	}
}

func init() {
	log.SetOutput(os.Stdout)
	SetLevelString(os.Getenv("GTFS_LOGLEVEL"))
	if v := os.Getenv("GTFS_LOGLEVEL_SQL"); v == "true" {
		LogQuery = true
	}
}
