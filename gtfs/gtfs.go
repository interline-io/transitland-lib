package gtfs

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tl/tt"
)

type EntityMap = tt.EntityMap

func entID(id int, gtfsid string) string {
	if id > 0 {
		return strconv.Itoa(id)
	}
	return gtfsid
}
