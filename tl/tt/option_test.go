package tt

import (
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptionString(t *testing.T) {
	testStr := "hello"
	quote := func(v string) string { return "\"" + v + "\"" }
	jsonNull := "null"
	atime := time.Now().UTC().Truncate(time.Second)
	aTimeIso := atime.Format(time.RFC3339)
	dayTimeStr := "2023-03-08"
	dayTime, _ := time.Parse("2006-01-02", dayTimeStr)
	type option interface {
		Scan(any) error
		String() string
		Value() (driver.Value, error)
		MarshalJSON() ([]byte, error)
		UnmarshalJSON([]byte) error
	}
	type tc struct {
		name string
		new  func() option
		scan map[any]any
		str  map[any]any
		uj   map[string]any
		mj   map[any]string
	}
	tcs := []tc{
		{
			name: "string",
			new:  func() option { return &Option[string]{} },
			scan: map[any]any{
				testStr:  testStr,
				"1":      "1",
				1:        "1",
				nil:      nil,
				true:     nil,
				"true":   "true",
				"nil":    "nil",
				1.23:     "1.23000",
				1.234567: "1.23457",
				nil:      nil,
			},
			str: map[any]any{
				testStr:  testStr,
				"1":      "1",
				1:        "1",
				nil:      "",
				true:     "",
				1.23:     "1.23000",
				1.234567: "1.23457",
			},
			uj: map[string]any{
				quote(testStr): testStr,
				quote("1"):     "1",
				quote(""):      "",
			},
			mj: map[any]string{
				testStr: quote(testStr),
				"1":     quote("1"),
				nil:     jsonNull,
				1.23456: quote("1.23456"),
			},
		},
		{
			name: "int64",
			new:  func() option { return &Option[int64]{} },
			scan: map[any]any{
				1234:    1234,
				1.234:   1,
				1.567:   1,
				1:       1,
				"1234":  1234,
				"1":     1,
				"fail":  nil,
				"1.234": nil,
			},
			str: map[any]any{
				1234:   "1234",
				1.234:  "1",
				1.567:  "1",
				"fail": "",
			},
			uj: map[string]any{
				"1":             1,
				"1.234":         nil, // should be more lenient?
				quote("fail"):   nil,
				quote(jsonNull): nil,
			},
			mj: map[any]string{
				1:      "1",
				1.234:  "1",
				"fail": jsonNull,
			},
		},
		{
			name: "float64",
			new:  func() option { return &Option[float64]{} },
			scan: map[any]any{
				1234:    1234,
				1.234:   1.234,
				1.567:   1.567,
				1:       1,
				"1234":  1234,
				"1":     1,
				"fail":  nil,
				"1.234": 1.234,
			},
			str: map[any]any{
				1234:   "1234.00000",
				1.234:  "1.23400",
				1.567:  "1.56700",
				"fail": "",
			},
			uj: map[string]any{
				"1":             1.0,
				"1.234":         1.234,
				quote("fail"):   nil,
				quote(jsonNull): nil,
			},
			mj: map[any]string{
				1:      "1",
				1.234:  "1.234",
				"fail": jsonNull,
			},
		},
		{
			name: "bool",
			new:  func() option { return &Option[bool]{} },
			scan: map[any]any{
				true:    true,
				false:   false,
				nil:     nil,
				"true":  true,
				"false": false,
				"fail":  nil,
			},
			str: map[any]any{
				true:  "true",
				false: "false",
				nil:   "",
				1:     "", // fail
				1.234: "", // fail
			},
			uj: map[string]any{
				"true":  true,
				"false": false,
				"fail":  nil, // fail
			},
			mj: map[any]string{
				true:  "true",  // json not quoted
				false: "false", // json not quoted
				nil:   jsonNull,
			},
		},
		{
			name: "time",
			new:  func() option { return &Option[time.Time]{} },
			scan: map[any]any{
				atime:      atime,
				aTimeIso:   atime,
				dayTimeStr: dayTime,
				"":         nil,
				"fail":     nil, // fail
				1:          nil, // fail
				true:       nil, // fail
			},
			str: map[any]any{
				atime:   aTimeIso,
				dayTime: dayTime.Format(time.RFC3339),
				"":      "", // fail
				"1":     "", // fail
				1:       "", // fail
			},
			uj: map[string]any{
				quote(aTimeIso):   atime,
				quote(dayTimeStr): nil, // date format is strict and requires time not just date
			},
			mj: map[any]string{
				atime:   quote(aTimeIso),
				dayTime: quote(dayTime.Format(time.RFC3339)),
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("Scan", func(t *testing.T) {
				for k, v := range tc.scan {
					a := tc.new()
					a.Scan(k)
					b, _ := a.Value()
					assert.EqualValues(t, v, b)
				}
			})
			t.Run("String", func(t *testing.T) {
				for k, v := range tc.str {
					a := tc.new()
					a.Scan(k)
					assert.EqualValues(t, v, a.String())
				}
			})
			t.Run("UnmarshalJSON", func(t *testing.T) {
				for k, v := range tc.uj {
					a := tc.new()
					if err := a.UnmarshalJSON([]byte(k)); err != nil {
						fmt.Println("err:", err)
					}
					b, err := a.Value()
					_ = err
					assert.EqualValues(t, v, b, "UnmarshalJSON value '%s' expected '%v', got '%v'", k, v, b)
				}
			})
			t.Run("MarshalJSON", func(t *testing.T) {
				for k, v := range tc.mj {
					a := tc.new()
					a.Scan(k)
					b, _ := a.MarshalJSON()
					assert.EqualValues(t, v, string(b))
				}
			})
		})
	}
}
