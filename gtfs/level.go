package gtfs

import "github.com/interline-io/transitland-lib/tl/tt"

// Level levels.txt
type Level struct {
	LevelID    string  `csv:",required"`
	LevelIndex float64 `csv:",required"`
	LevelName  string  `csv:"level_name"`
	tt.BaseEntity
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
