package tt

import (
	"fmt"
	"reflect"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/internal/tags"
	"github.com/jmoiron/sqlx/reflectx"
)

var mapperCache = tags.NewCache(reflectx.NewMapperFunc("csv", tags.ToSnakeCase))

func CheckErrors(ent any) []error {
	var errs []error
	if a, ok := ent.(EntityWithLoadErrors); ok {
		errs = append(errs, a.LoadErrors()...)
	}
	if a, ok := ent.(EntityWithConditionalErrors); ok {
		errs = append(errs, a.ConditionalErrors()...)
	}
	if a, ok := ent.(EntityWithErrors); ok {
		errs = append(errs, a.Errors()...)
		// Hand rolling Errors() is a fast path past the reflect based value
		// checks. It is not a way for a warn tagged field to escape the checks
		// that stay errors: the field's value is reported by CheckWarnings, but
		// a required value that is absent, and a tag the field's type cannot
		// act on, are still errors and are still reported here.
		errs = append(errs, reflectCheckWarnFieldErrors(ent)...)
	} else {
		errs = append(errs, ReflectCheckErrors(ent)...)
	}
	return errs
}

func CheckWarnings(ent any) []error {
	var errs []error
	if a, ok := ent.(EntityWithLoadErrors); ok {
		errs = append(errs, a.LoadWarnings()...)
	}
	if a, ok := ent.(EntityWithWarnings); ok {
		errs = append(errs, a.Warnings()...)
	}
	// Unlike CheckErrors, this runs for every entity. Hand rolling Errors() as
	// a fast path opts an entity out of the reflect based error checks; it does
	// not opt the entity out of having its warn tagged fields checked at all,
	// and this is the only place those values are reported.
	errs = append(errs, ReflectCheckWarnings(ent)...)
	return errs
}

type CanReflectCheck interface {
	String() string
	IsPresent() bool
	Check() error
}

type canReflectCheckInt interface {
	CanReflectCheck
	Int() int
}

type canReflectCheckFloat interface {
	CanReflectCheck
	Float() float64
}

func checkFloat(val *float64) (float64, bool) {
	if val == nil {
		return 0, false
	}
	return *val, true
}

// checkField is one field resolved for checking: the name it is reported
// under, its addressable value, its parsed tags, and its type's check
// interface if it has one.
//
// The interface is resolved once here because both passes over a field need
// it, and this runs for every field of every entity.
type checkField struct {
	name  string
	addr  any
	info  *tags.FieldInfo
	check CanReflectCheck
}

func newCheckField(name string, addr any, info *tags.FieldInfo) checkField {
	check, _ := addr.(CanReflectCheck)
	return checkField{name: name, addr: addr, info: info, check: check}
}

func (f checkField) hasRange() bool {
	return f.info.GreaterOrEqual != nil || f.info.LessOrEqual != nil || f.info.GreaterThan != nil || f.info.LessThan != nil
}

// checkErrors returns the failures on a field that stay errors whichever pass
// its value checks belong to.
//
// A required value that is not there is one: the "warn" option downgrades how
// a bad value is judged, not whether a required field has to be present at
// all. So is a tag the field's type cannot act on, which is a mistake in the
// struct rather than anything about the data. Downgrading that would file a
// developer's mistake in the feed's warning report, once per entity, under the
// producer's name.
func (f checkField) checkErrors() []error {
	var errs []error
	if f.check == nil {
		if f.info.Required || f.info.Warn {
			errs = append(errs, fmt.Errorf("type %T does not support reflect based error checks", f.addr))
		}
	} else if f.info.Required && !f.check.IsPresent() {
		errs = append(errs, causes.NewRequiredFieldError(f.name))
	}
	if f.hasRange() {
		if _, ok := f.addr.(canReflectCheckFloat); !ok {
			errs = append(errs, fmt.Errorf("could not convert %T to float for range check on field %s", f.addr, f.name))
		}
	}
	if len(f.info.EnumValues) > 0 {
		if _, ok := f.addr.(canReflectCheckInt); !ok {
			errs = append(errs, fmt.Errorf("could not convert %T to int for enum check on field %s", f.addr, f.name))
		}
	}
	return errs
}

