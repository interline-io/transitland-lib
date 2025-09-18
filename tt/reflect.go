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
		// Get field
		field := reflectx.FieldByIndexes(entValue, fieldInfo.Index)
		fieldAddr := field.Addr().Interface()
		if fieldAddr == nil {
			continue
		}

		// Check required and type based validation
		if fieldCheck, ok := fieldAddr.(CanReflectCheck); ok {
			if fieldInfo.Required && !fieldCheck.IsPresent() {
				errs = append(errs, causes.NewRequiredFieldError(fieldName))
			}
			if err := fieldCheck.Check(); err != nil {
				errs = append(errs, TrySetField(err, fieldName))
			}
		} else if fieldInfo.Required {
			errs = append(errs, fmt.Errorf("type %T does not support reflect based error checks", fieldAddr))
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
