package tt

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
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
		fmt.Printf("ERROR %T", val)
		p = errors.New("field not scannable")
	}
	return p
}

// ToCsv converts any value to a CSV string representation
func ToCsv(val any) (string, error) {
	// Check ToCsv() and Value() first, then primitives, then String()
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
		if math.IsNaN(v) {
			value = ""
		} else if v > -100_000 && v < 100_000 {
			// use pretty %g formatting but avoid exponents
			value = fmt.Sprintf("%g", v)
		} else {
			value = fmt.Sprintf("%0.5f", v)
		}
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