// checkValue returns the failures in a field's value: a value its own type
// rejects, the gt/gte/lt/lte bounds, and the enum values.
//
// They live together because the "warn" tag option moves all of them from
// CheckErrors to CheckWarnings at once. Splitting them would report a
// malformed value as a warning and an out of range one as an error, on the
// same field.
//
// A type that cannot act on one of these tags is passed over: checkErrors
// reports that as the struct mistake it is, so it is neither silent nor a
// warning about the feed.
func (f checkField) checkValue() []error {
	if f.check == nil {
		return nil
	}
	var errs []error

	// Check type based validation
	if err := f.check.Check(); err != nil {
		errs = append(errs, TrySetField(err, f.name))
	}

	// Check range min/max
	if f.hasRange() {
		if fieldCheck, ok := f.addr.(canReflectCheckFloat); ok && fieldCheck.IsPresent() {
			checkVal := fieldCheck.Float()
			if minVal, ok := checkFloat(f.info.GreaterThan); ok && checkVal <= minVal {
				checkErr := causes.NewInvalidFieldError(f.name, fieldCheck.String(), fmt.Errorf("out of bounds, less than or equal to %f", minVal))
				errs = append(errs, checkErr)
			}
			if maxVal, ok := checkFloat(f.info.LessThan); ok && checkVal >= maxVal {
				checkErr := causes.NewInvalidFieldError(f.name, fieldCheck.String(), fmt.Errorf("out of bounds, greater than or equal to %f", maxVal))
				errs = append(errs, checkErr)
			}
			if minVal, ok := checkFloat(f.info.GreaterOrEqual); ok && checkVal < minVal {
				checkErr := causes.NewInvalidFieldError(f.name, fieldCheck.String(), fmt.Errorf("out of bounds, less than %f", minVal))
				errs = append(errs, checkErr)
			}
			if maxVal, ok := checkFloat(f.info.LessOrEqual); ok && checkVal > maxVal {
				checkErr := causes.NewInvalidFieldError(f.name, fieldCheck.String(), fmt.Errorf("out of bounds, greater than %f", maxVal))
				errs = append(errs, checkErr)
			}
		}
	}

	// Check enum values
	if len(f.info.EnumValues) > 0 {
		if fieldCheck, ok := f.addr.(canReflectCheckInt); ok && fieldCheck.IsPresent() {
			checkVal := int64(fieldCheck.Int())
			found := false
			for _, enumValue := range f.info.EnumValues {
				if checkVal == enumValue {
					found = true
				}
			}
			if !found {
				checkErr := causes.NewInvalidFieldError(f.name, fieldCheck.String(), fmt.Errorf("not in allowed values"))
				errs = append(errs, checkErr)
			}
		}
	}
	return errs
}

// Error wrapping helpers
func ReflectCheckErrors(ent any) []error {
	var errs []error
	entValue := reflect.ValueOf(ent).Elem()
	fmap := mapperCache.GetStructTagMap(ent)
	for fieldName, fieldInfo := range fmap {
		// An alias is another name for a field that is checked under its own
		// name; checking it again would report the same value twice, once under
		// a name the file may not even contain.
		if fieldInfo.IsAlias() {
			continue
		}
		// Get field
		field := reflectx.FieldByIndexes(entValue, fieldInfo.Index)
		f := newCheckField(fieldName, field.Addr().Interface(), fieldInfo)
		errs = append(errs, f.checkErrors()...)
		// A warn field has its value reported by ReflectCheckWarnings instead.
		if !fieldInfo.Warn {
			errs = append(errs, f.checkValue()...)
		}
	}
	return errs
}

// ReflectCheckWarnings returns the field validation failures that the "warn"
// tag option downgrades from errors.
//
// A field is tagged this way when a value the library cannot recognize is not
// a reason to drop the entity: agency_lang is advisory, and the registries
// these values are checked against gain entries over time, so a feed can be
// correct while the library is merely out of date.
func ReflectCheckWarnings(ent any) []error {
	return eachWarnField(ent, checkField.checkValue)
}

// reflectCheckWarnFieldErrors returns what checkErrors reports about an
// entity's warn tagged fields. ReflectCheckErrors already does this for every
// field; this covers the entities that hand roll Errors() and so never reach
// it.
func reflectCheckWarnFieldErrors(ent any) []error {
	return eachWarnField(ent, checkField.checkErrors)
}

// eachWarnField applies check to each of the entity's warn tagged fields.
func eachWarnField(ent any, check func(checkField) []error) []error {
	// Nothing addressable to check. The callers are exported and take any, so
	// one that is handed a value or a nil returns nothing rather than panics.
	entValue := reflect.ValueOf(ent)
	if entValue.Kind() != reflect.Pointer || entValue.IsNil() {
		return nil
	}
	// This runs for every entity in a copy while almost no type has a warn
	// tagged field, so the fields are looked up rather than searched for.
	warnFields := mapperCache.GetWarnFields(ent)
	if len(warnFields) == 0 {
		return nil
	}
	var errs []error
	entElem := entValue.Elem()
	for _, fieldInfo := range warnFields {
		field := reflectx.FieldByIndexes(entElem, fieldInfo.Index)
		f := newCheckField(fieldInfo.Name, field.Addr().Interface(), fieldInfo)
		errs = append(errs, check(f)...)
	}
	return errs
}

func ReflectUpdateKeys(emap *EntityMap, ent any) []error {
	var errs []error
	fields := entityMapperCache.GetStructTagMap(ent)
	for fieldName, fieldInfo := range fields {
		// Get target from field tags
		if fieldInfo.Target == "" {
			continue
		}
		elem := reflect.ValueOf(ent).Elem()
		fieldValue := reflectx.FieldByIndexes(elem, fieldInfo.Index)
		fieldAddr := fieldValue.Addr().Interface()
		fieldSet, ok := fieldAddr.(canSet)
		if !ok {
			errs = append(errs, fmt.Errorf("type %T does not support reflect based reference checks", fieldAddr))
			continue
		}
		eid := fieldSet.String()
		if eid == "" {
			continue
		}
		// Check if reference exists
		newId, ok := emap.Get(fieldInfo.Target, eid)
		if !ok {
			errs = append(errs, TrySetField(causes.NewInvalidReferenceError(fieldName, eid), fieldName))
			continue
		}
		// Update the value *if* the reference exists
		fieldSet.Set(newId)
	}
	return errs
}
