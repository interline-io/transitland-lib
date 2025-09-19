package local

import (
	"context"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/meters"
)

func init() {
	var _ meters.MeterProvider = &LocalMeterProvider{}
}

type LocalMeterProvider struct {
	values map[string]localMeterUserEvents
	lock   sync.Mutex
}

func NewLocalMeterProvider() *LocalMeterProvider {
	return &LocalMeterProvider{
		values: map[string]localMeterUserEvents{},
	}
}

func (m *LocalMeterProvider) Flush() error {
	return nil
}

func (m *LocalMeterProvider) Close() error {
	return nil
}

func (m *LocalMeterProvider) NewMeter(user meters.MeterUser) meters.Meterer {
	return &localUserMeter{
		user: user,
		mp:   m,
	}
}

func (m *LocalMeterProvider) sendMeter(u meters.MeterUser, meterEvent meters.MeterEvent) error {
	if u == nil {
		return nil
	}
	userName := u.ID()

	m.lock.Lock()
	defer m.lock.Unlock()
	a, ok := m.values[meterEvent.Name]
	if !ok {
		a = localMeterUserEvents{}
		m.values[meterEvent.Name] = a
	}

	event := localMeterEvent{
		value: meterEvent.Value,
		dims:  meterEvent.Dimensions,
		time:  time.Now().In(time.UTC),
	}
	a[userName] = append(a[userName], event)
	log.TraceCheck(func() {
		lm := log.Trace().
			Str("user", userName).
			Str("meter", meterEvent.Name).
			Float64("meter_value", meterEvent.Value)
		for _, dim := range meterEvent.Dimensions {
			lm = lm.Str("dim:"+dim.Key, dim.Value)
		}
		lm.Msg("meter")
	})
	return nil
}

func (m *LocalMeterProvider) getValue(u meters.MeterUser, meterName string, startTime time.Time, endTime time.Time, checkDims meters.Dimensions) (float64, bool) {
	if u == nil {
		return 0, false
	}
	userName := u.ID()
	m.lock.Lock()
	defer m.lock.Unlock()
	a, ok := m.values[meterName]
	if !ok {
		return 0, false
	}
	total := 0.0
	for _, userEvent := range a[userName] {
		match := true
		if userEvent.time.Equal(endTime) || userEvent.time.After(endTime) {
			// fmt.Println("not matched on end time", userEvent.time, endTime)
			match = false
		}
		if userEvent.time.Before(startTime) {
			// fmt.Println("not matched on start time", userEvent.time, startTime)
			match = false
		}
		if !meters.DimsContainedIn(checkDims, userEvent.dims) {
			// fmt.Println("not matched on dims")
			match = false
		}
		if match {
			// fmt.Println("matched:", userEvent.value)
			total += userEvent.value
		}
	}
	return total, ok
}

type eventAddDim struct {
	Key   string
	Value string
}

type localUserMeter struct {
	user    meters.MeterUser
	addDims []eventAddDim
	mp      *LocalMeterProvider
}

func (m *localUserMeter) Meter(ctx context.Context, meterEvent meters.MeterEvent) error {
	// Copy in matching dimensions set through AddDimension
	var eventDims []meters.Dimension
	eventDims = append(eventDims, meterEvent.Dimensions...)
	for _, addDim := range m.addDims {
		eventDims = append(eventDims, meters.Dimension{Key: addDim.Key, Value: addDim.Value})
	}
	meterEvent.Dimensions = eventDims
	return m.mp.sendMeter(m.user, meterEvent)
}

func (m *localUserMeter) ApplyDimension(key string, value string) {
	m.addDims = append(m.addDims, eventAddDim{Key: key, Value: value})
}

func (m *localUserMeter) GetValue(ctx context.Context, meterName string, startTime time.Time, endTime time.Time, dims meters.Dimensions) (float64, bool) {
	return m.mp.getValue(m.user, meterName, startTime, endTime, dims)
}

func (m *localUserMeter) Check(ctx context.Context, meterName string, value float64, dims meters.Dimensions) (bool, error) {
	return true, nil
}

///////////

type localMeterEvent struct {
	time  time.Time
	dims  []meters.Dimension
	value float64
}

type localMeterUserEvents map[string][]localMeterEvent
