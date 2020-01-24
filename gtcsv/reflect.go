package gtcsv

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/internal/tags"
)

// check for SetString interface
type canSetString interface {
	SetString(string, string) error
	AddError(error)
}

type canString interface {
	String() string
}

type canScan interface {
	Scan(src interface{}) error
}

// TODO: use reflectx.Mapper for consistency

// SetString //

// SetString convenience method; checks for SetString method.
func SetString(ent gotransit.Entity, key string, value string) error {
	if fastent, ok := ent.(canSetString); ok {
		return fastent.SetString(key, value)
	}
	fmap := tags.GetStructTagMap(ent)
	k, ok := fmap[key]
	if !ok {
		// only SetExtra when loading from csv...
		// ent.SetExtra(key, value)
		return errors.New("unknown field")
	}
	// Already known valid field
	elem := reflect.ValueOf(ent).Elem()
	valueField := elem.Field(k.Index)
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
	case int:
		v, e := strconv.ParseInt(strv, 0, 0)
		p = e
		valueField.SetInt(v)
	case float64:
		v, e := strconv.ParseFloat(strv, 64)
		p = e
		valueField.SetFloat(v)
	case time.Time:
		v, e := time.Parse("20060102", strv)
		p = e
		valueField.Set(reflect.ValueOf(v))
	default:
		z := valueField.Addr().Interface()
		if cs, ok := z.(canScan); ok {
			p = cs.Scan(strv)
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

// GetString convenience method; gets a String representation of a field.
func GetString(ent gotransit.Entity, key string) (string, error) {
	if fastent, ok := ent.(canGetString); ok {
		return fastent.GetString(key)
	}
	fmap := tags.GetStructTagMap(ent)
	k, ok := fmap[key]
	if !ok {
		// only SetExtra when loading from csv...
		// ent.SetExtra(key, value)
		return "", errors.New("unknown field")
	}
	// Already known valid field
	elem := reflect.ValueOf(ent).Elem()
	valueField := elem.Field(k.Index)
	v, err := valGetString(valueField, key)
	if err != nil {
		return "", err
	}
	return v, nil
}

// valGetString returns a CSV representation of the field.
func valGetString(valueField reflect.Value, k string) (string, error) {
	value := ""
	switch v := valueField.Interface().(type) {
	case string:
		value = v
	case int:
		value = strconv.Itoa(v)
	case float64:
		if math.IsNaN(v) {
			value = ""
		} else {
			value = fmt.Sprintf("%0.5f", v)
		}
	case time.Time:
		if v.IsZero() {
			value = ""
		} else {
			value = v.Format("20060102")
		}
	default:
		z := valueField.Addr().Interface()
		if cs, ok := z.(canString); ok {
			value = cs.String()
		} else {
			return "", errors.New("field not stringable")
		}
	}
	return value, nil
}

// Loading: fast and reflect paths //

// loadRow selects the fastest method for loading an entity.
func loadRow(ent gotransit.Entity, row Row) {
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
		ent.AddError(causes.NewFileParseError(row.Line, row.Err))
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
func loadRowReflect(ent gotransit.Entity, row Row) {
	// Return if there was a row parsing error
	if row.Err != nil {
		ent.AddError(causes.NewFileParseError(row.Line, row.Err))
		return
	}
	// Get the struct tag map
	fmap := tags.GetStructTagMap(ent)
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
				switch val.Field(k.Index).Interface().(type) {
				case string:
				default:
					errs = append(errs, causes.NewRequiredFieldError(h))
				}
			}
			continue
		}
		// Handle different known types
		valueField := val.Field(k.Index)
		if err := valSetString(valueField, strv); err != nil {
			errs = append(errs, causes.NewFieldParseError(k.Csv, strv))
		}
	}
	for _, err := range errs {
		ent.AddError(err)
	}
}

// Dumping: fast and reflect paths //

// dumpHeader returns the header for an Entity.
func dumpHeader(ent gotransit.Entity) ([]string, error) {
	row := []string{}
	fmap := tags.GetStructTagMap(ent)
	// Order fields
	stms := []tags.StructTagMap{}
	for _, stm := range fmap {
		stms = append(stms, stm)
	}
	sort.Slice(stms, func(i, j int) bool { return stms[i].Index < stms[j].Index })
	// Return known CSV fields
	for _, stm := range stms {
		if len(stm.Csv) > 0 {
			row = append(row, stm.Csv)
		}
	}
	return row, nil
}

// dumpRow returns a []string for the Entity.
func dumpRow(ent gotransit.Entity, header []string) ([]string, error) {
	row := []string{}
	// Fast path
	if a, ok := ent.(canGetString); ok {
		for _, k := range header {
			v, err := a.GetString(k)
			if err != nil {
				return row, err
			}
			row = append(row, v)
		}
		return row, nil
	}
	// Reflect path
	val := reflect.ValueOf(ent).Elem()
	fmap := tags.GetStructTagMap(ent)
	for _, k := range header {
		field, ok := fmap[k]
		if !ok {
			continue
		}
		if len(field.Csv) == 0 {
			continue
		}
		valueField := val.Field(field.Index)
		value, err := valGetString(valueField, k)
		if err != nil {
			return row, err
		}
		row = append(row, value)
	}
	return row, nil
}
