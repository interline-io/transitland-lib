package tlcsv

import (
	"errors"
	"reflect"

	"github.com/interline-io/transitland-lib/internal/tags"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/jmoiron/sqlx/reflectx"
)

var MapperCache = tags.NewCache(reflectx.NewMapperFunc("csv", tags.ToSnakeCase))

// check for SetString interface
type canSetString interface {
	SetString(string, string) error
	AddError(error)
}

type canSetLine interface {
	SetLine(int)
}

// SetString //

// SetString convenience method; checks for SetString method.
func SetString(ent tl.Entity, key string, value string) error {
	if fastent, ok := ent.(canSetString); ok {
		return fastent.SetString(key, value)
	}
	fmap := MapperCache.GetStructTagMap(ent)
	k, ok := fmap[key]
	if !ok || k == nil {
		return errors.New("unknown field")
	}
	// Already known valid field
	elem := reflect.ValueOf(ent).Elem()
	fieldValue := reflectx.FieldByIndexes(elem, k.Index).Addr().Interface()
	if err := tt.FromCsv(fieldValue, value); err != nil {
		return err
	}
	return nil
}

// GetString //

type canGetString interface {
	GetString(string) (string, error)
}

// GetString convenience method; gets a string representation of a field.
func GetString(ent tl.Entity, key string) (string, error) {
	if fastent, ok := ent.(canGetString); ok {
		return fastent.GetString(key)
	}
	fmap := MapperCache.GetStructTagMap(ent)
	k, ok := fmap[key]
	if !ok || k == nil {
		return "", errors.New("unknown field")
	}
	// Already known valid field
	elem := reflect.ValueOf(ent).Elem()
	valueField := reflectx.FieldByIndexesReadOnly(elem, k.Index)
	v, err := tt.ToCsv(valueField.Interface())
	if err != nil {
		return "", err
	}
	return v, nil
}

// Loading: fast and reflect paths //

// loadRow selects the fastest method for loading an entity.
func loadRow(ent any, row Row) []error {
	// Check for fast path
	var errs []error
	if entfast, ok := ent.(canSetString); ok {
		errs = loadRowFast(entfast, row)
	} else {
		errs = loadRowReflect(ent, row)
	}
	if v, ok := ent.(canSetLine); ok {
		v.SetLine(row.Line)
	}
	return errs
}

// LoadRowFast uses a fast path for entities that support SetString.
func loadRowFast(ent canSetString, row Row) []error {
	var errs []error
	// Return if there was a row parsing error
	if row.Err != nil {
		errs = append(errs, causes.NewRowParseError(row.Line, row.Err))
		return errs
	}
	header := row.Header
	value := row.Row
	for i := 0; i < len(value) && i < len(header); i++ {
		if err := ent.SetString(header[i], value[i]); err != nil {
			errs = append(errs, err)
		}
	}
	for _, err := range errs {
		ent.AddError(err)
	}
	return errs
}

// loadRowReflect is the Reflect path
func loadRowReflect(ent interface{}, row Row) []error {
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
			fieldName := row.Header[i]
			strv := ""
			if i < len(row.Row) {
				strv = row.Row[i]
			}
			fieldInfo, ok := fmap[fieldName]
			// Add to extra fields if there's no struct tag
			if !ok {
				if extEnt, ok2 := ent.(tl.EntityWithExtra); ok2 {
					extEnt.SetExtra(fieldName, strv)
				}
				continue
			}
			// Skip if empty and not required
			if len(strv) == 0 {
				if fieldInfo.Required {
					// empty string type shandled in regular validators; avoid double errors
					switch reflectx.FieldByIndexes(entValue, fieldInfo.Index).Interface().(type) {
					case string:
					default:
						errs = append(errs, causes.NewRequiredFieldError(fieldName))
					}
				}
				continue
			}
			// Handle different known types
			fieldValue := reflectx.FieldByIndexes(entValue, fieldInfo.Index).Addr().Interface()
			if err := tt.FromCsv(fieldValue, strv); err != nil {
				errs = append(errs, causes.NewFieldParseError(fieldName, strv))
			}
		}
	}
	if len(errs) > 0 {
		if extEnt, ok := ent.(tl.EntityWithErrors); ok {
			for _, err := range errs {
				extEnt.AddError(err)
			}
		}
	}
	return errs
}

// Dumping: fast and reflect paths //

// dumpHeader returns the header for an Entity.
func dumpHeader(ent tl.Entity) ([]string, error) {
	return MapperCache.GetHeader(ent)
}

// dumpRow returns a []string for the Entity.
func dumpRow(ent tl.Entity, header []string) ([]string, error) {
	row := make([]string, 0, len(header))
	// Fast path
	if a, ok := ent.(canGetString); ok {
		for _, k := range header {
			v, err := a.GetString(k)
			if err != nil {
				return nil, err
			}
			row = append(row, v)
		}
		return row, nil
	}
	// Reflect path
	rv, err := MapperCache.GetInsert(ent, header)
	if err != nil || len(rv) != len(header) {
		return nil, errors.New("failed to get insert values for entity")
	}
	for _, v := range rv {
		value, err := tt.ToCsv(v)
		if err != nil {
			return nil, err
		}
		row = append(row, value)
	}
	return row, nil
}
