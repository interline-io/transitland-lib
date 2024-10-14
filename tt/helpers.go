package tt

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/causes"
)

// Error wrapping helpers

type canSetField interface {
	SetField(string)
}

func FirstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func TrySetField(err error, field string) error {
	if err == nil {
		return nil
	}
	if v, ok := err.(canSetField); ok {
		v.SetField(field)
	}
	return err
}

func AppendIf(err error, field string, errs []error) []error {
	if err != nil {
		if v, ok := err.(canSetField); ok {
			v.SetField(field)
		}
		errs = append(errs, err)
	}
	return errs
}

// CheckInArray returns an error if the value is not in the set of provided values.
func CheckInArray(field string, value string, values ...string) []error {
	for _, v := range values {
		if value == v {
			return nil
		}
	}
	return []error{causes.NewInvalidFieldError(field, value, fmt.Errorf("must be one of %s", strings.Join(values, ", ")))}
}

// CheckInArrayInt returns an error if the value is not in the set of provided values.
func CheckInArrayInt[T int | int64](field string, value T, values ...T) []error {
	for _, v := range values {
		if value == v {
			return nil
		}
	}
	var valueStrs []string
	for _, v := range values {
		valueStrs = append(valueStrs, fmt.Sprintf("%d", v))
	}
	return []error{causes.NewInvalidFieldError(field, fmt.Sprintf("%d", value), fmt.Errorf("must be one of %s", strings.Join(valueStrs, ", ")))}
}

// CheckPositive returns an error if the value is non-negative
func CheckPositive(field string, value float64) (errs []error) {
	if value < 0 {
		errs = append(errs, causes.NewInvalidFieldError(field, fmt.Sprintf("%f", value), fmt.Errorf("must be non-negative")))
	}
	return errs
}

// CheckPositiveInt returns an error if the value is non-negative
func CheckPositiveInt[T int | int64](field string, value T) (errs []error) {
	if value < 0 {
		errs = append(errs, causes.NewInvalidFieldError(field, fmt.Sprintf("%d", value), fmt.Errorf("must be non-negative")))
	}
	return errs
}

// CheckInsideRange returns an error if the value is outside of the specified range
func CheckInsideRange(field string, value float64, min float64, max float64) (errs []error) {
	if value < min || value > max {
		errs = append(errs, causes.NewInvalidFieldError(field, fmt.Sprintf("%f", value), fmt.Errorf("out of bounds, min %f max %f", min, max)))
	}
	return errs
}

// CheckInsideRangeInt returns an error if the value is outside of the specified range
func CheckInsideRangeInt[T int | int64](field string, value T, min T, max T) (errs []error) {
	if value < min || value > max {
		errs = append(errs, causes.NewInvalidFieldError(field, fmt.Sprintf("%d", value), fmt.Errorf("out of bounds, min %d max %d", min, max)))
	}
	return errs
}

// CheckPresent returns an error if a string is empty
func CheckPresent(field string, value string) (errs []error) {
	if value == "" {
		errs = append(errs, causes.NewRequiredFieldError(field))
	}
	return errs
}

func stripQuotes(v []byte) []byte {
	if len(v) < 2 {
		return v
	}
	if v[0] == '"' {
		v = v[1:]
	}
	if v[len(v)-1] == '"' {
		v = v[:len(v)-1]
	}
	return v
}

func jsonNull() []byte {
	return []byte("null")
}

func isEmpty(v string) bool {
	if len(v) == 0 || v == "" || v == "null" {
		return true
	}
	return false
}
