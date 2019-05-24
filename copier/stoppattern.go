package copier

import (
	"strings"

	"github.com/interline-io/gotransit"
)

func stopPatternKey(stoptimes []gotransit.StopTime) string {
	key := make([]string, len(stoptimes))
	for i := 0; i < len(stoptimes); i++ {
		key[i] = stoptimes[i].StopID
	}
	return strings.Join(key, string(byte(0)))
}
