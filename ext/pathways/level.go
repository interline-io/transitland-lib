package pathways

import (
	"github.com/interline-io/gotransit"
)

// Level levels.txt
type Level struct {
	LevelID    string  `csv:"level_id" required:"true"`
	LevelIndex float64 `csv:"level_index" required:"true"`
	LevelName  string  `csv:"level_name"`
	gotransit.BaseEntity
}

// Filename levels.txt
func (ent *Level) Filename() string {
	return "levels.txt"
}

// TableName ext_pathways_levels
func (ent *Level) TableName() string {
	return "ext_pathways_levels"
}
