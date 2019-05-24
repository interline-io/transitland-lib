package dmfr

import "fmt"

// AddCrosswalkIDs TODO
func AddCrosswalkIDs(baseRegistry *Registry, comparisonRegistries map[string]*Registry) *Registry {
	// Pass 1: by crosswalk IDs

	// Pass 2: by URL + spec type
	for _, baseFeed := range baseRegistry.Feeds {
		if comparisonRegistryID, comparisonFeed := findMatchingFeed(baseFeed.Spec, baseFeed.URL, comparisonRegistries); comparisonFeed != nil {
			fmt.Printf("baseFeed: %#v\n", baseFeed)
			baseFeed.IDCrosswalk[comparisonRegistryID] = comparisonFeed.ID
		}
	}
	return baseRegistry
	// Pass 3 (optional): by domain
}

func findMatchingFeed(feedSpec string, feedURL string, comparisonRegistries map[string]*Registry) (string, *Feed) {
	for comparisonRegistryID, comparisonRegistry := range comparisonRegistries {
		fmt.Printf("TEST: %#v\n", comparisonRegistry)
		for _, comparisonFeed := range comparisonRegistry.Feeds {
			if feedURL == comparisonFeed.URL && feedSpec == comparisonFeed.Spec {
				return comparisonRegistryID, &comparisonFeed
			}
		}
	}
	return "", nil
}
