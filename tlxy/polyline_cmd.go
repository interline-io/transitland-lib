package tlxy

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/spf13/pflag"
	"github.com/twpayne/go-geom/encoding/geojson"
	"github.com/twpayne/go-shapefile"
)

type PolylinesCommand struct {
	InputType  string
	Input      string
	OutputPath string
	IDKey      string
	ExtraKeys  []string
}

func (cmd *PolylinesCommand) HelpDesc() (string, string) {
	a := "Converts input geometry file to polylines"
	b := ""
	return a, b
}

func (cmd *PolylinesCommand) HelpArgs() string {
	return "[flags] <type> <input> <output>"
}

func (cmd *PolylinesCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringSliceVarP(&cmd.ExtraKeys, "key", "k", nil, "Include these keys in output")
	fl.StringVarP(&cmd.IDKey, "idkey", "i", "", "ID key")
}

func (c *PolylinesCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 3 {
		return fmt.Errorf("requires input type, input file, and output file")
	}
	c.InputType = fl.Arg(0)
	c.Input = fl.Arg(1)
	c.OutputPath = fl.Arg(2)
	return nil
}

func (c *PolylinesCommand) Run(ctx context.Context) error {
	// Download if necessary
	// TODO: replace with store
	if strings.HasPrefix(c.Input, "http") {
		tmpf, err := os.CreateTemp("", "")
		if err != nil {
			return err
		}
		tmpf.Close()
		tname := tmpf.Name()
		defer os.Remove(tname)
		if err := c.downloadFile(c.Input, tname); err != nil {
			return err
		}
		c.Input = tname
	}

	log.Info().Msgf("Reading %s from %s, output: %s\n", c.InputType, c.Input, c.OutputPath)
	var err error
	switch c.InputType {
	case "geojson":
		err = c.CreateFromGeojson()
	case "zipgeojson":
		err = c.CreateFromZipGeojson()
	case "shapefile":
		err = c.CreateFromShapefile()
	default:
		return fmt.Errorf("unknown format: %s", c.InputType)
	}
	return err
}

func (c *PolylinesCommand) CreateFromGeojson() error {
	fn := c.Input
	outfn := c.OutputPath
	idKey := c.IDKey
	keys := c.ExtraKeys
	w, _ := os.Create(outfn)
	defer w.Close()
	r, err := os.Open(fn)
	if err != nil {
		return err
	}
	fcData, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	fc := geojson.FeatureCollection{}
	if err := fc.UnmarshalJSON(fcData); err != nil {
		return err
	}
	if err := GeojsonToPolylines(fc, w, idKey, keys, 6); err != nil {
		return err
	}
	return nil
}

func (c *PolylinesCommand) CreateFromShapefile() error {
	fn := c.Input
	outfn := c.OutputPath
	idKey := c.IDKey
	keys := c.ExtraKeys
	r, err := shapefile.ReadZipFile(fn, nil)
	if err != nil {
		return err
	}
	var features []*geojson.Feature
	for i := 0; i < r.NumRecords(); i++ {
		rec, recGeom := r.Record(i)
		features = append(features, &geojson.Feature{
			Properties: rec,
			Geometry:   recGeom,
		})
	}

	w, _ := os.Create(outfn)
	defer w.Close()
	fc := geojson.FeatureCollection{
		Features: features,
	}
	return GeojsonToPolylines(fc, w, idKey, keys, 6)
}

func (c *PolylinesCommand) CreateFromZipGeojson() error {
	fn := c.Input
	outfn := c.OutputPath
	idKey := c.IDKey
	keys := c.ExtraKeys

	w, _ := os.Create(outfn)
	defer w.Close()
	zf, err := zip.OpenReader(fn)
	if err != nil {
		return err
	}
	for _, f := range zf.File {
		if !(strings.HasSuffix(f.Name, ".json") || strings.HasSuffix(f.Name, ".geojson")) {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		fcData, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		fc := geojson.FeatureCollection{}
		if err := fc.UnmarshalJSON(fcData); err != nil {
			return err
		}
		if err := GeojsonToPolylines(fc, w, idKey, keys, 6); err != nil {
			return err
		}
	}
	return nil
}

func (c *PolylinesCommand) downloadFile(url string, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
