package tlxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-polyline"
)

const polylineScale = 1e6

func DecodePolylineString(p string) ([]Point, error) {
	return DecodePolyline([]byte(p))
}

func DecodePolyline(p []byte) ([]Point, error) {
	coords, _, err := polyline.DecodeCoords(p)
	var ret []Point
	for _, c := range coords {
		ret = append(ret, Point{Lon: c[1], Lat: c[0]})
	}
	return ret, err
}

func EncodePolyline(coords []Point) []byte {
	var g [][]float64
	for _, c := range coords {
		g = append(g, []float64{c.Lat, c.Lon})
	}
	return polyline.EncodeCoords(g)
}

func PolylinesToGeojson(r io.Reader) (geojson.FeatureCollection, error) {
	codec := polyline.Codec{Dim: 2, Scale: polylineScale}
	var features []*geojson.Feature
	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, 1024*1024)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		sp := bytes.Split(scanner.Bytes(), []byte("\t"))
		if len(sp) < 2 {
			continue
		}
		g := geom.NewPolygon(geom.XY)
		tzName := string(sp[0])
		var props map[string]any

		if spi := sp[1]; len(spi) > 0 {
			if err := json.Unmarshal(spi, &props); err != nil {
				panic(err)
			}
		}
		for i := 2; i < len(sp); i++ {
			spi := sp[i]
			if len(spi) == 0 {
				continue
			}
			var dec []float64
			dec, _, err := codec.DecodeFlatCoords(dec, spi)
			if err != nil {
				panic(err)
			}
			g.Push(geom.NewLinearRingFlat(geom.XY, dec))
		}
		features = append(features, &geojson.Feature{
			ID:         tzName,
			Properties: props,
			Geometry:   g,
		})
	}
	return geojson.FeatureCollection{Features: features}, nil
}

func GeojsonToPolylines(fc geojson.FeatureCollection, w io.Writer, idKey string, keys []string) error {
	codec := polyline.Codec{Dim: 2, Scale: polylineScale}
	for i, feature := range fc.Features {
		if i == 0 {
			var recKeys []string
			for k := range feature.Properties {
				recKeys = append(recKeys, k)
			}
			fmt.Printf("first record has keys: %v\n", recKeys)
			fmt.Printf("selecting keys: %v\n", keys)
			fmt.Printf("first record has geom: %T\n", feature.Geometry)
		}
		fmt.Printf("processing record: %d\n", i)
		// jj, _ := json.Marshal(feature.Properties)
		// fmt.Println(string(jj))

		var polys []*geom.Polygon
		if v, ok := feature.Geometry.(*geom.Polygon); ok {
			polys = append(polys, v)
		} else if v, ok := feature.Geometry.(*geom.MultiPolygon); ok {
			for i := 0; i < v.NumPolygons(); i++ {
				polys = append(polys, v.Polygon(i))
			}
		}
		for _, g := range polys {
			tzName := feature.ID
			if a, ok := feature.Properties[idKey].(string); idKey != "" && ok {
				tzName = a
			}
			var jj []byte
			if len(keys) > 0 {
				props := map[string]any{}
				for _, key := range keys {
					props[key] = feature.Properties[key]
				}
				jj, _ = json.Marshal(props)
			}
			row := []string{tzName, string(jj)}
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
			w.Write([]byte(strings.Join(row, "\t") + "\n"))
		}
	}
	return nil
}
