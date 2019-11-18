package dmfr

import (
	"database/sql"

	"github.com/interline-io/gotransit"
)

// FeedState .
type FeedState struct {
	ID                    int
	FeedID                int
	ActiveFeedVersionID   gotransit.OptionalKey
	LastFetchError        string
	LastFetchedAt         gotransit.OptionalTime
	LastSuccessfulFetchAt gotransit.OptionalTime
	LastImportedAt        gotransit.OptionalTime
	Priority              sql.NullInt64
}
