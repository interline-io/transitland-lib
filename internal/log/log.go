package log

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Level values
const (
	FATAL = 100
	ERROR = 40
	INFO  = 20
	DEBUG = 10
	QUERY = 5
)

// LEVELSTRINGS provides log level aliases.
var LEVELSTRINGS = map[string]int{
	"FATAL": FATAL,
	"ERROR": ERROR,
	"INFO":  INFO,
	"DEBUG": DEBUG,
	"QUERY": QUERY,
	"TRACE": QUERY, // alias
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

// Error for notable errors.
func Error(fmt string, a ...interface{}) {
	logLog(ERROR, fmt, a...)
}

// Info for regular messages.
func Info(fmt string, a ...interface{}) {
	logLog(INFO, fmt, a...)
}

// Debug for debugging messages.
func Debug(fmt string, a ...interface{}) {
	logLog(DEBUG, fmt, a...)
}

// Query for printing database queries and statistics.
func Query(fmt string, a ...interface{}) {
	logLog(QUERY, fmt, a...)
}

// Fatal for fatal, unrecoverable errors.
func Fatal(fmta string, a ...interface{}) {
	logLog(FATAL, fmta, a...)
	panic(fmt.Sprintf(fmta, a...))
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
	Level = lvalue
}

func init() {
	lstr := strings.ToUpper(os.Getenv("GTFS_LOGLEVEL"))
	if lstr == "" {
		lstr = "INFO"
	}
	lvalue, ok := LEVELSTRINGS[strings.ToUpper(lstr)]
	if ok {
		SetLevel(lvalue)
	} else {
		log.Printf("[WARNING] Unknown log level '%s'", lstr)
	}
	if v := os.Getenv("GTFS_LOGLEVEL_SQL"); v == "true" {
		LogQuery = true
	}
}
