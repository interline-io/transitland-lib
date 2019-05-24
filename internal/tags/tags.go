package tags

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"sync"
)

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
		m = newStructTagMap(q)
		structTagMapCache[t] = m
	}
	structTagMapLock.Unlock()
	return m
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
