package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// A date's local noon exists even on a day whose midnight a DST change skips:
// America/Santiago jumped from midnight to 01:00 on 2024-09-08.
func TestTzTruncate(t *testing.T) {
	santiago, err := time.LoadLocation("America/Santiago")
	if err != nil {
		t.Fatal(err)
	}
	got := tzTruncate(time.Date(2024, 9, 8, 0, 0, 0, 0, time.UTC), santiago)
	assert.Equal(t, "2024-09-08T12:00:00-03:00", got.Format(time.RFC3339))
}
