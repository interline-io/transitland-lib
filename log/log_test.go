package log

import (
	"errors"
	"testing"
)

func TestInfof(t *testing.T) {
	Infof("infof")
}

func TestErrorf(t *testing.T) {
	Errorf("errorf")
}

func TestError(t *testing.T) {
	Error().Err(errors.New("test")).Msgf("error")
}

func TestWith(t *testing.T) {
	l := With().Str("with", "ok").Logger()
	l.Info().Msgf("with ok")
}
