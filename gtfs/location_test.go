package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
	geom "github.com/twpayne/go-geom"
)

func TestLocation_Errors(t *testing.T) {
	poly, err := geom.NewPolygon(geom.XY).SetCoords([][]geom.Coord{
		{{-122.4, 37.7}, {-122.4, 37.8}, {-122.3, 37.8}, {-122.3, 37.7}, {-122.4, 37.7}},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		location       *Location
		expectedErrors []ExpectError
	}{
		{
			name: "Valid: location with ID and geometry",
			location: &Location{
				LocationID: tt.NewString("loc1"),
				Geometry:   tt.NewGeometry(poly),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: location with optional fields",
			location: &Location{
				LocationID: tt.NewString("loc2"),
				StopName:   tt.NewString("Downtown Zone"),
				StopDesc:   tt.NewString("Zone for downtown flexible service"),
				ZoneID:     tt.NewString("zone1"),
				StopURL:    tt.NewUrl("https://example.com/zone1"),
				Geometry:   tt.NewGeometry(poly),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: missing location_id",
			location: &Location{
				Geometry: tt.NewGeometry(poly),
			},
			expectedErrors: ParseExpectErrors("RequiredFieldError:location_id"),
		},
		{
			name: "Invalid: missing geometry",
			location: &Location{
				LocationID: tt.NewString("loc3"),
			},
			expectedErrors: ParseExpectErrors("ConditionallyRequiredFieldError:geometry"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.location)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

func TestLocation_EntityMethods(t *testing.T) {
	loc := Location{
		LocationID: tt.NewString("test_loc"),
	}

	if key := loc.EntityKey(); key != "test_loc" {
		t.Errorf("EntityKey() = %q, want %q", key, "test_loc")
	}

	if filename := loc.Filename(); filename != "locations.geojson" {
		t.Errorf("Filename() = %q, want %q", filename, "locations.geojson")
	}

	if table := loc.TableName(); table != "gtfs_locations" {
		t.Errorf("TableName() = %q, want %q", table, "gtfs_locations")
	}

	if entityID := loc.EntityID(); entityID != "test_loc" {
		t.Errorf("EntityID() = %q, want %q", entityID, "test_loc")
	}
}
