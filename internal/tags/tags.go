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
	"sync/atomic"
	"time"

	"github.com/interline-io/log"
	"github.com/jmoiron/sqlx/reflectx"
)

// SortKind is the underlying primitive class of a struct field, used
// to drive type-aware sorting on serialized CSV cells.
type SortKind int

const (
	SortKindUnknown SortKind = iota
	SortKindString
	SortKindInt
	SortKindFloat
	SortKindDate
)

// optionTypeHint is satisfied by wrappers (e.g., tt.Option[T]) that expose
// their inner type. Defining the interface here keeps tt independent of tags.
type optionTypeHint interface {
	OptionType() reflect.Type
}

func inferKind(t reflect.Type) SortKind {
	if t == nil {
		return SortKindUnknown
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return SortKindString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return SortKindInt
	case reflect.Float32, reflect.Float64:
		return SortKindFloat
	}
	if t.Kind() == reflect.Struct {
		if t == reflect.TypeOf(time.Time{}) {
			return SortKindDate
		}
		zero := reflect.New(t).Elem().Interface()
		if h, ok := zero.(optionTypeHint); ok {
			return inferKind(h.OptionType())
		}
	}
	return SortKindUnknown
}

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
	AliasOf        string
	Required       bool
	Warn           bool
	Target         string
	Index          []int
	GreaterThan    *float64
	LessThan       *float64
	GreaterOrEqual *float64
	LessOrEqual    *float64
	EnumValues     []int64
	SortOrder      int
	Kind           SortKind
}

// IsAlias reports whether this entry is an additional name for another field
// rather than the field's own name.
func (fi *FieldInfo) IsAlias() bool {
	return fi.AliasOf != ""
}

// FieldMap contains all the parsed tags for a struct.
type FieldMap map[string]*FieldInfo

// typeCache is an immutable snapshot of the types parsed so far, keyed by
// reflect.Type rather than by its name so that a lookup hashes a pointer
// rather than the string Type.String() builds.
type typeCache struct {
	typemap map[reflect.Type]FieldMap
	warnmap map[reflect.Type][]*FieldInfo
}

// Cache caches the result of field/tag parsing for each type.
//
// The check passes ask this about every entity in a copy, which is the
// highest volume call in the library, so readers take the current snapshot
// with a single atomic load and never a lock: a mutex here, even a read
// shared one, would put every entity of every file through one cache line.
// Writers are serialized by lock and publish a copy. The set of types is
// small, fixed by the entity structs, and filled in the first moments of a
// copy, so copying it on a miss costs nothing that lasts.
type Cache struct {
	Mapper *reflectx.Mapper
	lock   sync.Mutex
	cached atomic.Pointer[typeCache]
}

// NewCache initializes a new cache.
func NewCache(mapper *reflectx.Mapper) *Cache {
	c := &Cache{Mapper: mapper}
	c.cached.Store(&typeCache{
		typemap: map[reflect.Type]FieldMap{},
		warnmap: map[reflect.Type][]*FieldInfo{},
	})
	return c
}

// GetStructTagMap .
func (c *Cache) GetStructTagMap(ent interface{}) FieldMap {
	t := reflect.TypeOf(ent)
	if m, ok := c.cached.Load().typemap[t]; ok {
		return m
	}
	m, _ := c.buildTypeMap(t, ent)
	return m
}

// GetWarnFields returns the fields tagged with the "warn" option, in no
// particular order.
//
// The answer is collected when the type is first mapped, because the warning
// pass asks this of every entity in a copy while almost no type has such a
// field. A hit costs one map lookup rather than a walk over the type's fields.
func (c *Cache) GetWarnFields(ent interface{}) []*FieldInfo {
	t := reflect.TypeOf(ent)
	if warnFields, ok := c.cached.Load().warnmap[t]; ok {
		return warnFields
	}
	_, warnFields := c.buildTypeMap(t, ent)
	return warnFields
}

