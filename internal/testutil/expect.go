package testutil

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/pkg/errors"
)

type context interface {
	Context() *causes.Context
}

///////////

// ExpectError represents a single expected error.
type ExpectError struct {
	Filename  string
	EntityID  string
	Field     string
	ErrorType string
}

// NewExpectError returns a new ExpectError.
func NewExpectError(filename, entityid, field, err string) ExpectError {
	return ExpectError{
		Filename:  filename,
		Field:     field,
		EntityID:  entityid,
		ErrorType: err,
	}
}

func (e *ExpectError) Error() string {
	return e.String()
}

func (e *ExpectError) String() string {
	return fmt.Sprintf("%s:%s:%s:%s", e.Filename, e.Field, e.EntityID, e.ErrorType)
}

// Equals checks if two expect errors are equivalent.
func (e *ExpectError) Equals(other ExpectError) bool {
	if len(e.ErrorType) > 0 && e.ErrorType != other.ErrorType {
		return false
	} else if len(e.Field) > 0 && e.Field != other.Field {
		return false
	} else if len(e.EntityID) > 0 && e.EntityID != other.EntityID {
		return false
	} else if len(e.Filename) > 0 && e.Filename != other.Filename {
		return false
	}
	return true
}

// Match checks an array of errors and looks for a match.
func (e *ExpectError) Match(errs []error) bool {
	nerrs := []ExpectError{}
	for _, err := range errs {
		// Outer cause, if known
		expect := ExpectError{}
		if outer, ok := err.(context); ok {
			expect.Filename = outer.Context().Filename
			expect.EntityID = outer.Context().EntityID
			expect.Field = outer.Context().Field
		}
		// Inner most cause
		cause := errors.Cause(err)
		if inner, ok := cause.(context); ok {
			ctx := inner.Context()
			if len(ctx.Filename) > 0 {
				expect.Filename = ctx.Filename
			}
			if len(ctx.EntityID) > 0 {
				expect.EntityID = ctx.EntityID
			}
			expect.Field = ctx.Field
		}
		errtype := strings.Replace(fmt.Sprintf("%T", cause), "*", "", 1)
		if len(strings.Split(errtype, ".")) > 1 {
			errtype = strings.Split(errtype, ".")[1]
		}
		expect.ErrorType = errtype
		nerrs = append(nerrs, expect)
	}
	for _, e2 := range nerrs {
		if e.Equals(e2) {
			return true
		}
	}
	return false
}

// ParseExpectError .
// e.g.:
//     InvalidFieldError:agency_name:agency.txt:bad_agency
func ParseExpectError(value string) ExpectError {
	v := strings.Split(value, ":")
	// pad out
	for i := 0; i < 4; i++ {
		v = append(v, "")
	}
	return NewExpectError(v[2], v[3], v[1], v[0])
}
