package tl

// Level levels.txt
type Level struct {
	LevelID    string  `csv:"level_id" required:"true"`
	LevelIndex float64 `csv:"level_index" required:"true"`
	LevelName  string  `csv:"level_name"`
	BaseEntity
}

// EntityID returns the ID or StopID.
func (ent *Level) EntityID() string {
	return entID(ent.ID, ent.LevelID)
}

// EntityKey returns the GTFS identifier.
func (ent *Level) EntityKey() string {
	return ent.LevelID
}

// Filename levels.txt
func (ent *Level) Filename() string {
	return "levels.txt"
}

// TableName ext_pathways_levels
func (ent *Level) TableName() string {
	return "gtfs_levels"
}
