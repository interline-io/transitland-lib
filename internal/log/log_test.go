package log

import (
	"bytes"
	"log"
	"os"
	"testing"
)

var prefix = "2019/03/19 17:08:58  "
var msg = "test %d"

func TestPrintln(t *testing.T) {
	lv := Level
	buf := bytes.NewBufferString("")
	Level = 20
	log.SetOutput(buf)
	Println("test", "ok")
	a := 7
	b := len(buf.String())
	if a >= b {
		t.Errorf("expected at least %d characters, got %d", a, b)
	}
	log.SetOutput(os.Stdout)
	Level = lv
}

func TestFatal(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// ok
		} else {
			t.Error("expected to recover from Fatal")
		}
	}()
	Fatal(msg, 123)
}

func TestLogLevels(t *testing.T) {
	funcs := []struct {
		name  string
		level int
		f     func(string, ...interface{})
	}{
		{"Info", 20, Info},
		{"Debug", 10, Debug},
		{"Trace", 5, Trace},
		{"Printf", 0, Printf},
	}
	for _, f := range funcs {
		t.Run(f.name, func(t *testing.T) {
			_ = f
			lv := Level
			buf := bytes.NewBufferString("")
			log.SetOutput(buf)
			Level = f.level
			// z := Info
			f.f(msg, 123)
			a := len(msg)
			b := len(buf.String())
			if a >= b {
				t.Errorf("expected at least %d characters, got %d", a, b)
			}
			log.SetOutput(os.Stdout)
			Level = lv
		})
	}
}

func Test_logLog(t *testing.T) {
	levels := []struct {
		name  string
		level int
	}{
		{"critical", 50},
		{"error", 40},
		{"warning", 30},
		{"info", 20},
		{"debug", 10},
		{"trace", 5},
	}
	for _, level := range levels {
		t.Run(level.name, func(t *testing.T) {
			lv := Level
			buf := bytes.NewBufferString("")
			log.SetOutput(buf)
			Level = level.level
			logLog(level.level, msg, 123)
			a := len(msg)
			b := len(buf.String())
			if a >= b {
				t.Errorf("expected at least %d characters, got %d", a, b)
			}
			log.SetOutput(os.Stdout)
			Level = lv
		})
	}
}
