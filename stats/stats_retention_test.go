package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOnestopIDsRetained(t *testing.T) {
	day := 24 * time.Hour
	cases := []struct {
		name      string
		retention int
		age       time.Duration
		want      bool
	}{
		{name: "never generate (-1)", retention: -1, age: 0, want: false},
		{name: "never generate ignores age", retention: -1, age: 100 * day, want: false},
		{name: "keep forever (0), new", retention: 0, age: 0, want: true},
		{name: "keep forever (0), old", retention: 0, age: 5000 * day, want: true},
		{name: "within window", retention: 365, age: 100 * day, want: true},
		{name: "past window", retention: 365, age: 400 * day, want: false},
		{name: "exactly at window is past", retention: 7, age: 7 * day, want: false},
		{name: "just inside window", retention: 7, age: 7*day - time.Hour, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, OnestopIDsRetained(tc.retention, tc.age))
		})
	}
}
