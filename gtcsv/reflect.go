package gtcsv

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/internal/tags"
)

type canSetString interface {
	SetString(string, string) error
	AddError(error)
}

// loadRow selects the fastest method for loading an entity.
func loadRow(ent gotransit.Entity, row Row) {
	// Check for fast path
	if entfast, ok := ent.(canSetString); ok {
		loadRowFast(entfast, row)
	} else {
		loadRowReflect(ent, row)
	}
}

// LoadRowFast uses a non-reflect fast path for entities that support SetString and AddError.
func loadRowFast(ent canSetString, row Row) {
	// Return if there was a row parsing error
	if row.Err != nil {
		ent.AddError(causes.NewFileParseError(row.Line, row.Err))
		return
	}
	header := row.Header
	value := row.Row
	for i := 0; i < len(value) && i < len(value); i++ {
		if err := ent.SetString(header[i], value[i]); err != nil {
			ent.AddError(err)
		}
	}
}

// loadRowStopTime skips an interface check, since this is 90%+ of gtfs data
func loadRowStopTime(ent *gotransit.StopTime, row Row) {
	if row.Err != nil {
		ent.AddError(causes.NewFileParseError(row.Line, row.Err))
		return
	}
	header := row.Header
	value := row.Row
	for i := 0; i < len(value) && i < len(value); i++ {
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
		strv, _ := row.Get(h)
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
		var p error
		switch valueField.Interface().(type) {
		case string:
			valueField.SetString(strv)
		case int:
			v, e := strconv.ParseInt(strv, 0, 0)
			p = e
			valueField.SetInt(v)
		case bool:
			v, e := strconv.ParseBool(strv)
			p = e
			valueField.SetBool(v)
		case float64:
			v, e := strconv.ParseFloat(strv, 64)
			p = e
			valueField.SetFloat(v)
		case time.Time:
			v, e := time.Parse("20060102", strv)
			p = e
			valueField.Set(reflect.ValueOf(v))
		case gotransit.WideTime:
			v, e := gotransit.NewWideTime(strv)
			p = e
			valueField.Set(reflect.ValueOf(v))
		default:
			p = errors.New("unknown field type")
		}
		// Was there an error?
		if p != nil {
			errs = append(errs, causes.NewFieldParseError(k.Csv, strv))
		}
	}
	for _, err := range errs {
		ent.AddError(err)
	}
}

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

type canGetString interface {
	GetString(string) (string, error)
}

// dumpRow returns a []string for the Entity.
func dumpRow(ent gotransit.Entity, header []string) ([]string, error) {
	row := []string{}
	var p error
	// Fast path
	if a, ok := ent.(canGetString); ok {
		for _, k := range header {
			v, err := a.GetString(k)
			if err != nil {
				p = err
			}
			row = append(row, v)
		}
		return row, p
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
		value := ""
		switch v := val.Field(field.Index).Interface().(type) {
		case string:
			// TODO: Remove special case
			if strings.HasSuffix(k, "_id") && v == "0" {
				v = ""
			}
			value = v
		case int:
			// TODO: Remove special case
			if v == math.MaxInt64 {
				value = ""
			} else {
				value = strconv.Itoa(v)
			}
		case bool:
			value = strconv.Itoa(boolToInt(v))
		case float64:
			if math.IsNaN(v) {
				value = ""
			} else {
				value = fmt.Sprintf("%0.5f", v)
			}
		case time.Time:
			if v.Year() < 1970 {
				value = ""
			} else {
				value = v.Format("20060102")
			}
		case gotransit.WideTime:
			t, _ := v.String()
			value = t
		default:
			p = errors.New("unknown field type")
		}
		row = append(row, value)
	}
	return row, p
}
