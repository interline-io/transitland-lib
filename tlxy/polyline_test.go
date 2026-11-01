package tlxy

import (
	"math"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/stretchr/testify/assert"
	"github.com/twpayne/go-polyline"
)

func TestDecodePolyline(t *testing.T) {
	check := "yfttIf{jR?B@BBDD?@ANDH@XHrAj@t@Z"
	expect := []Point{
		{Lon: -3.1738, Lat: 55.97821},
		{Lon: -3.17382, Lat: 55.97821},
		{Lon: -3.17384, Lat: 55.978199},
		{Lon: -3.17387, Lat: 55.978179},
		{Lon: -3.17387, Lat: 55.978149},
		{Lon: -3.17386, Lat: 55.978139},
		{Lon: -3.17389, Lat: 55.978059},
		{Lon: -3.17390, Lat: 55.978009},
		{Lon: -3.17395, Lat: 55.977879},
		{Lon: -3.17417, Lat: 55.977459},
		{Lon: -3.17431, Lat: 55.977189},
	}
	p, err := DecodePolylineString(check)
	if err != nil {
		t.Fatal(err)
	}
	if len(expect) != len(p) {
		t.Fatal("unequal length")
	}
	for i := range expect {
		assert.InDelta(t, expect[i].Lon, p[i].Lon, 0.001)
		assert.InDelta(t, expect[i].Lat, p[i].Lat, 0.001)
	}
}

func TestEncodePolyline(t *testing.T) {
	expect := "yfttIf{jR?B@BBDD?@ANDH@XHrAj@t@Z"
	check := []Point{
		{Lon: -3.1738, Lat: 55.97821},
		{Lon: -3.17382, Lat: 55.97821},
		{Lon: -3.17384, Lat: 55.978199},
		{Lon: -3.17387, Lat: 55.978179},
		{Lon: -3.17387, Lat: 55.978149},
		{Lon: -3.17386, Lat: 55.978139},
		{Lon: -3.17389, Lat: 55.978059},
		{Lon: -3.17390, Lat: 55.978009},
		{Lon: -3.17395, Lat: 55.977879},
		{Lon: -3.17417, Lat: 55.977459},
		{Lon: -3.17431, Lat: 55.977189},
	}
	p := EncodePolyline(check)
	assert.Equal(t, expect, string(p))
}

func TestEncodePolylinePrecision(t *testing.T) {
	check := []Point{
		{Lon: -122.4194155, Lat: 37.7749295},
		{Lon: -122.4183872, Lat: 37.7756012},
		{Lon: -122.2711137, Lat: 37.8043637},
	}
	assert.Equal(t, string(EncodePolyline(check)), string(EncodePolylinePrecision(check, 5)), "precision 5 is the original form")
	for _, precision := range []int{5, 6} {
		scale := math.Pow(10, float64(precision))
		coords, rest, err := polyline.Codec{Dim: 2, Scale: scale}.DecodeCoords(EncodePolylinePrecision(check, precision))
		if err != nil {
			t.Fatal(err)
		}
		assert.Empty(t, rest)
		assert.Len(t, coords, len(check))
		for i, c := range coords {
			// Latitude first, then longitude, each within a unit of the last digit.
			assert.InDelta(t, check[i].Lat, c[0], 1/scale, "precision %d point %d", precision, i)
			assert.InDelta(t, check[i].Lon, c[1], 1/scale, "precision %d point %d", precision, i)
		}
	}
}

func TestPolylinesToGeojson(t *testing.T) {
	fn := testpath.RelPath("testdata/tlxy/tz-example.polyline")
	r, err := os.Open(fn)
	if err != nil {
		t.Fatal(err)
	}
	fc, err := PolylinesToGeojson(r)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 53, len(fc.Features))
	featureCounts := map[string]int{}
	for _, f := range fc.Features {
		featureCounts[f.ID]++
	}
	assert.Equal(t, 3, featureCounts["America/Los_Angeles"])
	assert.Equal(t, 6, featureCounts["America/New_York"])
}
