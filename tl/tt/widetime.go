package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type WideTime = Seconds

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

func (wt Seconds) HMS() (int, int, int) {
	secs := int(wt.Val)
	if secs < 0 {
		secs = 0
	}
	hours := secs / 3600
	minutes := (secs % 3600) / 60
	seconds := (secs % 3600) % 60
	return hours, minutes, seconds
}

func (wt Seconds) String() string {
	if !wt.Valid {
		return ""
	}
	return SecondsToString(int(wt.Val))
}

func (wt Seconds) Value() (driver.Value, error) {
	if !wt.Valid {
		return nil, nil
	}
	return wt.Val, nil
}

func (wt Seconds) ToCsv() string {
	return wt.String()
}

func (wt *Seconds) FromCsv(v string) error {
	wt.Valid = false
	if v == "" {
		return nil
	}
	if s, err := StringToSeconds(v); err != nil {
		return err
	} else {
		wt.Valid = true
		wt.Val = int64(s)
	}
	return nil
}

func (wt *Seconds) Scan(src interface{}) error {
	wt.Valid = false
	wt.Val = 0
	var p error
	switch v := src.(type) {
	case nil:
		return nil
	case string:
		return wt.FromCsv(v)
	case int:
		if v < 0 {
			return nil
		}
		wt.Val = int64(v)
	case int64:
		if v < 0 {
			return nil
		}
		wt.Val = v
	case json.Number:
		wt.Val, _ = v.Int64()
	default:
		p = errors.New("could not parse time")
	}
	wt.Valid = (p == nil)
	return p
}

func (wt *Seconds) UnmarshalGQL(v interface{}) error {
	return wt.Scan(v)
}

func (wt Seconds) MarshalGQL(w io.Writer) {
	if !wt.Valid {
		w.Write(jsonNull())
		return
	}
	w.Write([]byte(fmt.Sprintf("\"%s\"", wt.String())))
}

func (wt Seconds) Seconds() int {
	return int(wt.Val)
}

/////////////

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
