package dmfr

import "time"

// Feed listed in a parsed DMFR file
type Feed struct {
	ID                    int
	FeedNamespaceID       string
	Spec                  string
	URL                   string
	URLs                  map[string]string
	AssociatedFeeds       []string
	Languages             []string
	License               map[string]string
	Authorization         map[string]string
	OtherIDs              map[string]string `json:"other_ids"`
	IDCrosswalk           map[string]string `json:"id_crosswalk"`
	LastFetchedAt         time.Time
	LastSuccessfulFetchAt time.Time
	LastFetchError        string
}
