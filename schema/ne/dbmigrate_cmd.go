package ne

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tldb"
	postgresAdapter "github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/spf13/pflag"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-shapefile"
)

// Command loads Natural Earth admin boundaries and populated places into a
// Postgres database.
type Command struct {
	DBURL     string
	Overwrite bool
	Adapter   tldb.Adapter
}

func (cmd *Command) HelpDesc() (string, string) {
	return "Load Natural Earth admin boundaries and populated places into the database", ""
}

func (cmd *Command) AddFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
	fl.BoolVar(&cmd.Overwrite, "overwrite", false, "Reload even if data is already present (truncates first); default skips when present")
}

func (cmd *Command) Parse(args []string) error {
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	return nil
}

func (cmd *Command) Run(ctx context.Context) error {
	atx := postgresAdapter.PostgresAdapter{DBURL: cmd.DBURL}
	db, err := atx.OpenDB()
	if err != nil {
		return err
	}
	defer atx.Close()
	cmd.Adapter = postgresAdapter.NewPostgresAdapterFromDBX(db)

	// The loader appends rows unconditionally (these tables have no natural unique key),
	// so guard against duplicating an existing load: skip when data is already present
	// unless --overwrite, which truncates both tables and reloads.
	var count int
	if err := cmd.Adapter.Get(ctx, &count, "SELECT count(*) FROM ne_10m_populated_places"); err != nil {
		return err
	}
	if count > 0 && !cmd.Overwrite {
		log.Info().Msgf("Natural Earth data already present (%d populated places); skipping (use --overwrite to reload)", count)
		return nil
	}
	if cmd.Overwrite {
		if _, err := cmd.Adapter.DBX().ExecContext(ctx, "TRUNCATE ne_10m_admin_1_states_provinces, ne_10m_populated_places"); err != nil {
			return err
		}
	}

	return fs.WalkDir(EmbeddedNaturalEarthData, ".", func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".zip") {
			return nil
		}
		var load shapeHandler
		switch path {
		case adminsZip:
			load = cmd.loadAdmins
		case placesZip:
			load = cmd.loadPlaces
		default:
			return nil
		}
		shapes, err := readEmbeddedShapes(path)
		if err != nil {
			return err
		}
		return load(ctx, cmd.Adapter, shapes)
	})
}

type shapeHandler = func(context.Context, tldb.Adapter, []neShape) error

type neShape struct {
	geometry geom.T
	attrs    map[string]string
}

// readEmbeddedShapes reads the shapes and their attributes from one of the
// embedded shapefile zips.
func readEmbeddedShapes(name string) ([]neShape, error) {
	neZipFile, err := EmbeddedNaturalEarthData.ReadFile(name)
	if err != nil {
		return nil, err
	}
	ret := []neShape{}
	zipReader, err := zip.NewReader(bytes.NewReader(neZipFile), int64(len(neZipFile)))
	if err != nil {
		return nil, err
	}
	scanner, err := shapefile.NewScannerFromZipReader(zipReader, &shapefile.ReadShapefileOptions{
		DBF: &shapefile.ReadDBFOptions{
			SkipBrokenFields: true,
		},
	})
	if err != nil {
		return nil, err
	}
	shapeFields := scanner.DBFFieldDescriptors()
	for scanner.Next() {
		shpGeom, _, shpRec := scanner.Scan()
		if shpGeom == nil {
			continue
		}
		t := neShape{}
		t.attrs = map[string]string{}
		for i, fieldDesc := range shapeFields {
			fieldName := fieldDesc.Name
			fieldValue := shpRec[i]
			t.attrs[fieldName] = fmt.Sprintf("%v", fieldValue)
		}
		t.geometry = shpGeom.Geom
		ret = append(ret, t)
	}
	// The scanner stops with each of its files at EOF.
	if err := scanner.Error(); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return ret, nil
}

type neAdmin struct {
	Name     tt.String   `db:"name"`
	Admin    tt.String   `db:"admin"`
	IsoName  tt.String   `db:"iso_3166_2"`
	IsoA2    tt.String   `db:"iso_a2"`
	Geometry tt.Geometry `db:"geometry"`
}

func (ent *neAdmin) TableName() string {
	return "ne_10m_admin_1_states_provinces"
}

func (cmd *Command) loadAdmins(ctx context.Context, atx tldb.Adapter, shapes []neShape) error {
	var ents []any
	for _, nes := range shapes {
		ent := neAdmin{}
		ent.Geometry = tt.NewGeometry(nes.geometry)
		for k, val := range nes.attrs {
			switch k {
			case "admin":
				ent.Admin.Set(val)
			case "name":
				ent.Name.Set(val)
			case "iso_3166_2":
				ent.IsoName.Set(val)
			case "iso_a2":
				ent.IsoA2.Set(val)
			}
		}
		ents = append(ents, &ent)
	}
	log.Info().Msgf("Inserting %d admin boundaries", len(ents))
	if _, err := cmd.Adapter.MultiInsert(ctx, ents); err != nil {
		return err
	}
	return nil
}

type nePlace struct {
	Name     tt.String   `db:"name"`
	Adm0Name tt.String   `db:"adm0name"`
	Adm1Name tt.String   `db:"adm1name"`
	IsoA2    tt.String   `db:"iso_a2"`
	Geometry tt.Geometry `db:"geometry"`
}

func (ent *nePlace) TableName() string {
	return "ne_10m_populated_places"
}

func (cmd *Command) loadPlaces(ctx context.Context, atx tldb.Adapter, shapes []neShape) error {
	var ents []any
	for _, nes := range shapes {
		ent := nePlace{}
		ent.Geometry = tt.NewGeometry(nes.geometry)
		for k, val := range nes.attrs {
			switch k {
			case "name":
				ent.Name.Set(val)
			case "adm0name":
				ent.Adm0Name.Set(val)
			case "adm1name":
				ent.Adm1Name.Set(val)
			case "iso_a2":
				ent.IsoA2.Set(val)
			}
		}
		ents = append(ents, &ent)
	}
	log.Info().Msgf("Inserting %d populated places", len(ents))
	if _, err := cmd.Adapter.MultiInsert(ctx, ents); err != nil {
		return err
	}
	return nil
}
