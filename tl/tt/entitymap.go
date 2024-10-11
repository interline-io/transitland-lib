package tt

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

// Set directly adds an entry to the set.
func (emap *EntityMap) Set(efn string, oldid string, newid string) error {
	if i, ok := emap.ids[efn]; ok {
		i[oldid] = newid
	} else {
		emap.ids[efn] = map[string]string{oldid: newid}
	}
	return nil
}

// Entity provides an interface for GTFS entities.
type Entity interface {
	EntityID() string
	Filename() string
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
