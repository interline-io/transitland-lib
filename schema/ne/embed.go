package ne

import "embed"

//go:embed *.zip
var EmbeddedNaturalEarthData embed.FS

// The embedded shapefiles: first-level admin areas, and populated places.
const (
	adminsZip = "ne_10m_admin_1_states_provinces.zip"
	placesZip = "ne_10m_populated_places_simple.zip"
)
