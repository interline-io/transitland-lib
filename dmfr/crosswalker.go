package dmfr

import (
	"github.com/interline-io/gotransit/internal/log"
)

// AddCrosswalkIDs TODO
func AddCrosswalkIDs(baseRegistry *Registry, comparisonRegistries map[string]*Registry) *Registry {
	// Pass 1: by crosswalk IDs

	// Pass 2: by URL + spec type
	for _, baseFeed := range baseRegistry.Feeds {
		if comparisonRegistryID, comparisonFeed := findMatchingFeed(baseFeed.Spec, baseFeed.URL, comparisonRegistries); comparisonFeed != nil {
			log.Trace("baseFeed: %#v", baseFeed)
			baseFeed.OtherIDs[comparisonRegistryID] = comparisonFeed.ID
		}
	}
	return baseRegistry
	// Pass 3 (optional): by domain
}

func findMatchingFeed(feedSpec string, feedURL string, comparisonRegistries map[string]*Registry) (string, *Feed) {
	for comparisonRegistryID, comparisonRegistry := range comparisonRegistries {
		log.Trace("test: %#v", comparisonRegistry)
		for _, comparisonFeed := range comparisonRegistry.Feeds {
			if feedURL == comparisonFeed.URL && feedSpec == comparisonFeed.Spec {
				return comparisonRegistryID, &comparisonFeed
			}
		}
	}
	return "", nil
}
