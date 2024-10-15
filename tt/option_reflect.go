package tt

import (
	"fmt"
	"reflect"

	"github.com/interline-io/log"
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
	IsValid() bool
	Check() error
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

		// Field cant be checked
		fieldCheck, ok := fieldAddr.(CanReflectCheck)
		if !ok {
			log.Error().Msgf("type %T does not support reflect based error checks", fieldAddr)
			continue
		}
		if fieldInfo.Required && !fieldCheck.IsValid() {
			errs = append(errs, causes.NewRequiredFieldError(fieldName))
			continue
		}
		if !fieldCheck.IsValid() {
			continue
		}
		// Only perform additional checks if field is present
		if err := fieldCheck.Check(); err != nil {
			errs = append(errs, causes.NewInvalidFieldError(fieldName, fieldCheck.String(), err))
		}
		if fieldInfo.RangeMin != nil || fieldInfo.RangeMax != nil {
			if a, ok := fieldAddr.(canFloat); ok {
				checkVal := a.Float()
				if fieldInfo.RangeMin != nil && checkVal < *fieldInfo.RangeMin {
					checkErr := causes.NewInvalidFieldError(fieldName, fieldCheck.String(), fmt.Errorf("out of bounds, less than %f", *fieldInfo.RangeMin))
					errs = append(errs, checkErr)
				}
				if fieldInfo.RangeMax != nil && checkVal > *fieldInfo.RangeMax {
					checkErr := causes.NewInvalidFieldError(fieldName, fieldCheck.String(), fmt.Errorf("out of bounds, greater than %f", *fieldInfo.RangeMax))
					errs = append(errs, checkErr)
				}
			} else {
				log.Error().Msgf("could not convert %T to float for range check", fieldAddr)
			}
		}
		if len(fieldInfo.EnumValues) > 0 {
			if a, ok := fieldAddr.(canInt); ok {
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
			} else {
				log.Error().Msgf("could not convert %T to int for enum check", fieldAddr)
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
			log.Error().Msgf("type %T does not support reflect based reference checks", fieldAddr)
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
