package gtfs

import (
	"strconv"
)

func entID(id int, gtfsid string) string {
	if id > 0 {
		return strconv.Itoa(id)
	}
	return gtfsid
}
