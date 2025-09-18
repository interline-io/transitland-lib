package tags

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/interline-io/log"
	"github.com/jmoiron/sqlx/reflectx"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// FieldInfo contains the parsed tag values for a single attribute.
type FieldInfo struct {
	Name           string
	Required       bool
	Target         string
	Index          []int
	GreaterThan    *float64
	LessThan       *float64
	GreaterOrEqual *float64
	LessOrEqual    *float64
	EnumValues     []int64
}

// FieldMap contains all the parsed tags for a struct.
type FieldMap map[string]*FieldInfo

// Cache caches the result of field/tag parsing for each type.
type Cache struct {
	Mapper  *reflectx.Mapper
	lock    sync.Mutex
	typemap map[string]FieldMap
}

// NewCache initializes a new cache.
func NewCache(mapper *reflectx.Mapper) *Cache {
	return &Cache{
		Mapper:  mapper,
		typemap: map[string]FieldMap{},
	}
}

// GetStructTagMap .
func (c *Cache) GetStructTagMap(ent interface{}) FieldMap {
	c.lock.Lock()
	t := reflect.TypeOf(ent).String()
	m, ok := c.typemap[t]
	if !ok {
		ctx := context.TODO()
		logTag := func(key string, optVal string, err error) {
			log.For(ctx).Error().Msgf(
				"error constructing field map for type %T: could not parse tag '%s' with value '%s' as *float64: %s",
				ent,
				key,
				optVal,
				err.Error(),
			)
		}
		m = FieldMap{}
		fields := c.Mapper.TypeMap(reflect.TypeOf(ent))
		for i, fi := range fields.Index {
			_ = i
			if fi.Name == "" {
				fi.Name = ToSnakeCase(fi.Field.Name)
			}
			// TODO: This is a very bad hack. Figure out the correct way to exclude embedded fields with tags.
			if fi.Name == "id" || fi.Name == "val" || fi.Name == "valid" {
				continue
			}
			if fi.Embedded || strings.Contains(fi.Path, ".") {
				continue
			}
			mfi := FieldInfo{
				Name:   fi.Name,
				Index:  fi.Index,
				Target: fi.Field.Tag.Get("target"),
			}

			_, mfi.Required = fi.Options["required"]
			if optVal := fi.Field.Tag.Get("gt"); optVal != "" {
				if optParse, err := strconv.ParseFloat(optVal, 64); err != nil {
					logTag("gt", optVal, err)
				} else {
					mfi.GreaterThan = &optParse
				}
			}
			if optVal := fi.Field.Tag.Get("gte"); optVal != "" {
				if optParse, err := strconv.ParseFloat(optVal, 64); err != nil {
					logTag("gte", optVal, err)
				} else {
					mfi.GreaterOrEqual = &optParse
				}
			}
			if optVal := fi.Field.Tag.Get("lt"); optVal != "" {
				if optParse, err := strconv.ParseFloat(optVal, 64); err != nil {
					logTag("lte", optVal, err)
				} else {
					mfi.LessThan = &optParse
				}
			}
			if optVal := fi.Field.Tag.Get("lte"); optVal != "" {
				if optParse, err := strconv.ParseFloat(optVal, 64); err != nil {
					logTag("lte", optVal, err)
				} else {
					mfi.LessOrEqual = &optParse
				}
			}
			if optVal := fi.Field.Tag.Get("range"); optVal != "" {
				p := strings.Split(optVal, ",")
				if len(p) > 0 && p[0] != "" {
					if optParse, err := strconv.ParseFloat(p[0], 64); err != nil {
						logTag("range", optVal, err)
					} else {
						mfi.GreaterOrEqual = &optParse
					}
				}
				if len(p) > 1 && p[1] != "" {
					if optParse, err := strconv.ParseFloat(p[1], 64); err != nil {
						logTag("range", optVal, err)
					} else {
						mfi.LessOrEqual = &optParse
					}
				}
			}
			if optVal := fi.Field.Tag.Get("enum"); optVal != "" {
				for _, enumVal := range strings.Split(optVal, ",") {
					if optParse, err := strconv.ParseInt(enumVal, 10, 64); err != nil {
						log.For(ctx).Error().Msgf(
							"error constructing field map for type %T: could not parse tag 'enum' with value '%s' as []int64: %s",
							ent,
							optVal,
							err.Error(),
						)
					} else {
						mfi.EnumValues = append(mfi.EnumValues, optParse)
					}
				}
			}
			m[fi.Name] = &mfi
		}
		c.typemap[t] = m
	}
	c.lock.Unlock()
	return m
}

// Header returns the field names in the same order as the struct definition.
func (c *Cache) GetHeader(ent interface{}) ([]string, error) {
	row := []string{}
	fmap := c.GetStructTagMap(ent)
	stms := []*FieldInfo{}
	for _, stm := range fmap {
		stms = append(stms, stm)
	}
	sort.Slice(stms, func(i, j int) bool {
		for pos := 0; ; pos++ {
			if pos >= len(stms[i].Index) {
				return true
			}
			if pos >= len(stms[j].Index) {
				return false
			}
			a := stms[i].Index[pos]
			b := stms[j].Index[pos]
			if a == b {
				continue
			}
			return a < b
		}
	})
	for _, stm := range stms {
		row = append(row, stm.Name)
	}
	return row, nil
}

type canGetValue interface {
	GetValue(string) (any, bool)
}

// GetInsert returns values in the same order as the header.
func (c *Cache) GetInsert(ent any, header []string) ([]any, error) {
	var fmap FieldMap
	cgv, cgvOk := ent.(canGetValue)
	vals := make([]any, 0, len(header))
	for _, key := range header {
		var valOk bool
		var innerVal any
		if cgvOk {
			innerVal, valOk = cgv.GetValue(key)
		}
		if !valOk {
			if fmap == nil {
				fmap = c.GetStructTagMap(ent)
			}
			fi, ok := fmap[key]
			if !ok {
				// This should not happen.
				return nil, fmt.Errorf("unknown field: %s", key)
			}
			val := reflect.ValueOf(ent).Elem()
			innerVal = reflectx.FieldByIndexesReadOnly(val, fi.Index).Interface()
		}
		vals = append(vals, innerVal)
	}
	return vals, nil
}
