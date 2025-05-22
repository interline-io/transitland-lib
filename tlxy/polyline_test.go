package tlxy

import (
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/stretchr/testify/assert"
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
