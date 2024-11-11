package adapters

import (
	"iter"
	"reflect"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/internal/tags"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/jmoiron/sqlx/reflectx"
)

const bufferSize = 1_000

var MapperCache = tags.NewCache(reflectx.NewMapperFunc("csv", tags.ToSnakeCase))

type Row struct {
	Header []string
	Values []any
	Hindex map[string]int
	Line   int
	Err    error
}

type RowReader interface {
	ReadRows(any) iter.Seq[Row]
}

// check for SetString interface
type canSetValue interface {
	SetValue(string, any) (bool, error)
	AddError(error)
}

type canSetLine interface {
	SetLine(int)
}

func ReadEntities[T any](reader RowReader) chan T {
	// To get Filename() or TableName()
	var entType T
	// Prepare channel
	eout := make(chan T, bufferSize)
	go func(c chan T) {
		for row := range reader.ReadRows(entType) {
			var e T
			loadRowReflect(&e, row)
			c <- e
		}
		close(c)
	}(eout)
	return eout
}

func ReadEntitiesIter[T any](reader RowReader) iter.Seq[T] {
	// To get Filename() or TableName()
	var entType T
	return func(yield func(ent T) bool) {
		for row := range reader.ReadRows(entType) {
			var e T
			loadRowReflect(&e, row)
			yield(e)
		}
	}
}

// loadRowReflect is the Reflect path
func loadRowReflect(ent any, row Row) []error {
	var errs []error
	// Return if there was a row parsing error
	if row.Err != nil {
		errs = append(errs, causes.NewRowParseError(row.Line, row.Err))
	} else {
		// Get the struct tag map
		fmap := MapperCache.GetStructTagMap(ent)
		// For each struct tag, set the field value
		entValue := reflect.ValueOf(ent).Elem()
		for i := 0; i < len(row.Header); i++ {
			if i > len(row.Values) {
				continue
			}
			fieldName := row.Header[i]
			fieldValue := row.Values[i]
			fieldInfo, ok := fmap[fieldName]

			// Add to extra fields if there's no struct tag
			if !ok {
				if extEnt, ok2 := ent.(tt.EntityWithExtra); ok2 {
					extEnt.SetExtra(fieldName, toStrv(fieldValue))
				}
				continue
			}

			// Skip if empty, special case for strings
			if fieldValue == nil {
				continue
			} else if v, ok := fieldValue.(string); ok && v == "" {
				continue
			}

			// Handle different known types
			entFieldAddr := reflectx.FieldByIndexes(entValue, fieldInfo.Index).Addr().Interface()
			if _, scanErr := tt.ConvertAssign(entFieldAddr, fieldValue); scanErr != nil {
				errs = append(errs, causes.NewFieldParseError(fieldName, toStrv(fieldValue)))
			}
		}
	}
	if len(errs) > 0 {
		if extEnt, ok := ent.(tt.EntityWithLoadErrors); ok {
			for _, err := range errs {
				extEnt.AddError(err)
			}
		}
	}
	return errs
}

func toStrv(value any) string {
	if v, ok := value.(string); ok {
		return v
	}
	strv := ""
	tt.ConvertAssign(&strv, value)
	return strv
}
