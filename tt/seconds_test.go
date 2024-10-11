package tt

import (
	"testing"
)

func Test_StringToSeconds(t *testing.T) {
	expect := map[string]int64{
		"00:00:00":               0,
		"01:02:03":               3723,
		"01:02":                  3720,
		"01":                     3600,
		"":                       0,
		"2562047788015215:30:07": 1<<63 - 1,
		"2562047788015215:30:08": -(1 << 63),
	}
	for k, v := range expect {
		s, err := StringToSeconds(k)
		if s != v || err != nil {
			t.Error("expected seconds", v, "for", k, "got", s, "; error:", err)
		}
	}
	// Errors
	errs := map[string]int64{
		"01:61:00":    0,
		"a:b:c":       0,
		"01:02:03:04": 0,
	}
	for k, v := range errs {
		s, err := StringToSeconds(k)
		if s != v || err == nil {
			t.Error("expected seconds", v, "for", k, "got", s, "; error:", err)
		}
	}
}

func TestNewWideTime(t *testing.T) {
	if wt, err := NewSecondsFromString("01:02:03"); wt.Int() != 3723 || err != nil {
		t.Error(err)
	}
	if wt, err := NewSecondsFromString("a:b:c"); wt.Int() != 0 || err == nil {
		t.Error("expected error")
	}
}

func TestWideTime_String(t *testing.T) {
	expect := map[string]int64{
		"01:02:03": 3723,
		"01:02:00": 3720,
		"01:00:00": 3600,
		"00:00:00": 0,
	}
	for k, v := range expect {
		wt, err := NewSecondsFromString(k)
		if wt.Val != v {
			t.Errorf("expected %d, got %d", v, wt.Val)
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

func Benchmark_StringToSeconds(b *testing.B) {
	for n := 0; n < b.N; n++ {
		StringToSeconds("12:34:56")
	}
}

func Benchmark_NewWideTime(b *testing.B) {
	for n := 0; n < b.N; n++ {
		NewSecondsFromString("12:34:56")
	}
}
