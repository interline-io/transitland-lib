package tlcsv

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/interline-io/transitland-lib/internal/tags"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/jmoiron/sqlx/reflectx"
)

var MapperCache = tags.NewCache(reflectx.NewMapperFunc("csv", tags.ToSnakeCase))

// check for SetString interface
type canSetString interface {
	SetString(string, string) error
	AddError(error)
}

// check for Value
type canValue interface {
	Value() (driver.Value, error)
}

type canString interface {
	String() string
}

type canScan interface {
	Scan(src interface{}) error
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
	valueField := reflectx.FieldByIndexes(elem, k.Index) // elem.Field(k.Index)
	if err := valSetString(valueField, value); err != nil {
		return err
	}
	return nil
}

// valSetString sets the field from a CSV representation of the value.
func valSetString(valueField reflect.Value, strv string) error {
	var p error
	switch valueField.Interface().(type) {
	case string:
		valueField.SetString(strv)
	case int, int64:
		v, e := strconv.ParseInt(strv, 0, 0)
		p = e
		valueField.SetInt(v)
	case float64:
		v, e := strconv.ParseFloat(strv, 64)
		p = e
		valueField.SetFloat(v)
	case bool:
		if strv == "true" {
			valueField.SetBool(true)
		} else {
			valueField.SetBool(false)
		}
	case time.Time:
		v, e := time.Parse("20060102", strv)
		p = e
		valueField.Set(reflect.ValueOf(v))
	default:
		z := valueField.Addr().Interface()
		if cs, ok := z.(canScan); ok {
			p = cs.Scan(strv)
			if p != nil {
				cs.Scan(nil) // Reset valid to false
			}
		} else {
			p = errors.New("field not scannable")
		}
	}
	return p
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
	valueField := reflectx.FieldByIndexesReadOnly(elem, k.Index) // .Field(k.Index)
	v, err := valGetString(valueField, key)
	if err != nil {
		return "", err
	}
	return v, nil
}

// valGetString returns a string representation of the field.
func valGetString(valueField reflect.Value, k string) (string, error) {
	value := ""
	rfi := valueField.Interface()
	if v, ok := rfi.(tl.WideTime); ok {
		return v.String(), nil
	}
	if v, ok := rfi.(canValue); ok {
		var err error
		rfi, err = v.Value()
		if err != nil {
			return "", err
		}
	}
	switch v := rfi.(type) {
	case nil:
		value = ""
	case string:
		value = v
	case int:
		value = strconv.Itoa(v)
	case int64:
		value = strconv.Itoa(int(v))
	case bool:
		if v {
			value = "true"
		} else {
			value = "false"
		}
	case float64:
		if math.IsNaN(v) {
			value = ""
		} else if v > -100_000 && v < 100_000 {
			// use pretty %g formatting but avoid exponents
			value = fmt.Sprintf("%g", v)
		} else {
			value = fmt.Sprintf("%0.5f", v)
		}
	case time.Time:
		if v.IsZero() {
			value = ""
		} else {
			value = v.Format("20060102")
		}
	case []byte:
		value = string(v)
	case canString:
		value = v.String()
	default:
		return "", fmt.Errorf("can not convert field '%s' to string, type %T", k, v)
	}
	return value, nil
}

// Loading: fast and reflect paths //

// loadRow selects the fastest method for loading an entity.
func loadRow(ent tl.Entity, row Row) {
	// Check for fast path
	if entfast, ok := ent.(canSetString); ok {
		loadRowFast(entfast, row)
	} else {
		loadRowReflect(ent, row)
	}
}

// LoadRowFast uses a fast path for entities that support SetString and AddError.
func loadRowFast(ent canSetString, row Row) {
	// Return if there was a row parsing error
	if row.Err != nil {
		ent.AddError(causes.NewRowParseError(row.Line, row.Err))
		return
	}
	header := row.Header
	value := row.Row
	for i := 0; i < len(value) && i < len(header); i++ {
		if err := ent.SetString(header[i], value[i]); err != nil {
			ent.AddError(err)
		}
	}
}

// loadRowReflect is the Reflect path
func loadRowReflect(ent tl.Entity, row Row) {
	// Return if there was a row parsing error
	if row.Err != nil {
		ent.AddError(causes.NewRowParseError(row.Line, row.Err))
		return
	}
	// Get the struct tag map
	fmap := MapperCache.GetStructTagMap(ent)
	errs := []error{}
	// For each struct tag, set the field value
	val := reflect.ValueOf(ent).Elem()
	for _, h := range row.Header {
		strv, ok := row.Get(h)
		if !ok {
			strv = ""
		}
		k, ok := fmap[h]
		// Add to extra fields if there's no struct tag
		if !ok {
			ent.SetExtra(h, strv)
			continue
		}
		// Skip if empty and not required
		if len(strv) == 0 {
			if k.Required {
				// empty string type shandled in regular validators; avoid double errors
				switch reflectx.FieldByIndexes(val, k.Index).Interface().(type) {
				case string:
				default:
					errs = append(errs, causes.NewRequiredFieldError(h))
				}
			}
			continue
		}
		// Handle different known types
		valueField := reflectx.FieldByIndexes(val, k.Index)
		if err := valSetString(valueField, strv); err != nil {
			errs = append(errs, causes.NewFieldParseError(k.Name, strv))
		}
	}
	for _, err := range errs {
		ent.AddError(err)
	}
}

// Dumping: fast and reflect paths //

// dumpHeader returns the header for an Entity.
func dumpHeader(ent tl.Entity) ([]string, error) {
	return MapperCache.GetHeader(ent)
}

// dumpRow returns a []string for the Entity.
func dumpRow(ent tl.Entity, header []string) ([]string, error) {
	row := []string{}
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
	for i, v := range rv {
		value, err := valGetString(reflect.ValueOf(v), header[i])
		if err != nil {
			return nil, err
		}
		row = append(row, value)
	}
	return row, nil
}
