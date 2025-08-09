package dbfinder

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"

	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/interline-io/transitland-mw/dbutil"
	"github.com/jmoiron/sqlx"
)

type adminCacheItem struct {
	Adm0Name string
	Adm1Name string
	Adm0Iso  string
	Adm1Iso  string
	Geometry *geom.Polygon
}

type adminCache struct {
	index *tlxy.PolygonIndex
}

func newAdminCache(ctx context.Context, dbx sqlx.Ext) (*adminCache, error) {
	ac := &adminCache{}
	if err := ac.loadAdmins(ctx, dbx); err != nil {
		return nil, err
	}
	return ac, nil
}

func (c *adminCache) loadAdmins(ctx context.Context, dbx sqlx.Ext) error {
	var ents []struct {
		Adm0Name tt.String
		Adm1Name tt.String
		Adm0Iso  tt.String
		Adm1Iso  tt.String
		Geometry tt.Geometry
	}
	q := sq.Select(
		"ne.name as adm1_name",
		"ne.admin as adm0_name",
		"iso_a2 as adm0_iso",
		"iso_3166_2 as adm1_iso",
		"ne.geometry",
	).
		From("ne_10m_admin_1_states_provinces ne")
	if err := dbutil.Select(ctx, dbx, q, &ents); err != nil {
		return err
	}

	var fc []*geojson.Feature
	for _, ent := range ents {
		g, ok := ent.Geometry.Val.(*geom.MultiPolygon)
		if !ok {
			continue
		}
		for i := 0; i < g.NumPolygons(); i++ {
			adm := adminCacheItem{
				Adm0Name: ent.Adm0Name.Val,
				Adm1Name: ent.Adm1Name.Val,
				Adm0Iso:  ent.Adm0Iso.Val,
				Adm1Iso:  ent.Adm1Iso.Val,
			}
			fc = append(fc, &geojson.Feature{
				Geometry:   g.Polygon(i),
				ID:         ent.Adm1Iso.Val,
				Properties: map[string]interface{}{"adm": adm},
			})
		}
	}
	idx, err := tlxy.NewPolygonIndex(geojson.FeatureCollection{Features: fc})
	if err != nil {
		return err
	}
	c.index = idx
	return nil
}

func (c *adminCache) Check(pt tlxy.Point) (adminCacheItem, bool) {
	feat, ok := c.index.NearestFeature(pt)
	if ok == 0 {
		return adminCacheItem{}, false
	}
	adm := adminCacheItem{}
	if v, ok := feat.Properties["adm"].(adminCacheItem); ok {
		adm = v
	}
	return adm, true
}
