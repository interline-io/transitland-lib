package dmfr

import (
	"github.com/interline-io/gotransit/internal/log"
)

// AddCrosswalkIDs TODO
func AddCrosswalkIDs(baseRegistry *Registry, comparisonRegistries map[string]*Registry) *Registry {
	// Pass 1: by crosswalk IDs

	// Pass 2: by URL + spec type
	for _, baseFeed := range baseRegistry.Feeds {
		if comparisonRegistryID, comparisonFeed := findMatchingFeed(baseFeed.Spec, baseFeed.URLs.StaticCurrent, comparisonRegistries); comparisonFeed != nil {
			log.Trace("baseFeed: %#v", baseFeed)
			baseFeed.OtherIDs[comparisonRegistryID] = comparisonFeed.FeedID
		}
	}
	return baseRegistry
	// Pass 3 (optional): by domain
}

func findMatchingFeed(feedSpec string, feedURL string, comparisonRegistries map[string]*Registry) (string, *Feed) {
	for comparisonRegistryID, comparisonRegistry := range comparisonRegistries {
		log.Trace("test: %#v", comparisonRegistry)
		for _, comparisonFeed := range comparisonRegistry.Feeds {
			if feedURL == comparisonFeed.URLs.StaticCurrent && feedSpec == comparisonFeed.Spec {
				return comparisonRegistryID, &comparisonFeed
			}
		}
	}
	return "", nil
}
