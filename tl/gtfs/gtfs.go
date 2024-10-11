package gtfs

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tl/tt"
)

type BaseEntity = tt.BaseEntity

// For StopTimes
type FeedVersionEntity = tt.FeedVersionEntity
type MinEntity = tt.MinEntity
type ErrorEntity = tt.ErrorEntity
type ExtraEntity = tt.ExtraEntity

type EntityMap = tt.EntityMap

func entID(id int, gtfsid string) string {
	if id > 0 {
		return strconv.Itoa(id)
	}
	return gtfsid
}
