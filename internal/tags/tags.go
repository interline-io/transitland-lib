package tags

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx/reflectx"
)

var mapper = reflectx.NewMapperFunc("csv", toSnakeCase)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// StructTagMap contains the parsed tag values for a single attribute.
type StructTagMap struct {
	Csv       string
	Validator string
	Min       float64
	Max       float64
	Required  bool
	Index     int
}

// FieldTagMap contains all the parsed tags for a struct.
type FieldTagMap = map[string]StructTagMap

var structTagMapCache = map[string]FieldTagMap{}
var structTagMapLock sync.Mutex

// GetStructTagMap returns Struct tags.
func GetStructTagMap(q interface{}) FieldTagMap {
	structTagMapLock.Lock()
	t := fmt.Sprintf("%T", q)
	m, ok := structTagMapCache[t]
	if !ok {
		m = newStructTagMap2(q)
		structTagMapCache[t] = m
	}
	structTagMapLock.Unlock()
	return m
}

// GetInsert returns column names and a slice of placeholders or squirrel expressions.
func GetInsert(ent interface{}, header []string) ([]interface{}, error) {
	vals := make([]interface{}, 0)
	val := reflect.ValueOf(ent).Elem()
	fm := mapper.FieldMap(val)
	for _, name := range header {
		v, ok := fm[name]
		if !ok {
			// This should not happen.
			return nil, fmt.Errorf("unknown field: %s", name)
		}
		vals = append(vals, v.Interface())
	}
	return vals, nil
}

func getFieldNameIndexes(ent interface{}) ([]string, []string) {
	names := []string{}
	wraps := []string{}
	fields := mapper.TypeMap(reflect.TypeOf(ent))
	for _, fi := range fields.Index {
		if fi.Embedded == true || fi.Name == "id" || strings.Contains(fi.Path, ".") {
			continue
		}
		w := ""
		if wrap, ok := fi.Options["insert"]; ok {
			w = strings.Replace(wrap, "@", ",", -1)
		}
		names = append(names, fi.Path)
		wraps = append(wraps, w)
	}
	return names, wraps
}

// newStructTagMap returns a FieldTagMap for an Entity.
func newStructTagMap(q interface{}) FieldTagMap {
	m := FieldTagMap{}
	qtype := reflect.TypeOf(q).Elem()
	for i := 0; i < qtype.NumField(); i++ {
		t := qtype.Field(i).Tag
		ftag := StructTagMap{}
		ftag.Index = i
		ftag.Csv = t.Get("csv")
		if len(ftag.Csv) == 0 {
			continue
		}
		ftag.Validator = t.Get("validator")
		ftag.Min = math.Inf(-1)
		ftag.Max = math.Inf(1)
		if v, err := strconv.ParseFloat(t.Get("min"), 64); err == nil {
			ftag.Min = v
		}
		if v, err := strconv.ParseFloat(t.Get("max"), 64); err == nil {
			ftag.Max = v
		}
		if t.Get("required") == "true" {
			ftag.Required = true
		}
		m[ftag.Csv] = ftag
	}
	return m
}

// newStructTagMap returns a FieldTagMap for an Entity.
func newStructTagMap2(ent interface{}) FieldTagMap {
	m := FieldTagMap{}
	fields := mapper.TypeMap(reflect.TypeOf(ent))
	for i, fi := range fields.Index {
		if fi.Embedded == true || fi.Name == "id" || strings.Contains(fi.Path, ".") {
			continue
		}
		_, required := fi.Options["required"]
		m[fi.Name] = StructTagMap{
			Csv:      fi.Name,
			Required: required,
			Index:    i,
		}
	}
	return m
}
