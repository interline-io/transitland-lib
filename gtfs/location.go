package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Location represents a GeoJSON feature from locations.geojson
// Locations define zones using GeoJSON Polygon or MultiPolygon geometries
// where riders can request pickups or drop-offs for flexible services
type Location struct {
	// ID is the feature ID from the GeoJSON, shares namespace with stop_id
	LocationID tt.String `json:"id" csv:",required" standardized_sort:"1"`
	// Properties
	StopName tt.String   `json:"stop_name"`
	StopDesc tt.String   `json:"stop_desc"`
	ZoneID   tt.String   `json:"zone_id"`
	StopURL  tt.Url      `json:"stop_url"`
	Geometry tt.Geometry `json:"geometry"`
	tt.BaseEntity
}

func (ent *Location) EntityKey() string {
	return ent.LocationID.Val
}

func (ent *Location) EntityID() string {
	return entID(ent.ID, ent.LocationID.Val)
}

func (ent *Location) Filename() string {
	return "locations.geojson"
}

func (ent *Location) TableName() string {
	return "gtfs_locations"
}

// ConditionalErrors for this Entity.
func (ent *Location) ConditionalErrors() (errs []error) {
	// Geometry must be present
	if !ent.Geometry.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("geometry"))
	}

	return errs
}
