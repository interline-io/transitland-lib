package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	geom "github.com/twpayne/go-geom"
)

type canString interface {
	String() string
}

type canValue interface {
	Value() (driver.Value, error)
}

type canCsvString interface {
	ToCsv() string
}

type canFromCsvString interface {
	FromCsv(string) error
}

type canScan interface {
	Scan(src interface{}) error
}

// FromCSV sets the field from a CSV representation of the value.
func FromCsv(val any, strv string) error {
	var p error
	switch vf := val.(type) {
	case *string:
		*vf = strv
	case *int:
		v, e := strconv.ParseInt(strv, 0, 0)
		p = e
		*vf = int(v)
	case *int64:
		v, e := strconv.ParseInt(strv, 0, 0)
		p = e
		*vf = v
	case *float64:
		v, e := strconv.ParseFloat(strv, 64)
		p = e
		*vf = v
	case *bool:
		if strv == "true" {
			*vf = true
		} else {
			*vf = false
		}
	case *time.Time:
		v, e := time.Parse("20060102", strv)
		p = e
		*vf = v
	case canFromCsvString:
		if err := vf.FromCsv(strv); err != nil {
			p = errors.New("field not scannable")
		}
	case canScan:
		if err := vf.Scan(strv); err != nil {
			p = errors.New("field not scannable")
		}
	default:
		p = errors.New("field not scannable")
	}
	return p
}

func formatFloat(v float64) string {
	value := ""
	if math.IsNaN(v) {
		value = ""
	} else if v > -100_000 && v < 100_000 {
		// use pretty %g formatting but avoid exponents
		value = fmt.Sprintf("%g", v)
	} else {
		value = fmt.Sprintf("%0.5f", v)
	}
	return value
}

// ToCsv converts any value to a CSV string representation
func ToCsv(val any) (string, error) {
	value := ""
	switch v := val.(type) {
	case nil:
		value = ""
	case string:
		value = v
	case int:
		value = strconv.Itoa(v)
	case int64:
		value = strconv.Itoa(int(v))
	case bool:
		if v {
			value = "true"
		} else {
			value = "false"
		}
	case float64:
		value = formatFloat(v)
	case float32:
		value = formatFloat(float64(v))
	case time.Time:
		if v.IsZero() {
			value = ""
		} else {
			value = v.Format("20060102")
		}
	case []byte:
		value = string(v)
	case canCsvString:
		value = v.ToCsv()
	case canValue:
		a, err := v.Value()
		if err != nil {
			return "", err
		}
		return ToCsv(a)
	case canString:
		value = v.String()
	case int8, int16, int32, uint, uint8, uint16, uint32, uint64:
		value = fmt.Sprintf("%d", v)
	default:
		return "", fmt.Errorf("can not convert field to string")
	}
	return value, nil
}

// TryCsv converts any value to a CSV string representation, ignoring errors
func TryCsv(val any) string {
	a, _ := ToCsv(val)
	return a
}

func convertAssign(dest any, src any) error {
	if src == nil {
		return nil
	}
	var err error
	switch d := dest.(type) {
	case *string:
		switch s := src.(type) {
		case string:
			*d = s
		case []byte:
			*d = string(s)
		case int:
			*d = strconv.Itoa(s)
		case int64:
			*d = strconv.Itoa(int(s))
		case float64:
			*d = fmt.Sprintf("%0.5f", s)
		case time.Time:
			*d = s.Format(time.RFC3339)
		default:
			err = cannotConvert(dest, src)
		}
	case *int:
		switch s := src.(type) {
		case string:
			*d, err = strconv.Atoi(s)
		case []byte:
			*d, err = strconv.Atoi(string(s))
		case int:
			*d = int(s)
		case int64:
			*d = int(s)
		case float64:
			*d = int(s)
		default:
			err = cannotConvert(dest, src)
		}
	case *int64:
		switch s := src.(type) {
		case string:
			*d, err = strconv.ParseInt(s, 10, 64)
		case []byte:
			*d, err = strconv.ParseInt(string(s), 10, 64)
		case int:
			*d = int64(s)
		case int64:
			*d = int64(s)
		case float64:
			*d = int64(s)
		default:
			err = cannotConvert(dest, src)
		}
	case *float64:
		switch s := src.(type) {
		case string:
			*d, err = strconv.ParseFloat(s, 64)
		case []byte:
			*d, err = strconv.ParseFloat(string(s), 64)
		case int:
			*d = float64(s)
		case int64:
			*d = float64(s)
		case float64:
			*d = float64(s)
		default:
			err = cannotConvert(dest, src)
		}
	case *bool:
		switch s := src.(type) {
		case []byte:
			ss := string(s)
			if ss == "true" || ss == "1" {
				*d = true
			} else if ss == "false" || ss == "0" {
				*d = false
			} else {
				err = cannotConvert(dest, src)
			}
		case string:
			if s == "true" || s == "1" {
				*d = true
			} else if s == "false" || s == "0" {
				*d = false
			} else {
				err = cannotConvert(dest, src)
			}
		case bool:
			*d = s
		default:
			err = cannotConvert(dest, src)
		}
	case *time.Time:
		switch s := src.(type) {
		case []byte:
			*d, err = parseTime(string(s))
		case string:
			*d, err = parseTime(s)
		case time.Time:
			*d = s
		default:
			err = cannotConvert(dest, src)
		}
	default:
		err = cannotConvert(dest, src)
	}
	return err
}

func cannotConvert(dest any, src any) error {
	return fmt.Errorf("could not convert type %T into %T", src, dest)
}

func parseTime(d string) (time.Time, error) {
	var err error
	var s time.Time
	if len(d) == 8 {
		s, err = time.Parse("20060102", d)
	} else if len(d) == 10 {
		s, err = time.Parse("2006-01-02", d)
	} else {
		s, err = time.Parse(time.RFC3339, d)
	}
	return s, err
}

func toJson(v any) ([]byte, error) {
	switch s := v.(type) {
	case geom.T:
		return geojsonEncode(s)
	}
	return json.Marshal(v)
}
