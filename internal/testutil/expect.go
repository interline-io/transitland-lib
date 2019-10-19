package testutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
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
func NewExpectError(filename, entityid, field, err string) *ExpectError {
	return &ExpectError{
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
	return fmt.Sprintf("%s:%s:%s", e.Filename, e.ErrorType, e.Field)
}

// Equals checks if two expect errors are equivalent.
func (e *ExpectError) Equals(other ExpectError) bool {
	// log.Trace("e: %#v other: %#v", e, other)
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

////////////


// TestEntityErrors checks that all expected Entity errors are present.
func TestEntityErrors(t *testing.T, r gotransit.Reader) {
	t.Run("Agencies", func(t *testing.T) {
		for ent := range r.Agencies() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Stops", func(t *testing.T) {
		for ent := range r.Stops() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Routes", func(t *testing.T) {
		for ent := range r.Routes() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Trips", func(t *testing.T) {
		for ent := range r.Trips() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("StopTimes", func(t *testing.T) {
		for ent := range r.StopTimes() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Calendars", func(t *testing.T) {
		for ent := range r.Calendars() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("CalendarDates", func(t *testing.T) {
		for ent := range r.CalendarDates() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("FareAttributes", func(t *testing.T) {
		for ent := range r.FareAttributes() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("FareRules", func(t *testing.T) {
		for ent := range r.FareRules() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("FeedInfos", func(t *testing.T) {
		for ent := range r.FeedInfos() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Shapes", func(t *testing.T) {
		for ent := range r.Shapes() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Transfer", func(t *testing.T) {
		for ent := range r.Transfers() {
			CheckEntityErrors(&ent, t)
		}
	})
	t.Run("Frequency", func(t *testing.T) {
		for ent := range r.Frequencies() {
			CheckEntityErrors(&ent, t)
		}
	})
}

// GetExpectError gets any ExpectError specified by an Entity.
func GetExpectError(ent gotransit.Entity) *ExpectError {
	ex := ent.Extra()
	if value, ok := ex["expect_error"]; len(value) > 0 && ok {
		ee := ExpectError{}
		ee.EntityID = ent.EntityID()
		ee.Filename = ent.Filename()
		v := strings.Split(value, ":")
		if len(v) >= 4 {
			ee.EntityID = v[3]
		}
		if len(v) >= 3 {
			ee.Filename = v[2]
		}
		if len(v) >= 2 {
			ee.Field = v[1]
		}
		if len(v) >= 1 {
			ee.ErrorType = v[0]
		}
		return &ee
	}
	return nil
}

// CheckEntityErrors checks if an Entity produced the specified ExpectError.
func CheckEntityErrors(ent gotransit.Entity, t *testing.T) {
	errs := ent.Errors()
	errs = append(errs, ent.Warnings()...)
	expect := GetExpectError(ent)
	if expect == nil {
		return
	}
	expect.Filename = ""
	expect.EntityID = ""
	if expect.ErrorType == "" {
		if len(errs) > 0 {
			t.Error("expected no errors, got:", len(errs), errs)
		}
		return
	}
	t.Run(fmt.Sprintf("%s:%s", expect.ErrorType, expect.Field), func(t *testing.T) {
		if !expect.Match(errs) {
			t.Error("did not find:", expect, "got:", errs)
		}
	})
}

