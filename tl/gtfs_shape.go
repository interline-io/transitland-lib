package tl

// Shape shapes.txt
type Shape struct {
	ShapeID   string `csv:",required"`
	Geometry  LineString
	Generated bool
	BaseEntity
}

// Filename shapes.txt
func (ent *Shape) Filename() string {
	return "shapes.txt"
}

// TableName gtfs_shapes
func (ent *Shape) TableName() string {
	return "gtfs_shapes"
}

// EntityID returns the ID or ShapeID.
func (ent *Shape) EntityID() string {
	return entID(ent.ID, ent.ShapeID)
}

// EntityKey returns the GTFS identifier.
func (ent *Shape) EntityKey() string {
	return ent.ShapeID
}
