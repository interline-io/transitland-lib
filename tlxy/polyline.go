package tlxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/interline-io/log"
	"github.com/pkg/errors"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-polyline"
)

func DecodePolylineString(p string) ([]Point, error) {
	return DecodePolyline([]byte(p))
}

func DecodePolyline(p []byte) ([]Point, error) {
	// Decode directly into Points with go-polyline's integer primitive, avoiding the
	// [][]float64 (and a 2-float64 slice per point) that polyline.DecodeCoords builds.
	var ret []Point
	var lat, lon int
	for len(p) > 0 {
		dlat, rest, err := polyline.DecodeInt(p)
		if err != nil {
			return ret, err
		}
		dlon, rest, err := polyline.DecodeInt(rest)
		if err != nil {
			return ret, err
		}
		lat += dlat
		lon += dlon
		ret = append(ret, Point{Lat: float64(lat) / 1e5, Lon: float64(lon) / 1e5})
		p = rest
	}
	return ret, nil
}

func EncodePolyline(coords []Point) []byte {
	// Encode directly with go-polyline's integer primitive (scale 1e5, matching its
	// default Codec) instead of building a [][]float64 with a throwaway 2-float64
	// slice per point. That intermediate is a heavy transient allocation when caching
	// millions of shape points and shows up as a multi-GB heap spike during import.
	buf := make([]byte, 0, len(coords)*4)
	var lastLat, lastLon int
	for _, c := range coords {
		lat := polylineRound(1e5 * c.Lat)
		lon := polylineRound(1e5 * c.Lon)
		buf = polyline.EncodeInt(buf, lat-lastLat)
		buf = polyline.EncodeInt(buf, lon-lastLon)
		lastLat, lastLon = lat, lon
	}
	return buf
}

// polylineRound matches go-polyline's rounding (round half away from zero).
func polylineRound(x float64) int {
	if x < 0 {
		return int(-math.Floor(-x + 0.5))
	}
	return int(math.Floor(x + 0.5))
}

func PolylinesToGeojson(r io.Reader) (geojson.FeatureCollection, error) {
	fc := geojson.FeatureCollection{}
	var features []*geojson.Feature

	// Scan through rows in the input data
	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, 1024*1024)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		// Split TSV row
		sp := bytes.Split(scanner.Bytes(), []byte("\t"))
		if len(sp) < 3 {
			continue
		}

		// First column: Get the feature ID
		featId := strings.TrimSpace(string(sp[0]))

		// Second column: Decode the properties
		var props map[string]any
		if spi := bytes.TrimSpace(sp[1]); len(spi) > 0 {
			if err := json.Unmarshal(spi, &props); err != nil {
				return fc, errors.Wrap(err, "failed to decode properties")
			}
		}

		// Third column: Get the scale and initialize the codec
		polylineScale := float64(6)
		if spi := strings.TrimSpace(string(sp[2])); len(spi) > 0 {
			a, err := strconv.ParseInt(spi, 10, 64)
			if err != nil {
				return fc, errors.Wrap(err, "failed to decode scale")
			}
			polylineScale = float64(a)
		}

		// Fourth and following columns: Decode the coordinates
		codec := polyline.Codec{Dim: 2, Scale: math.Pow(10, polylineScale)}
		g := geom.NewPolygon(geom.XY)
		for i := 3; i < len(sp); i++ {
			spi := sp[i]
			if len(spi) == 0 {
				continue
			}
			var dec []float64
			dec, _, err := codec.DecodeFlatCoords(dec, spi)
			if err != nil {
				return fc, errors.Wrap(err, "failed to decode coordinates")
			}
			g.Push(geom.NewLinearRingFlat(geom.XY, dec))
		}

		// Add the feature
		features = append(features, &geojson.Feature{
			ID:         featId,
			Properties: props,
			Geometry:   g,
		})
	}

	// Return the feature collection
	fc.Features = features
	return fc, nil
}

func GeojsonToPolylines(fc geojson.FeatureCollection, w io.Writer, idKey string, keys []string, polylineScalePow int) error {
	codec := polyline.Codec{Dim: 2, Scale: math.Pow(10, float64(polylineScalePow))}
	for i, feature := range fc.Features {
		if i == 0 {
			var recKeys []string
			for k := range feature.Properties {
				recKeys = append(recKeys, k)
			}
			log.Info().Msgf("first record has keys: %v\n", recKeys)
			log.Info().Msgf("selecting keys: %v\n", keys)
			log.Info().Msgf("first record has geom: %T\n", feature.Geometry)
		}
		log.Info().Msgf("processing record: %d\n", i)

		// Split into polygons
		var polys []*geom.Polygon
		if v, ok := feature.Geometry.(*geom.Polygon); ok {
			polys = append(polys, v)
		} else if v, ok := feature.Geometry.(*geom.MultiPolygon); ok {
			for i := 0; i < v.NumPolygons(); i++ {
				polys = append(polys, v.Polygon(i))
			}
		}

		// Process each polygon into a row
		for _, g := range polys {
			// Get the feature ID
			featId := feature.ID
			if a, ok := feature.Properties[idKey].(string); idKey != "" && ok {
				featId = a
			}

			// Prepare the properties
			var jj []byte
			if len(keys) > 0 {
				props := map[string]any{}
				for _, key := range keys {
					if key == "" {
						continue
					}
					props[key] = feature.Properties[key]
				}
				if len(props) > 0 {
					jj, _ = json.Marshal(props)
				}
			}

			// Prepare the row
			row := []string{
				featId,
				string(jj),
				strconv.Itoa(polylineScalePow),
			}

			// Encode coordinates
			for p := 0; p < g.NumLinearRings(); p++ {
				pring := g.LinearRing(p)
				var pc [][]float64
				for _, p2 := range pring.Coords() {
					pc = append(pc, []float64{p2[0], p2[1]})
				}
				var buf []byte
				buf = codec.EncodeCoords(buf, pc)
				row = append(row, string(buf))
			}

			// Write as TSV
			w.Write([]byte(strings.Join(row, "\t") + "\n"))
		}
	}
	return nil
}
