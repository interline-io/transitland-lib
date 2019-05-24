package log

import (
	"fmt"
	"log"
	"os"
)

// LEVELSTRINGS provides log level aliases.
var LEVELSTRINGS = map[string]int{
	"critical": 50,
	"error":    40,
	"warning":  30,
	"info":     20,
	"debug":    10,
	"trace":    5,
}

// Level is the log level.
var Level = 10

// Printf is the same as Info.
func Printf(fmt string, a ...interface{}) {
	logLog(20, fmt, a...)
}

// Println is for compatibility.
func Println(a ...interface{}) {
	log.Println(a...)
}

// Info for regular messages.
func Info(fmt string, a ...interface{}) {
	logLog(20, fmt, a...)
}

// Debug for debugging messages.
func Debug(fmt string, a ...interface{}) {
	logLog(10, fmt, a...)
}

// Trace for really deep debugging.
func Trace(fmt string, a ...interface{}) {
	logLog(5, fmt, a...)
}

// Fatal for fatal, unrecoverable errors.
func Fatal(fmta string, a ...interface{}) {
	logLog(100, fmta, a...)
	panic(fmt.Sprintf(fmta, a...))
}

func logLog(level int, fmt string, a ...interface{}) {
	if level >= Level {
		log.Printf(fmt, a...)
	}
}

// SetLevel sets the log level.
func SetLevel(level int) {
	Level = level
}

// SetLevelString uses a string alias to set the log level.
func SetLevelString(lstr string) {
	lvalue, ok := LEVELSTRINGS[lstr]
	if !ok {
		lvalue = 20
	}
	SetLevel(lvalue)
}

func init() {
	log.SetOutput(os.Stdout)
	SetLevelString(os.Getenv("GTFS_LOGLEVEL"))
}
