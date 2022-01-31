package log

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Error for notable errors.
func Error(fmts string, a ...interface{}) {
	log.Error().Msgf(fmts, a...)
}

// Info for regular messages.
func Info(fmts string, a ...interface{}) {
	log.Info().Msgf(fmts, a...)
}

// Debug for debugging messages.
func Debug(fmts string, a ...interface{}) {
	log.Debug().Msgf(fmts, a...)
}

// Trace for debugging messages.
func Trace(fmts string, a ...interface{}) {
	log.Trace().Msgf(fmts, a...)
}

// Exit with an error message.
func Exit(fmts string, args ...interface{}) {
	Print(fmts, args...)
	os.Exit(1)
}

// Print - simple print, without timestamp, without regard to log level.
func Print(fmts string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, fmts+"\n", args...)
}

// SetLevel sets the log level.
func SetLevel(lvalue zerolog.Level) {
	zerolog.SetGlobalLevel(lvalue)
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
	log.Logger = log.Output(output)
}

func init() {
	if os.Getenv("TL_LOG_JSON") == "true" {
		// use json logging
	} else {
		setConsoleLogger()
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if v := os.Getenv("TL_LOG"); v != "" {
		setLevelByName(v)
	}
}
