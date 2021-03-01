package tests

import (
	"testing"

	"github.com/interline-io/transitland-lib/tl"
)

func Test_StringToSeconds(t *testing.T) {
	expect := map[string]int{
		"00:00:00":               0,
		"01:02:03":               3723,
		"01:02":                  3720,
		"01":                     3600,
		"":                       0,
		"2562047788015215:30:07": 1<<63 - 1,
		"2562047788015215:30:08": -(1 << 63),
	}
	for k, v := range expect {
		s, err := tl.StringToSeconds(k)
		if s != v || err != nil {
			t.Error("expected seconds", v, "for", k, "got", s, "; error:", err)
		}
	}
	// Errors
	errs := map[string]int{
		"01:61:00":    0,
		"a:b:c":       0,
		"01:02:03:04": 0,
	}
	for k, v := range errs {
		s, err := tl.StringToSeconds(k)
		if s != v || err == nil {
			t.Error("expected seconds", v, "for", k, "got", s, "; error:", err)
		}
	}
}

func TestNewWideTime(t *testing.T) {
	if wt, err := tl.NewWideTime("01:02:03"); wt.Seconds != 3723 || err != nil {
		t.Error(err)
	}
	if wt, err := tl.NewWideTime("a:b:c"); wt.Seconds != 0 || err == nil {
		t.Error("expected error")
	}
}

func TestWideTime_String(t *testing.T) {
	expect := map[string]int{
		"01:02:03": 3723,
		"01:02:00": 3720,
		"01:00:00": 3600,
		"00:00:00": 0,
	}
	for k, v := range expect {
		wt, err := tl.NewWideTime(k)
		if wt.Seconds != v {
			t.Errorf("expected %d, got %d", v, wt.Seconds)
		}
		if err != nil {
			t.Error(err)
			continue
		}
		s := wt.String()
		if s != k {
			t.Errorf("expected %s, got %s", k, s)
		}
	}
}

var result int

func Benchmark_StringToSeconds(b *testing.B) {
	for n := 0; n < b.N; n++ {
		tl.StringToSeconds("12:34:56")
	}
}

func Benchmark_NewWideTime(b *testing.B) {
	for n := 0; n < b.N; n++ {
		tl.NewWideTime("12:34:56")
	}
}
