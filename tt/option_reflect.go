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
	Check() error
	IsValid() bool
	IsZero() bool
	String() string
	Float() float64
	Int() int
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
		if err := fieldCheck.Check(); err != nil {
			errs = append(errs, causes.NewInvalidFieldError(fieldName, fieldCheck.String(), err))
			continue
		}
		if fieldInfo.Required && !fieldCheck.IsValid() {
			errs = append(errs, causes.NewRequiredFieldError(fieldName))
			continue
		}
		if fieldInfo.RangeMin != nil {

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
