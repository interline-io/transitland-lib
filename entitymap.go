package gotransit

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

// Set sets the old and new ID for an Entity.
func (emap *EntityMap) Set(ent Entity, oldid string, newid string) error {
	efn := ent.Filename()
	if i, ok := emap.ids[efn]; ok {
		i[oldid] = newid
	} else {
		emap.ids[efn] = map[string]string{oldid: newid}
	}
	return nil
}

// Get returns the new ID for an Entity.
func (emap *EntityMap) Get(ent Entity) (string, bool) {
	efn := ent.Filename()
	eid := ent.EntityID()
	if i, ok := emap.ids[efn]; ok {
		a, ok := i[eid]
		return a, ok
	}
	emap.ids[efn] = map[string]string{}
	return "", false
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
