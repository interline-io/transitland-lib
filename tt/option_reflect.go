package tt

import (
	"fmt"
	"reflect"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/internal/tags"
	"github.com/jmoiron/sqlx/reflectx"
)

var mapperCache = tags.NewCache(reflectx.NewMapperFunc("csv", tags.ToSnakeCase))

type CanCheck interface {
	String() string
	IsValid() bool
	Check() error
}

func getString(v any) string {
	type canString interface {
		String() string
	}

	if a, ok := v.(canString); ok {
		return a.String()
	}

	return ""
}

func getFloat(v any) string {
	type canString interface {
		String() string
	}
	if a, ok := v.(canString); ok {
		return a.String()
	}
	return ""
}

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

// Error wrapping helpers
func ReflectCheckErrors(ent any) []error {
	var errs []error
	entValue := reflect.ValueOf(ent).Elem()
	fmap := mapperCache.GetStructTagMap(ent)
	for fieldName, fieldInfo := range fmap {
		field := reflectx.FieldByIndexes(entValue, fieldInfo.Index)
		fieldAddr := field.Addr().Interface()
		if fieldAddr == nil {
			continue
		}
		fieldCheck, ok := fieldAddr.(CanCheck)
		if !ok {
			if fieldInfo.Required && field.IsZero() {
				errs = append(errs, causes.NewRequiredFieldError(fieldName))
			}
			continue
		}
		if fieldInfo.Required && !fieldCheck.IsValid() {
			errs = append(errs, causes.NewRequiredFieldError(fieldName))
			continue
		}
		if !fieldCheck.IsValid() {
			continue
		}
		if err := fieldCheck.Check(); err != nil {
			errs = append(errs, causes.NewInvalidFieldError(fieldName, fieldCheck.String(), err))
		}
		if fieldInfo.RangeMin != nil || fieldInfo.RangeMax != nil {
			a, ok := fieldAddr.(canFloat)
			if !ok {
				errs = append(errs, fmt.Errorf("could not convert %T to float for range check", fieldAddr))
				continue
			}
			checkVal := a.Float()
			if fieldInfo.RangeMin != nil && checkVal < *fieldInfo.RangeMin {
				checkErr := causes.NewInvalidFieldError(fieldName, fieldCheck.String(), fmt.Errorf("out of bounds, less than %f", *fieldInfo.RangeMin))
				errs = append(errs, checkErr)
			}
			if fieldInfo.RangeMax != nil && checkVal > *fieldInfo.RangeMax {
				checkErr := causes.NewInvalidFieldError(fieldName, fieldCheck.String(), fmt.Errorf("out of bounds, greater than %f", *fieldInfo.RangeMax))
				errs = append(errs, checkErr)
			}
		}
		if len(fieldInfo.EnumValues) > 0 {
			a, ok := fieldAddr.(canInt)
			if !ok {
				errs = append(errs, fmt.Errorf("could not convert %T to int for enum check", fieldAddr))
				continue
			}
			checkVal := int64(a.Int())
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

func ReflectUpdateKeys(emap *EntityMap, ent any) error {
	fields := entityMapperCache.GetStructTagMap(ent)
	for fieldName, fieldInfo := range fields {
		if fieldInfo.Target == "" {
			continue
		}
		elem := reflect.ValueOf(ent).Elem()
		fieldValue := reflectx.FieldByIndexes(elem, fieldInfo.Index).Addr().Interface()
		fieldSet, ok := fieldValue.(canSet)
		if !ok {
			return fmt.Errorf("EntityMap ReflectUpdate cannot be used on field '%s', does not support Set()", fieldName)
		}
		eid := fieldSet.String()
		if eid == "" {
			continue
		}
		newId, ok := emap.Get(fieldInfo.Target, eid)
		if !ok {
			return TrySetField(causes.NewInvalidReferenceError(fieldName, eid), fieldName)
		}
		fieldSet.Set(newId)
	}
	return nil
}
