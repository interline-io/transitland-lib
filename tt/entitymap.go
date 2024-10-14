package tt

import (
	"fmt"
	"reflect"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/internal/tags"
	"github.com/jmoiron/sqlx/reflectx"
)

var mapperCache = tags.NewCache(reflectx.NewMapperFunc("csv", tags.ToSnakeCase))

// EntityMap stores correspondances between Entity IDs, e.g. StopID -> Stop's integer ID in a database.
type EntityMap struct {
	ids map[string]map[string]string
}

// NewEntityMap returns a new EntityMap.
func NewEntityMap() *EntityMap {
	return &EntityMap{
		ids: map[string]map[string]string{},
	}
}

type canSet interface {
	String() string
	Set(string)
}

func (emap *EntityMap) ReflectUpdate(ent Entity) error {
	fields := mapperCache.GetStructTagMap(ent)
	for fieldName, fieldInfo := range fields {
		if fieldInfo.Target == "" {
			continue
		}
		elem := reflect.ValueOf(ent).Elem()
		fieldValue := reflectx.FieldByIndexes(elem, fieldInfo.Index).Addr().Interface()
		fieldSet, ok := fieldValue.(canSet)
		if !ok {
			return fmt.Errorf("EntityMap ReflectUpdate cannot be used on field '%s', does not support Set()", fieldName)
		}
		eid := fieldSet.String()
		if eid == "" {
			continue
		}
		newId, ok := emap.Get(fieldInfo.Target, eid)
		if !ok {
			return TrySetField(causes.NewInvalidReferenceError(fieldName, eid), fieldName)
		}
		fieldSet.Set(newId)
	}
	return nil
}

func (emap *EntityMap) UpdateKeyField(v canSet, efn string, fieldName string) error {
	return TrySetField(emap.UpdateKey(v, efn), fieldName)
}

func (emap *EntityMap) UpdateKey(v canSet, efn string) error {
	eid := v.String()
	if eid == "" {
		return nil
	}
	newEid, ok := emap.Get(efn, eid)
	if !ok {
		return causes.NewInvalidReferenceError(efn, eid)
	}
	v.Set(newEid)
	return nil
}

// Set directly adds an entry to the set.
func (emap *EntityMap) Set(efn string, oldid string, newid string) error {
	if i, ok := emap.ids[efn]; ok {
		i[oldid] = newid
	} else {
		emap.ids[efn] = map[string]string{oldid: newid}
	}
	return nil
}

// SetEntity sets the old and new ID for an Entity.
func (emap *EntityMap) SetEntity(ent Entity, oldid string, newid string) error {
	return emap.Set(ent.Filename(), oldid, newid)
}

// GetEntity returns the new ID for an Entity.
func (emap *EntityMap) GetEntity(ent Entity) (string, bool) {
	efn := ent.Filename()
	eid := ent.EntityID()
	return emap.Get(efn, eid)
}

// Get gets directly by filename, eid
func (emap *EntityMap) Get(efn string, eid string) (string, bool) {
	if i, ok := emap.ids[efn]; ok {
		a, ok := i[eid]
		return a, ok
	}
	emap.ids[efn] = map[string]string{}
	return "", false
}

// KeysFor returns the keys for a filename.
func (emap *EntityMap) KeysFor(efn string) []string {
	ret := []string{}
	for k := range emap.ids[efn] {
		ret = append(ret, k)
	}
	return ret
}

// Update copies values from another EntityMap.
func (emap *EntityMap) Update(other EntityMap) {
	for efn, m := range other.ids {
		if _, ok := emap.ids[efn]; !ok {
			emap.ids[efn] = map[string]string{}
		}
		for sid, eid := range m {
			emap.ids[efn][sid] = eid
		}
	}
}
