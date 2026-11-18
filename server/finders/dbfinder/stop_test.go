package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
)

// Stops that share an id, as a validation's stops do, each get the admin area
// holding their own point.
func TestFinder_StopPlacesByStopID(t *testing.T) {
	f := NewFinder(testutil.MustOpenTestDB(t))
	params := []model.StopPlaceParam{
		{ID: 0, Point: tlxy.Point{Lon: -122.431297, Lat: 37.773972}},
		{ID: 0, Point: tlxy.Point{Lon: -74.132285, Lat: 40.625665}},
		{ID: 0, Point: tlxy.Point{Lon: -94.237, Lat: 26.874}},
	}
	places, errs := f.StopPlacesByStopID(context.Background(), params)
	assert.Nil(t, errs)
	var got []string
	for _, p := range places {
		if p == nil {
			got = append(got, "")
			continue
		}
		got = append(got, *p.Adm1Iso)
	}
	assert.Equal(t, []string{"US-CA", "US-NY", ""}, got)
}
