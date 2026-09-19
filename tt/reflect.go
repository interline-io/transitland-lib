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
	// and this is the only place those are reported.
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
		fieldAddr := field.Addr().Interface()

		// Check required. A tag the field's type cannot act on is a mistake in
		// the struct rather than anything about the data, so it is reported
		// here whichever pass the field's value checks belong to.
		fieldCheck, canCheck := fieldAddr.(CanReflectCheck)
		if !canCheck {
			if fieldInfo.Required || fieldInfo.Warn {
				errs = append(errs, fmt.Errorf("type %T does not support reflect based error checks", fieldAddr))
			}
		} else if fieldInfo.Required && !fieldCheck.IsPresent() {
			errs = append(errs, causes.NewRequiredFieldError(fieldName))
		}

		// A warn field has its value reported by ReflectCheckWarnings instead.
		// Absence is still an error above: the tag downgrades how a bad value
		// is judged, not whether a required field has to be there.
		if !fieldInfo.Warn {
			errs = append(errs, reflectCheckValue(fieldName, fieldAddr, fieldInfo)...)
		}
	}
	return errs
}

// reflectCheckValue runs the checks on a field's value: a value its own type
// rejects, the gt/gte/lt/lte/range bounds, and the enum values.
//
// They live together because the "warn" tag option moves all of them from
// CheckErrors to CheckWarnings at once. Splitting them would report a
// malformed value as a warning and an out of range one as an error, on the
// same field.
func reflectCheckValue(fieldName string, fieldAddr any, fieldInfo *tags.FieldInfo) []error {
	var errs []error

	// Check type based validation
	if fieldCheck, ok := fieldAddr.(CanReflectCheck); ok {
		if err := fieldCheck.Check(); err != nil {
			errs = append(errs, TrySetField(err, fieldName))
		}
	}

	// Check range min/max
	if fieldInfo.GreaterOrEqual != nil || fieldInfo.LessOrEqual != nil || fieldInfo.GreaterThan != nil || fieldInfo.LessThan != nil {
		if fieldCheck, ok := fieldAddr.(canReflectCheckFloat); !ok {
			errs = append(errs, fmt.Errorf("could not convert %T to float for range check", fieldAddr))
		} else if fieldCheck.IsPresent() {
			checkVal := fieldCheck.Float()
			if minVal, ok := checkFloat(fieldInfo.GreaterThan); ok && checkVal <= minVal {
				checkErr := causes.NewInvalidFieldError(fieldName, fieldCheck.String(), fmt.Errorf("out of bounds, less than or equal to %f", minVal))
				errs = append(errs, checkErr)
			}
			if maxVal, ok := checkFloat(fieldInfo.LessThan); ok && checkVal >= maxVal {
				checkErr := causes.NewInvalidFieldError(fieldName, fieldCheck.String(), fmt.Errorf("out of bounds, greater than or equal to %f", maxVal))
				errs = append(errs, checkErr)
			}
			if minVal, ok := checkFloat(fieldInfo.GreaterOrEqual); ok && checkVal < minVal {
				checkErr := causes.NewInvalidFieldError(fieldName, fieldCheck.String(), fmt.Errorf("out of bounds, less than %f", minVal))
				errs = append(errs, checkErr)
			}
			if maxVal, ok := checkFloat(fieldInfo.LessOrEqual); ok && checkVal > maxVal {
				checkErr := causes.NewInvalidFieldError(fieldName, fieldCheck.String(), fmt.Errorf("out of bounds, greater than %f", maxVal))
				errs = append(errs, checkErr)
			}
		}
	}

	// Check enum values
	if len(fieldInfo.EnumValues) > 0 {
		if fieldCheck, ok := fieldAddr.(canReflectCheckInt); !ok {
			errs = append(errs, fmt.Errorf("could not convert %T to int for enum check", fieldAddr))
		} else if fieldCheck.IsPresent() {
			checkVal := int64(fieldCheck.Int())
			found := false
			for _, enumValue := range fieldInfo.EnumValues {
				if checkVal == enumValue {
					found = true
				}
			}
			if !found {
				checkErr := causes.NewInvalidFieldError(fieldName, fieldCheck.String(), fmt.Errorf("not in allowed values"))
				errs = append(errs, checkErr)
			}
		}
	}
	return errs
}

// ReflectCheckWarnings returns field validation failures that the "warn" tag
// option downgrades from errors.
//
// A field is tagged this way when a value the library cannot recognize is not
// a reason to drop the entity: agency_lang is advisory, and the registries
// these values are checked against gain entries over time, so a feed can be
// correct while the library is merely out of date.
func ReflectCheckWarnings(ent any) []error {
	// Nothing addressable to check. This is exported and takes any, so a
	// caller that passes a value or a nil gets no warnings rather than a
	// panic.
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
		errs = append(errs, reflectCheckValue(fieldInfo.Name, field.Addr().Interface(), fieldInfo)...)
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
