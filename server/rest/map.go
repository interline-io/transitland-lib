package rest

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"image/png"

	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// CIRCLESIZE .
const CIRCLESIZE = 10

// CIRCLEWIDTH .
const CIRCLEWIDTH = 5

func renderMap(data []byte, width int, height int) ([]byte, error) {
	fc := geojson.FeatureCollection{}
	if err := fc.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	ctx := sm.NewContext()
	ctx.SetSize(width, height)
	ctx.SetTileProvider(sm.NewTileProviderCartoLight())

	// Excuse this enormously ugly block of type checks.
	stops := map[int]bool{}
	for _, feature := range fc.Features {
		if rss, ok := feature.Properties["route_stops"].([]interface{}); ok {
			for _, rs := range rss {
				if a, ok := rs.(hw); ok {
					if b, ok := a["stop"].(hw); ok {
						id := 0
						if v, ok := b["id"].(float64); ok {
							id = int(v)
						}
						if v, ok := b["geometry"].(hw); ok {
							if v2, ok := v["coordinates"].([]interface{}); len(v2) > 1 && ok {
								if p1, ok := v2[0].(float64); ok {
									if p2, ok := v2[1].(float64); ok {
										if _, ok := stops[id]; !ok {
											fc.Features = append(fc.Features, &geojson.Feature{
												Geometry:   geom.NewPointFlat(geom.XY, []float64{p1, p2}),
												Properties: hw{"id": id},
											})
											stops[id] = true
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Draw onto map
	for _, feature := range fc.Features {
		if g, ok := feature.Geometry.(*geom.LineString); ok {
			positions := []s2.LatLng{}
			for _, coord := range g.Coords() {
				positions = append(positions, s2.LatLngFromDegrees(coord.Y(), coord.X()))
			}
			ctx.AddPath(sm.NewPath(positions, color.RGBA{0x1c, 0x96, 0xd6, 0xff}, 4.0)) // #1c96d6
		} else if g, ok := feature.Geometry.(*geom.Point); ok {
			ctx.AddCircle(sm.NewCircle(s2.LatLngFromDegrees(g.Coords().Y(), g.Coords().X()), color.RGBA{0xff, 0x00, 0x00, 0xff}, color.RGBA{0xff, 0x00, 0x00, 0xff}, CIRCLESIZE, CIRCLEWIDTH))
		} else {
			fmt.Println("cant draw geom type")
		}
	}
	img, err := ctx.Render()
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	err = png.Encode(buf, img)
	return buf.Bytes(), nil
}

func processGeoJSON(ent apiHandler, response map[string]interface{}) error {
	fkey := ""
	if v, ok := ent.(hasResponseKey); ok {
		fkey = v.ResponseKey()
	} else {
		return errors.New("geojson not supported")
	}
	entities, ok := response[fkey].([]interface{})
	if !ok {
		return errors.New("invalid graphql response")
	}
	features := []hw{}
	for _, feature := range entities {
		f, ok := feature.(map[string]interface{})
		if !ok {
			fmt.Println("skipping feature")
			continue
		}
		geometry := f["geometry"]
		if geometry == nil {
			fmt.Println("no geometry")
			continue
		}
		properties := hw{}
		for k, v := range f {
			properties[k] = v
		}
		features = append(features, hw{
			"type":       "Feature",
			"properties": properties,
			"geometry":   geometry,
		})
		delete(f, "geometry")
	}
	delete(response, fkey)
	response["type"] = "FeatureCollection"
	response["features"] = features
	return nil
}
