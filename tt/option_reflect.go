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
	Check() error
	IsValid() bool
	IsZero() bool
}

type HasLoadErrors interface {
	LoadErrors() []error
}

type HasConditionalErrors interface {
	ConditionalErrors() []error
}

// Error wrapping helpers
func ReflectCheckErrors(ent any) []error {
	var errs []error
	if a, ok := ent.(HasLoadErrors); ok {
		errs = append(errs, a.LoadErrors()...)
	}
	if a, ok := ent.(HasConditionalErrors); ok {
		errs = append(errs, a.ConditionalErrors()...)
	}
	entValue := reflect.ValueOf(ent).Elem()
	fmap := mapperCache.GetStructTagMap(ent)
	for fieldName, fieldInfo := range fmap {
		fmt.Println("checking field:", fieldName, "index:", fieldInfo.Index, fieldInfo.Name)
		field := reflectx.FieldByIndexes(entValue, fieldInfo.Index)
		fieldAddr := field.Addr().Interface()
		if fieldAddr == nil {
			fmt.Println("\tno fieldAddr")
			continue
		}
		fieldCheck, ok := fieldAddr.(CanCheck)
		fmt.Printf("\tfield: %#v\n", fieldAddr)
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
