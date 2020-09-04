package copier

import (
	"strings"

	"github.com/interline-io/transitland-lib/tl"
)

func stopPatternKey(stoptimes []tl.StopTime) string {
	key := make([]string, len(stoptimes))
	for i := 0; i < len(stoptimes); i++ {
		key[i] = stoptimes[i].StopID
	}
	return strings.Join(key, string(byte(0)))
}
