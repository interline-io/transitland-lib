package gtfs

import "github.com/interline-io/transitland-lib/tt"

// Level levels.txt
type Level struct {
	LevelID    tt.String `csv:",required" standardized_sort:"1"`
	LevelIndex tt.Float  `csv:",required"`
	LevelName  tt.String
	tt.BaseEntity
}

// EntityID returns the ID or StopID.
func (ent *Level) EntityID() string {
	return entID(ent.ID, ent.LevelID.Val)
}

// EntityKey returns the GTFS identifier.
func (ent *Level) EntityKey() string {
	return ent.LevelID.Val
}

// Filename levels.txt
func (ent *Level) Filename() string {
	return "levels.txt"
}

// TableName ext_pathways_levels
func (ent *Level) TableName() string {
	return "gtfs_levels"
}
