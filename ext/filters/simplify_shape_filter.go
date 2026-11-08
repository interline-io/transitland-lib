package filters

import (
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/twpayne/go-geom/xy"
)

type SimplifyShapeFilter struct {
	SimplifyValue float64
}

func (e *SimplifyShapeFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	v, ok := ent.(*service.ShapeLine)
	if !ok {
		return nil
	}
	sv := e.SimplifyValue / 1e6
	pnts := v.Geometry.FlatCoords()
	stride := v.Geometry.Stride()
	ii := xy.SimplifyFlatCoords(pnts, sv, stride)
	// Move each kept point into place whole, so it keeps its own measure.
	for i, j := range ii {
		copy(pnts[i*stride:(i+1)*stride], pnts[j*stride:(j+1)*stride])
	}
	pnts = pnts[:len(ii)*stride]
	v.Geometry = tt.NewLineStringFromFlatCoords(pnts)
	return nil
}
