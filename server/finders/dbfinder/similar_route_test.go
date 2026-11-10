package dbfinder

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServiceInstant(t *testing.T) {
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2018, 6, 4, 0, 0, 0, 0, time.UTC)
	tuesday := time.Date(2018, 6, 5, 0, 0, 0, 0, time.UTC)
	hms := func(h, m int) int { return h*3600 + m*60 }
	t.Run("past midnight on one date is early on the next", func(t *testing.T) {
		assert.Equal(t, serviceInstant(tuesday, hms(0, 15), la), serviceInstant(monday, hms(24, 15), la))
	})
	t.Run("the same local time in another timezone is another moment", func(t *testing.T) {
		assert.Equal(t, int64(3*3600), serviceInstant(monday, hms(8, 0), la)-serviceInstant(monday, hms(8, 0), ny))
		assert.Equal(t, serviceInstant(monday, hms(8, 0), la), serviceInstant(monday, hms(11, 0), ny))
	})
	t.Run("seconds count from noon minus twelve hours", func(t *testing.T) {
		// On the day clocks spring forward, noon minus twelve hours is 23:00 the
		// evening before, so a time after the change still reads as the clock.
		spring := time.Date(2018, 3, 11, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, time.Date(2018, 3, 11, 0, 0, 0, 0, la).Unix()-3600, serviceInstant(spring, 0, la))
		assert.Equal(t, time.Date(2018, 3, 11, 8, 0, 0, 0, la).Unix(), serviceInstant(spring, hms(8, 0), la))
	})
}
