package log

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Zerolog

var Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

func Fatal() *zerolog.Event {
	return Logger.Info()
}

func Info() *zerolog.Event {
	return Logger.Info()
}

func Error() *zerolog.Event {
	return Logger.Error()
}

func Debug() *zerolog.Event {
	return Logger.Debug()
}

func Trace() *zerolog.Event {
	return Logger.Trace()
}

func With() zerolog.Context {
	return Logger.With()
}

// Zerolog simple wrappers

// Error for notable errors.
func Errorf(fmts string, a ...interface{}) {
	Logger.Error().Msgf(fmts, a...)
}

// Info for regular messages.
func Infof(fmts string, a ...interface{}) {
	Logger.Info().Msgf(fmts, a...)
}

// Debug for debugging messages.
func Debugf(fmts string, a ...interface{}) {
	Logger.Debug().Msgf(fmts, a...)
}

// Trace for debugging messages.
func Tracef(fmts string, a ...interface{}) {
	Logger.Trace().Msgf(fmts, a...)
}

// Traceln - prints to trace
func Traceln(args ...interface{}) {
	Logger.Trace().Msg(fmt.Sprintln(args...))
}

// Helper functions

// Print - simple print, without timestamp, without regard to log level.
func Print(fmts string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, fmts+"\n", args...)
}

// Log init and settings

// SetLevel sets the log level.
func SetLevel(lvalue zerolog.Level) {
	zerolog.SetGlobalLevel(lvalue)
	Infof("Set global log value to %s", lvalue)
}

// setLevelByName sets the log level by string name.
func setLevelByName(lstr string) {
	switch strings.ToUpper(lstr) {
	case "FATAL":
		SetLevel(zerolog.FatalLevel)
	case "ERROR":
		SetLevel(zerolog.ErrorLevel)
	case "INFO":
		SetLevel(zerolog.InfoLevel)
	case "DEBUG":
		SetLevel(zerolog.DebugLevel)
	case "TRACE":
		SetLevel(zerolog.TraceLevel)
	}
}

func setConsoleLogger() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("[%-5s]", i))
	}
	Logger = zerolog.New(os.Stderr).With().Timestamp().Logger().Output(output).Level(zerolog.TraceLevel)
}

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if os.Getenv("TL_LOG_JSON") == "true" {
		// use json logging
	} else {
		setConsoleLogger()
	}
	if v := os.Getenv("TL_LOG"); v != "" {
		setLevelByName(v)
	}
}
