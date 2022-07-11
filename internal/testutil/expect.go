package testutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

type HasContext interface {
	Context() *causes.Context
}

///////////

// GetExpectErrors gets any ExpectError specified by an Entity.
func GetExpectErrors(ent tl.Entity) []ExpectError {
	ret := []ExpectError{}
	ex := ent.Extra()
	value, ok := ex["expect_error"]
	if len(value) == 0 || !ok {
		return ret
	}
	for _, v := range strings.Split(value, "|") {
		ee := ParseExpectError(v)
		if ee.Filename == "" {
			ee.Filename = ent.Filename()
		}
		if ee.EntityID == "" {
			ee.EntityID = ent.EntityID()
		}
		ret = append(ret, ee)
	}
	return ret
}

// CheckErrors checks actual vs. expected errors.
func CheckErrors(expecterrs []ExpectError, errs []error, t *testing.T) {
	s1 := []string{}
	for _, err := range errs {
		s1 = append(s1, fmt.Sprintf("%#v", err))
	}
	if len(errs) > len(expecterrs) {
		s2 := []string{}
		for _, err := range expecterrs {
			s2 = append(s2, fmt.Sprintf("%#v", err))
		}

		t.Errorf("got %d errors/warnings, more than the expected expected %d, got: %s expect: %s", len(errs), len(expecterrs), strings.Join(s1, " "), strings.Join(s2, " "))
		return
	}
	for _, expect := range expecterrs {
		expect.Filename = ""
		expect.EntityID = ""
		if !expect.Match(errs) {
			t.Errorf("did not find match for expected error %#v, got: %s", expect, strings.Join(s1, " "))
		}
	}
}

// ExpectError represents a single expected error.
type ExpectError struct {
	Filename       string
	EntityID       string
	Field          string
	ErrorType      string
	InnerErrorType string
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
	if e.ErrorType != "" && (e.ErrorType != other.ErrorType && e.ErrorType != other.InnerErrorType) {
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

type HasCause interface {
	Cause() error
}

// Match checks an array of errors and looks for a match.
func (e *ExpectError) Match(errs []error) bool {
	nerrs := []ExpectError{}
	for _, err := range errs {
		// Outer cause, if known
		expect := ExpectError{}
		if outer, ok := err.(HasContext); ok {
			expect.Filename = outer.Context().Filename
			expect.EntityID = outer.Context().EntityID
			expect.Field = outer.Context().Field
		}
		// Get error location context
		if inner, ok := err.(HasContext); ok {
			ctx := inner.Context()
			if len(ctx.Filename) > 0 {
				expect.Filename = ctx.Filename
			}
			if len(ctx.EntityID) > 0 {
				expect.EntityID = ctx.EntityID
			}
			expect.Field = ctx.Field
		}
		if err != nil {
			errtype := strings.Replace(fmt.Sprintf("%T", err), "*", "", 1)
			if len(strings.Split(errtype, ".")) > 1 {
				errtype = strings.Split(errtype, ".")[1]
			}
			expect.ErrorType = errtype
		}
		if inner, ok := err.(HasCause); ok && inner.Cause() != nil {
			cause := inner.Cause()
			causetype := strings.Replace(fmt.Sprintf("%T", cause), "*", "", 1)
			if len(strings.Split(causetype, ".")) > 1 {
				causetype = strings.Split(causetype, ".")[1]
			}
			expect.InnerErrorType = causetype
		}
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
