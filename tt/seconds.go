package tt

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Seconds struct {
	Option[int64]
}

func NewSecondsFromString(s string) (Seconds, error) {
	val, err := StringToSeconds(s)
	if err != nil {
		return Seconds{}, err
	}
	return Seconds{Option: NewOption(int64(val))}, nil
}

func NewSeconds(s int) Seconds {
	return Seconds{Option: NewOption(int64(s))}
}

func (r *Seconds) SetInt(v int) {
	r.Val = int64(v)
	r.Valid = true
}

func (r Seconds) HMS() (int, int, int) {
	secs := int(r.Val)
	if secs < 0 {
		secs = 0
	}
	hours := secs / 3600
	minutes := (secs % 3600) / 60
	seconds := (secs % 3600) % 60
	return hours, minutes, seconds
}

func (r Seconds) String() string {
	if !r.Valid {
		return ""
	}
	return SecondsToString(r.Val)
}

func (r Seconds) ToCsv() string {
	return r.String()
}

func (r *Seconds) Scan(src interface{}) error {
	r.Valid = false
	r.Val = 0
	var p error
	switch v := src.(type) {
	case nil:
		return nil
	case string:
		r.Val, p = StringToSeconds(v)
		r.Valid = (p == nil)
	case int:
		if v >= 0 {
			r.Val = int64(v)
			r.Valid = (p == nil)
		}
	case int64:
		if v >= 0 {
			r.Val = v
			r.Valid = true
		}
	case json.Number:
		r.Val, p = v.Int64()
		r.Valid = (p == nil)
	default:
		p = errors.New("could not parse time")
	}
	return p
}

func (r *Seconds) UnmarshalJSON(d []byte) error {
	return r.Scan(string(stripQuotes(d)))
}

func (r Seconds) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return jsonNull(), nil
	}
	return json.Marshal(r.String())
}

func (r Seconds) Int() int {
	return int(r.Val)
}

/////////////

// SecondsToString takes seconds-since-midnight and returns a GTFS-style time.
func SecondsToString(secs int64) string {
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
func StringToSeconds(value string) (int64, error) {
	if len(value) == 0 {
		return 0, nil
	} else if len(value) == 7 {
		value = "0" + value
	} else if len(value) != 8 {
		return slowStringToSeconds(value)
	}
	// fast path, avoiding strings.Split (6x faster)
	a, ae := strconv.ParseInt(value[0:2], 10, 64)
	b, be := strconv.ParseInt(value[3:5], 10, 64)
	c, ce := strconv.ParseInt(value[6:8], 10, 64)
	if ae != nil || be != nil || ce != nil {
		// fallback if errors
		return slowStringToSeconds(value)
	}
	if b > 60 || c > 60 {
		return 0, errors.New("hours and mins must be 0 - 60")
	}
	return (a*3600 + b*60 + c), nil
}

func slowStringToSeconds(value string) (int64, error) {
	t := strings.SplitN(value, ":", 3)
	switch len(t) {
	case 3: // ok
	case 2:
		t = append(t, "0")
	case 1:
		t = append(t, "0", "0")
	}
	a, ae := strconv.ParseInt(t[0], 10, 64)
	b, be := strconv.ParseInt(t[1], 10, 64)
	c, ce := strconv.ParseInt(t[2], 10, 64)
	if ae != nil || be != nil || ce != nil {
		return 0, errors.New("error parsing time")
	}
	if b > 60 || c > 60 {
		return 0, errors.New("hours and mins must be 0 - 60")
	}
	return (a*3600 + b*60 + c), nil
}
