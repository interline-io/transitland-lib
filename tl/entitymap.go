package tl

type EMap interface {
	Get(string, string) error
	GetEntity(Entity, string) error
	Set(string, string, string) error
	SetEntity(Entity, string, string) error
	KeysFor(string) []string
	GetStopGeometry(string) (Point, bool)
	GetShapeGeometry(string) (LineString, bool)
}

// EntityMap stores correspondances between Entity IDs, e.g. StopID -> Stop's integer ID in a database.
type EntityMap struct {
	ids             map[string]map[string]string
	stopGeometries  map[string]Point
	shapeGeometries map[string]LineString
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

// GetStopGeometry returns a cached stop geometry.
func (emap *EntityMap) GetStopGeometry(eid string) (Point, bool) {
	a, ok := emap.stopGeometries[eid]
	return a, ok
}

// GetShapeGeometry returns a cached shape geometry.
func (emap *EntityMap) GetShapeGeometry(eid string) (LineString, bool) {
	a, ok := emap.shapeGeometries[eid]
	return a, ok
}

// AddStopGeometry adds a stop geometry to the cache.
func (emap *EntityMap) AddStopGeometry(eid string, g Point) {
	emap.stopGeometries[eid] = g
}

// AddShapeGeometry adds a shape geometry to the cache.
func (emap *EntityMap) AddShapeGeometry(eid string, g LineString) {
	emap.shapeGeometries[eid] = g
}
