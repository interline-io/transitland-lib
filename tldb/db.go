package tldb

import (
	// Driver
	"errors"
	"net/url"
	"strconv"

	"github.com/interline-io/transitland-lib/internal/log"
	_ "github.com/lib/pq"
)

var bufferSize = 1000

// check for error and panic
// TODO: don't do this. panic is bad.
func check(err error) {
	if err != nil {
		log.Debug("Error: %s", err)
		panic(err)
	}
}

func getFvids(dburl string) ([]int, string, error) {
	fvids := []int{}
	u, err := url.Parse(dburl)
	if err != nil {
		return nil, "", err
	}
	vars := u.Query()
	if a, ok := vars["fvid"]; ok {
		for _, v := range a {
			fvid, err := strconv.Atoi(v)
			if err != nil {
				return nil, "", errors.New("invalid feed version id")
			}
			fvids = append(fvids, fvid)
		}
	}
	delete(vars, "fvid")
	u.RawQuery = vars.Encode()
	return fvids, u.String(), nil
}
