package tt

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/interline-io/transitland-lib/tl/causes"
)

type Option[T any] struct {
	Val   T
	Valid bool
}

func (r *Option[T]) Present() bool {
	return r.Valid
}

func (r Option[T]) String() string {
	if !r.Valid {
		return ""
	}
	b, _ := r.MarshalJSON()
	return string(b)
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
	var z T
	json.Unmarshal(v, &z)
	return r.Scan(z)
}

func (r *Option[T]) MarshalJSON() ([]byte, error) {
	if !r.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(r.Val)
}

func (r *Option[T]) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

func (r *Option[T]) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}

func convertAssign(dest any, src any) error {
	if src == nil {
		return nil
	}
	var err error
	switch s := dest.(type) {
	case *string:
		switch d := src.(type) {
		case string:
			*s = d
		case []byte:
			*s = string(d)
		case int:
			*s = strconv.Itoa(d)
		case int64:
			*s = strconv.Itoa(int(d))
		case float32:
			*s = fmt.Sprintf("%0.5f", d)
		case float64:
			*s = fmt.Sprintf("%0.5f", d)
		default:
			err = cannotConvert()
		}
	case *int:
		switch d := src.(type) {
		case string:
			*s, err = strconv.Atoi(d)
		case []byte:
			*s, err = strconv.Atoi(string(d))
		case int:
			*s = int(d)
		case int64:
			*s = int(d)
		case float32:
			*s = int(d)
		case float64:
			*s = int(d)
		default:
			err = cannotConvert()
		}
	case *int64:
		switch d := src.(type) {
		case string:
			*s, err = strconv.ParseInt(d, 10, 64)
		case []byte:
			*s, err = strconv.ParseInt(string(d), 10, 64)
		case int:
			*s = int64(d)
		case int64:
			*s = int64(d)
		case float32:
			*s = int64(d)
		case float64:
			*s = int64(d)
		default:
			err = cannotConvert()
		}
	case *float64:
		switch d := src.(type) {
		case string:
			*s, err = strconv.ParseFloat(d, 64)
		case []byte:
			*s, err = strconv.ParseFloat(string(d), 64)
		case int:
			*s = float64(d)
		case int64:
			*s = float64(d)
		case float32:
			*s = float64(d)
		case float64:
			*s = float64(d)
		default:
			err = cannotConvert()
		}
	case *time.Time:
		switch d := src.(type) {
		case []byte:
			*s, err = parseTime(string(d))
		case string:
			*s, err = parseTime(d)
		case time.Time:
			*s = d
		default:
			err = cannotConvert()
		}
	default:
		err = cannotConvert()
	}
	return err
}

func cannotConvert() error {
	return errors.New("cannot convert")
}

func bcString(v string) causes.Context {
	return causes.Context{Value: v}
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
