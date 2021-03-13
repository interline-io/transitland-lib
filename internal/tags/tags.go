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
	Index3   int
	Index2   []int
}

// FieldMap contains all the parsed tags for a struct.
type FieldMap = map[string]*FieldInfo

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
	t := fmt.Sprintf("%T", ent)
	m, ok := c.typemap[t]
	if !ok {
		m = FieldMap{}
		fields := c.Mapper.TypeMap(reflect.TypeOf(ent))
		for i, fi := range fields.Index {
			_ = i
			if fi.Name == "" {
				fi.Name = ToSnakeCase(fi.Field.Name)
			}
			if fi.Embedded == true || fi.Name == "id" || strings.Contains(fi.Path, ".") {
				continue
			}
			_, required := fi.Options["required"]
			m[fi.Name] = &FieldInfo{
				Name:     fi.Name,
				Required: required,
				Index2:   fi.Index,
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
	sort.Slice(stms, func(i, j int) bool { return stms[i].Index2[0] < stms[j].Index2[0] })
	for _, stm := range stms {
		row = append(row, stm.Name)
	}
	return row, nil
}

// GetInsert returns values in the same order as the header.
func (c *Cache) GetInsert(ent interface{}, header []string) ([]interface{}, error) {
	fmap := c.GetStructTagMap(ent)
	vals := make([]interface{}, 0)
	val := reflect.ValueOf(ent).Elem()
	for _, key := range header {
		fi, ok := fmap[key]
		if !ok {
			// This should not happen.
			return nil, fmt.Errorf("unknown field: %s index: %d", key, fi.Index2)
		}
		v := reflectx.FieldByIndexesReadOnly(val, fi.Index2)
		vals = append(vals, v.Interface())
	}
	return vals, nil
}
