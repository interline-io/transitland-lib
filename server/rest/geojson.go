package rest

import (
	"context"
	"errors"

	"github.com/interline-io/log"
)

// processGeoJSON reshapes a GraphQL entity response into a GeoJSON FeatureCollection in place.
func processGeoJSON(ctx context.Context, ent apiHandler, response map[string]interface{}) error {
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
			log.For(ctx).Info().Msg("feature not map[string]any, skipping")
			continue
		}
		geometry := f["geometry"]
		if geometry == nil {
			geometry = map[string]any{
				"type":        "Polygon",
				"coordinates": []float64{},
			}
		}
		delete(f, "geometry")
		properties := hw{}
		for k, v := range f {
			properties[k] = v
		}
		features = append(features, hw{
			"type":       "Feature",
			"properties": properties,
			"geometry":   geometry,
		})
	}
	delete(response, fkey)
	response["type"] = "FeatureCollection"
	response["features"] = features
	return nil
}
