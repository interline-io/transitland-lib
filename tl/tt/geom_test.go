package tt

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeGeojson(t *testing.T) {
	t.Run("point", func(t *testing.T) {
		lon, lat := -122.40798, 37.78458
		pt := Point{}
		data := fmt.Sprintf(`{"type":"Point","coordinates":[%0.5f,%0.5f]}`, lon, lat)
		if err := pt.UnmarshalJSON([]byte(data)); err != nil {
			t.Error(err)
		}
		assert.Equal(t, pt.X(), lon, "lon")
		assert.Equal(t, pt.Y(), lat, "lat")
	})
	t.Run("point as map", func(t *testing.T) {
		lon, lat := -122.40798, 37.78458
		pt := Point{}
		data := map[string]any{"type": "Point", "coordinates": []float64{lon, lat}}
		if err := pt.UnmarshalGQL(data); err != nil {
			t.Error(err)
		}
		assert.Equal(t, pt.X(), lon, "lon")
		assert.Equal(t, pt.Y(), lat, "lat")
	})
}
