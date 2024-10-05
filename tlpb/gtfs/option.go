package gtfs

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/interline-io/transitland-lib/tl/tt"
)

func (r *EnumValue) FromCsv(v string) error {
	a, err := parseInt32(v)
	*r = EnumValue(a)
	return err
}

func (r *Seconds) FromCsv(v string) error {
	wt, err := tt.NewWideTime(v)
	r.Val = int64(wt.Seconds)
	r.Valid = (err == nil)
	return nil
}

type Option[T any] struct {
	Val   T
	Valid bool
}

func (r *Option[T]) Present() bool {
	return r.Valid
}

func (r *Option[T]) FromCsv(v string) error {
	err := convertAssign(&r.Val, v)
	if err != nil {
		// fmt.Printf("\tfailed string '%s' into %T: %s\n", v, r.Val, err.Error())
		return err
	}
	// fmt.Printf("\t\tgot: %T %v\n", r.Val, r.Val)
	r.Valid = (err == nil)
	return nil
}

func (r Option[T]) String() string {
	if !r.Valid {
		return ""
	}
	out := ""
	if err := convertAssign(&out, r.Val); err != nil {
		b, _ := r.MarshalJSON()
		return string(b)
	}
	return out
}

func (r *Option[T]) Error() error {
	return nil
}

func (r *Option[T]) Scan(src interface{}) error {
	err := convertAssign(&r.Val, src)
	r.Valid = (src != nil && err == nil)
	return err
}

func (r Option[T]) Value() (driver.Value, error) {
	if !r.Valid {
		return nil, nil
	}
	return r.Val, nil
}

func (r *Option[T]) UnmarshalJSON(v []byte) error {
	err := json.Unmarshal(v, &r.Val)
	r.Valid = (err == nil)
	return err
}

func (r Option[T]) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Val)
}

type canFromCsv interface {
	FromCsv(string) error
}

func convertAssign(dest any, src any) error {
	// fmt.Printf("convertAssign %T '%v' into %T\n", src, src, dest)
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
			err = cannotConvert()
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
			err = cannotConvert()
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
			err = cannotConvert()
		}
	case *int32:
		switch s := src.(type) {
		case string:
			*d, err = parseInt32(s)
		case []byte:
			*d, err = parseInt32(string(s))
		case int:
			*d = int32(s)
		case int64:
			*d = int32(s)
		case float64:
			*d = int32(s)
		default:
			err = cannotConvert()
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
			err = cannotConvert()
		}
	case *bool:
		switch s := src.(type) {
		case string:
			if s == "true" {
				*d = true
			} else if s == "false" {
				*d = false
			} else {
				err = cannotConvert()
			}
		case bool:
			*d = s
		default:
			err = cannotConvert()
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
			err = cannotConvert()
		}
	default:
		// Handle type aliases
		k := reflect.ValueOf(dest).Elem()
		if k.CanInt() {
			xi := int64(0)
			err = convertAssign(&xi, src)
			k.SetInt(xi)
		} else {
			err = cannotConvert()
		}
	}
	return err
}

func parseInt32(s string) (int32, error) {
	d, err := strconv.ParseInt(s, 10, 64)
	return int32(d), err
}

func cannotConvert() error {
	return errors.New("cannot convert")
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
