package filters

import (
	"testing"

	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Simplifying a shape keeps each remaining point's own measure.
func TestSimplifyShapeFilter_KeepsMeasures(t *testing.T) {
	// North along a meridian, then east: the middle point of the northern leg
	// lies on the line, so it goes, and the corner stays.
	sh := &service.ShapeLine{Geometry: tt.NewLineStringFromFlatCoords([]float64{
		-122, 37, 0,
		-122, 37.001, 100,
		-122, 37.002, 200,
		-121.999, 37.002, 300,
	})}
	require.NoError(t, (&SimplifyShapeFilter{SimplifyValue: 5}).Filter(sh, nil))
	assert.Equal(t, []float64{
		-122, 37, 0,
		-122, 37.002, 200,
		-121.999, 37.002, 300,
	}, sh.Geometry.FlatCoords())
}
