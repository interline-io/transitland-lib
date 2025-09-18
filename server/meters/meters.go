package meters

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MeterEvent represents a metered event with a name, value, timestamp, dimensions, and request ID.
type MeterEvent struct {
	EventID    string
	Name       string
	Value      float64
	Timestamp  time.Time
	Dimensions Dimensions
	RequestID  string // Request ID associated with the event, useful for tracing
	StatusCode int    // HTTP status code of the request that generated this event
	Success    bool   // Indicates if the event was successful (e.g., HTTP status < 400)
}

// NewMeterEvent creates a new MeterEvent with the current time in UTC.
func NewMeterEvent(name string, value float64, dims Dimensions) MeterEvent {
	return MeterEvent{
		EventID:    uuid.New().String(),
		Name:       name,
		Value:      value,
		Timestamp:  time.Now().In(time.UTC),
		Dimensions: dims,
	}
}

// MeterRecorder is an interface for recording metered events.
// ApplyDimension mutates the MeterRecorder and adds a dimension that will be applied to all future Meter calls.
type MeterRecorder interface {
	Meter(context.Context, MeterEvent) error
	ApplyDimension(key, value string)
}

// MeterReader is an interface for reading metered values and checking rate limits.
type MeterReader interface {
	GetValue(context.Context, string, time.Time, time.Time, Dimensions) (float64, bool)
	Check(context.Context, string, float64, Dimensions) (bool, error)
}

// Meterer combines both MeterReader and MeterRecorder interfaces.
type Meterer interface {
	MeterReader
	MeterRecorder
}

// MeterProvider is an interface for creating new Meterers.
// It also provides methods for closing the provider and flushing any buffered data.
// The NewMeter method takes a MeterUser, which provides user-specific context for metering.
// The Close method is used to clean up resources, and Flush is used to ensure all data is written out.
type MeterProvider interface {
	// NewMeter creates a new Meterer for the given MeterUser.
	NewMeter(MeterUser) Meterer
	// Close and Flush are used to clean up resources and ensure all data is written out.
	Close() error
	// Flush is used to ensure all buffered data is written out.
	Flush() error
}

// MeterUser is an interface representing a user for metering purposes.
// It provides an ID method to get the user's identifier and a GetExternalData method
// to retrieve external data associated with the user, which can be used for metering purposes.
type MeterUser interface {
	ID() string
	GetExternalData(string) (string, bool)
}

// InjectContext adds a MeterRecorder to the context, allowing it to be retrieved later.
func InjectContext(ctx context.Context, m MeterRecorder) context.Context {
	return context.WithValue(ctx, meterCtxKey, m)
}

// ForContext retrieves the MeterRecorder from the context.
func ForContext(ctx context.Context) MeterRecorder {
	raw, _ := ctx.Value(meterCtxKey).(MeterRecorder)
	return raw
}

var meterCtxKey = struct{ name string }{"apiMeter"}

// Dimension represents a key-value pair used for metering dimensions.
type Dimension struct {
	Key   string
	Value string
}

type Dimensions []Dimension

// DimsContains checks if the given dimensions contain all the dimensions in checkDims.
func DimsContainedIn(checkDims Dimensions, eventDims Dimensions) bool {
	for _, matchDim := range checkDims {
		match := false
		for _, ed := range eventDims {
			if ed.Key == matchDim.Key && ed.Value == matchDim.Value {
				match = true
			}
		}
		if !match {
			return false
		}
	}
	return true
}

// PeriodSpan returns the start and end time for a given period.
func PeriodSpan(period string) (time.Time, time.Time, error) {
	now := time.Now().In(time.UTC)
	d1 := now
	d2 := now
	if period == "hourly" {
		d1 = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
		d2 = d1.Add(3600 * time.Second)
	} else if period == "daily" {
		d1 = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		d2 = d1.AddDate(0, 0, 1)
	} else if period == "monthly" {
		d1 = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		d2 = d1.AddDate(0, 1, 0)
	} else if period == "yearly" {
		d1 = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		d2 = d1.AddDate(1, 0, 0)
	} else if period == "total" {
		d1 = time.Unix(0, 0)
		d2 = time.Unix(1<<63-1, 0)
	} else {
		return now, now, fmt.Errorf("unknown period: %s", period)
	}
	return d1, d2, nil
}
