package testdb

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

// Caltrain returns a simple feed inserted into a database.
func Caltrain(atx tldb.Adapter, url string) tl.Feed {
	// Create dummy feed
	tlfeed := tl.Feed{}
	tlfeed.FeedID = url
	tlfeed.URLs.StaticCurrent = url
	tlfeed.ID = MustInsert(atx, &tlfeed)
	return tlfeed
}