// buildTypeMap returns the field map and the warn tagged fields for a type,
// parsing the type's tags on first use and publishing a snapshot that
// includes them. ent is needed only to name the type in a log message.
func (c *Cache) buildTypeMap(t reflect.Type, ent interface{}) (FieldMap, []*FieldInfo) {
	c.lock.Lock()
	defer c.lock.Unlock()
	// Another goroutine may have parsed this type while this one waited.
	cur := c.cached.Load()
	m, ok := cur.typemap[t]
	var warnFields []*FieldInfo
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
		aliases := map[string]*FieldInfo{}
		fields := c.Mapper.TypeMap(t)
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
				Kind:   inferKind(fi.Field.Type),
			}

			_, mfi.Required = fi.Options["required"]
			_, mfi.Warn = fi.Options["warn"]
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
					logTag("lt", optVal, err)
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
			if optVal := fi.Field.Tag.Get("standardized_sort"); optVal != "" {
				if optParse, err := strconv.Atoi(optVal); err != nil {
					log.For(ctx).Error().Msgf(
						"error constructing field map for type %T: could not parse tag 'standardized_sort' with value '%s' as int: %s",
						ent,
						optVal,
						err.Error(),
					)
				} else {
					mfi.SortOrder = optParse
				}
			}
			// The alias is read from this mapper's own tag options, so an alias
			// declared for one tag namespace stays invisible to the others.
			if optVal := fi.Options["alias"]; optVal != "" {
				if prev, exists := aliases[optVal]; exists {
					log.For(ctx).Error().Msgf(
						"error constructing field map for type %T: fields '%s' and '%s' both declare the alias '%s'; ignoring the one on '%s'",
						ent, prev.Name, fi.Name, optVal, prev.Name,
					)
				}
				aliases[optVal] = &mfi
			}
			m[fi.Name] = &mfi
			if mfi.Warn {
				warnFields = append(warnFields, &mfi)
			}
		}
		// Register aliases after every field is mapped, so a field's own name is
		// never displaced by another field's alias. An alias entry carries only
		// what is needed to locate the field: validation tags stay on the field's
		// own entry, so a value is checked once, under the name the file uses.
		for aliasName, fi := range aliases {
			if _, exists := m[aliasName]; exists {
				log.For(ctx).Error().Msgf(
					"error constructing field map for type %T: alias '%s' is already the name of a field; ignoring it",
					ent, aliasName,
				)
				continue
			}
			m[aliasName] = &FieldInfo{
				Name:    aliasName,
				AliasOf: fi.Name,
				Index:   fi.Index,
				Kind:    fi.Kind,
			}
		}
		next := &typeCache{
			typemap: make(map[reflect.Type]FieldMap, len(cur.typemap)+1),
			warnmap: make(map[reflect.Type][]*FieldInfo, len(cur.warnmap)+1),
		}
		for k, v := range cur.typemap {
			next.typemap[k] = v
		}
		for k, v := range cur.warnmap {
			next.warnmap[k] = v
		}
		next.typemap[t] = m
		next.warnmap[t] = warnFields
		c.cached.Store(next)
	} else {
		warnFields = cur.warnmap[t]
	}
	return m, warnFields
}

// GetSortColumns returns the entity's fields tagged with standardized_sort,
// sorted ascending by SortOrder.
func (c *Cache) GetSortColumns(ent interface{}) []*FieldInfo {
	fmap := c.GetStructTagMap(ent)
	var cols []*FieldInfo
	for _, fi := range fmap {
		if fi.SortOrder > 0 {
			cols = append(cols, fi)
		}
	}
	sort.Slice(cols, func(i, j int) bool { return cols[i].SortOrder < cols[j].SortOrder })
	return cols
}

// Header returns the field names in the same order as the struct definition.
func (c *Cache) GetHeader(ent interface{}) ([]string, error) {
	row := []string{}
	fmap := c.GetStructTagMap(ent)
	stms := []*FieldInfo{}
	for _, stm := range fmap {
		if stm.IsAlias() {
			continue
		}
		stms = append(stms, stm)
	}
	sort.Slice(stms, func(i, j int) bool {
		for pos := 0; ; pos++ {
			if pos >= len(stms[i].Index) {
				return pos < len(stms[j].Index)
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
