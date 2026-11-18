// Package ne holds Natural Earth's admin areas and populated places: the
// shapefiles, an index of them in memory, and a command that loads them into
// Postgres.
package ne

import "embed"

// EmbeddedNaturalEarthData holds the Natural Earth shapefile zips.
//
//go:embed *.zip
var EmbeddedNaturalEarthData embed.FS

// The embedded shapefiles: first-level admin areas, and populated places.
const (
	adminsZip = "ne_10m_admin_1_states_provinces.zip"
	placesZip = "ne_10m_populated_places_simple.zip"
)
