package filters

import (
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/twpayne/go-geom/xy"
)

type SimplifyShapeFilter struct {
	SimplifyValue float64
}

func (e *SimplifyShapeFilter) Filter(ent tt.Entity, _ *tt.EntityMap) error {
	v, ok := ent.(*service.ShapeLine)
	if !ok {
		return nil
	}
	sv := e.SimplifyValue / 1e6
	pnts := v.Geometry.FlatCoords()
	stride := v.Geometry.Stride()
	ii := xy.SimplifyFlatCoords(pnts, sv, stride)
	for i, j := range ii {
		if i == j*stride {
			continue
		}
		pnts[i*stride], pnts[i*stride+1] = pnts[j*stride], pnts[j*stride+1]
	}
	pnts = pnts[:len(ii)*stride]
	v.Geometry = tt.NewLineStringFromFlatCoords(pnts)
	return nil
}
