package tags

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

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
	Name     string
	Required bool
	Target   string
	Index    []int
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
			_, required := fi.Options["required"]
			m[fi.Name] = &FieldInfo{
				Name:     fi.Name,
				Required: required,
				Index:    fi.Index,
				Target:   fi.Field.Tag.Get("target"),
			}
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
