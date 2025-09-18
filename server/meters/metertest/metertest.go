package metertest

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/meters"
	"github.com/stretchr/testify/assert"
)

type MeterUser = meters.MeterUser
type MeterProvider = meters.MeterProvider
type Dimension = meters.Dimension

type TestUser struct {
	name string
	data map[string]string
}

func NewTestUser(name string, data map[string]string) TestUser {
	return TestUser{
		name: name,
		data: data,
	}
}

func (u TestUser) ID() string {
	return u.name
}

func (u TestUser) GetExternalData(key string) (string, bool) {
	if u.data == nil {
		return "", false
	}
	a, ok := u.data[key]
	return a, ok
}

func checkErr(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

func checkOk(t *testing.T, ok bool) {
	if !ok {
		t.Error("expected true")
	}
}

type Config struct {
	TestMeter1 string
	TestMeter2 string
	User1      MeterUser
	User2      MeterUser
	User3      MeterUser
}

func TestMeter(t *testing.T, mp MeterProvider, cfg Config) {
	ctx := context.Background()
	d1, d2, _ := meters.PeriodSpan("hourly")
	t.Run("Meter", func(t *testing.T) {
		m := mp.NewMeter(cfg.User1)
		v, _ := m.GetValue(ctx, cfg.TestMeter1, d1, d2, nil)

		checkErr(t, m.Meter(ctx, meters.NewMeterEvent(cfg.TestMeter1, 1, nil)))
		mp.Flush()

		a, ok := m.GetValue(ctx, cfg.TestMeter1, d1, d2, nil)
		checkOk(t, ok)
		assert.Equal(t, 1.0, a-v)

		checkErr(t, m.Meter(ctx, meters.NewMeterEvent(cfg.TestMeter1, 1, nil)))
		mp.Flush()

		b, ok := m.GetValue(ctx, cfg.TestMeter1, d1, d2, nil)
		checkOk(t, ok)
		assert.Equal(t, 2.0, b-v)
	})
	t.Run("NewMeter", func(t *testing.T) {
		m1 := mp.NewMeter(cfg.User1)

		v1, _ := m1.GetValue(ctx, cfg.TestMeter1, d1, d2, nil)
		v2, _ := m1.GetValue(ctx, cfg.TestMeter2, d1, d2, nil)

		checkErr(t, m1.Meter(ctx, meters.NewMeterEvent(cfg.TestMeter1, 1, nil)))
		checkErr(t, m1.Meter(ctx, meters.NewMeterEvent(cfg.TestMeter2, 2, nil)))
		mp.Flush()

		va1, ok := m1.GetValue(ctx, cfg.TestMeter1, d1, d2, nil)
		checkOk(t, ok)
		assert.Equal(t, 1.0, va1-v1)
		va2, ok := m1.GetValue(ctx, cfg.TestMeter2, d1, d2, nil)
		checkOk(t, ok)
		assert.Equal(t, 2.0, va2-v2)
	})
	t.Run("GetValue", func(t *testing.T) {
		m1 := mp.NewMeter(cfg.User1)
		m2 := mp.NewMeter(cfg.User2)
		m3 := mp.NewMeter(cfg.User3)
		v1, _ := m1.GetValue(ctx, cfg.TestMeter1, d1, d2, nil)
		v2, _ := m2.GetValue(ctx, cfg.TestMeter1, d1, d2, nil)
		v3, _ := m3.GetValue(ctx, cfg.TestMeter1, d1, d2, nil)

		checkErr(t, m1.Meter(ctx, meters.NewMeterEvent(cfg.TestMeter1, 1, nil)))
		checkErr(t, m2.Meter(ctx, meters.NewMeterEvent(cfg.TestMeter1, 2.0, nil)))
		mp.Flush()

		a, ok := m1.GetValue(ctx, cfg.TestMeter1, d1, d2, nil)
		checkOk(t, ok)
		assert.Equal(t, 1.0, a-v1)
		assert.Equal(t, true, ok)

		a, ok = m2.GetValue(ctx, cfg.TestMeter1, d1, d2, nil)
		checkOk(t, ok)
		assert.Equal(t, 2.0, a-v2)
		assert.Equal(t, true, ok)

		a, ok = m3.GetValue(ctx, cfg.TestMeter1, d1, d2, nil)
		checkOk(t, ok)
		assert.Equal(t, 0.0, a-v3)
	})

	t.Run("GetValue match dims", func(t *testing.T) {
		addDims1 := []Dimension{{Key: "test", Value: "a"}, {Key: "other", Value: "boo"}}
		addDims2 := []Dimension{{Key: "test", Value: "b"}}
		checkDims1 := []Dimension{{Key: "test", Value: "a"}}
		checkDims2 := []Dimension{{Key: "test", Value: "b"}}

		m1 := mp.NewMeter(cfg.User1)
		m2 := mp.NewMeter(cfg.User2)
		m3 := mp.NewMeter(cfg.User3)

		// Initial values
		v1, _ := m1.GetValue(ctx, cfg.TestMeter1, d1, d2, checkDims1)
		v2, _ := m2.GetValue(ctx, cfg.TestMeter1, d1, d2, checkDims2)
		v3, _ := m3.GetValue(ctx, cfg.TestMeter1, d1, d2, checkDims1)

		// m1 meter
		checkErr(t, m1.Meter(ctx, meters.NewMeterEvent(cfg.TestMeter1, 1, addDims1)))
		// m2 uses different dimension
		checkErr(t, m2.Meter(ctx, meters.NewMeterEvent(cfg.TestMeter1, 2.0, addDims2)))
		mp.Flush()

		a, ok := m1.GetValue(ctx, cfg.TestMeter1, d1, d2, checkDims1)
		assert.Equal(t, 1.0, a-v1)
		assert.Equal(t, true, ok)

		a, ok = m2.GetValue(ctx, cfg.TestMeter1, d1, d2, checkDims1)
		assert.Equal(t, 0.0, a)
		assert.Equal(t, true, ok)

		a, ok = m2.GetValue(ctx, cfg.TestMeter1, d1, d2, checkDims2)
		assert.Equal(t, 2.0, a-v2)
		assert.Equal(t, true, ok)

		a, _ = m3.GetValue(ctx, cfg.TestMeter1, d1, d2, checkDims1)
		assert.Equal(t, 0.0, a-v3)
	})
}

// TestMeterWrite is a helper function for testing the MeterProvider interface for writing.
// It only tests that writes are successful and do not generate errors.
func TestMeterWrite(t *testing.T, mp MeterProvider, cfg Config) {
	ctx := context.Background()
	t.Run("Meter", func(t *testing.T) {
		m := mp.NewMeter(cfg.User1)
		checkErr(t, m.Meter(ctx, meters.NewMeterEvent(cfg.TestMeter1, 1, nil)))
		mp.Flush()
	})
	t.Run("NewMeter", func(t *testing.T) {
		m1 := mp.NewMeter(cfg.User1)
		checkErr(t, m1.Meter(ctx, meters.NewMeterEvent(cfg.TestMeter1, 1, nil)))
		checkErr(t, m1.Meter(ctx, meters.NewMeterEvent(cfg.TestMeter2, 2, nil)))
		mp.Flush()
	})
}
