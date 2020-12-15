package tl

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func slowStringToSeconds(value string) (int, error) {
	t := strings.SplitN(value, ":", 3)
	switch len(t) {
	case 3: // ok
	case 2:
		t = append(t, "0")
	case 1:
		t = append(t, "0", "0")
	}
	a, ae := strconv.Atoi(t[0])
	b, be := strconv.Atoi(t[1])
	c, ce := strconv.Atoi(t[2])
	if ae != nil || be != nil || ce != nil {
		return 0, errors.New("error parsing time")
	}
	if b > 60 || c > 60 {
		return 0, errors.New("hours and mins must be 0 - 60")
	}
	return int(a*3600 + b*60 + c), nil
}

// SecondsToString takes seconds-since-midnight and returns a GTFS-style time.
func SecondsToString(secs int) string {
	if secs < 0 {
		return ""
	}
	if secs > 1<<31 {
		return ""
	}
	hours := secs / 3600
	minutes := (secs % 3600) / 60
	seconds := (secs % 3600) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// StringToSeconds parses a GTFS-style time and returns seconds since midnight.
func StringToSeconds(value string) (int, error) {
	if len(value) == 0 {
		return 0, nil
	} else if len(value) == 7 {
		value = "0" + value
	} else if len(value) != 8 {
		return slowStringToSeconds(value)
	}
	// fast path, avoiding strings.Split (6x faster)
	a, ae := strconv.Atoi(value[0:2])
	b, be := strconv.Atoi(value[3:5])
	c, ce := strconv.Atoi(value[6:8])
	if ae != nil || be != nil || ce != nil {
		// fallback if errors
		return slowStringToSeconds(value)
	}
	if b > 60 || c > 60 {
		return 0, errors.New("hours and mins must be 0 - 60")
	}
	return int(a*3600 + b*60 + c), nil
}

// WideTime handles seconds since midnight, allows >24 hours.
type WideTime struct {
	Seconds int
	Valid   bool
}

func (wt *WideTime) String() string {
	return SecondsToString(wt.Seconds)
}

// Value implements driver.Value
func (wt WideTime) Value() (driver.Value, error) {
	return int64(wt.Seconds), nil
}

// Scan implements sql.Scanner
func (wt *WideTime) Scan(src interface{}) error {
	wt.Valid = false
	var p error
	switch v := src.(type) {
	case string:
		if s, err := StringToSeconds(v); err == nil {
			wt.Seconds = s
		} else {
			p = err
		}
	case int:
		wt.Seconds = v
	case int64:
		wt.Seconds = int(v)
	default:
		p = errors.New("could not parse time")
	}
	if p == nil {
		wt.Valid = true
	}
	return p
}

// NewWideTime converts the csv string to a WideTime.
func NewWideTime(value string) (wt WideTime, err error) {
	a, err := StringToSeconds(value)
	if err != nil {
		return wt, err
	}
	wt.Seconds = a
	return wt, nil
}
