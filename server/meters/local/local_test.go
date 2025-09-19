package local

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/meters/metertest"
)

func TestLocalMeter(t *testing.T) {
	mp := NewLocalMeterProvider()
	testConfig := metertest.Config{
		TestMeter1: "test1",
		TestMeter2: "test2",
		User1:      metertest.NewTestUser("test1", nil),
		User2:      metertest.NewTestUser("test2", nil),
		User3:      metertest.NewTestUser("test3", nil),
	}
	metertest.TestMeter(t, mp, testConfig)
}
