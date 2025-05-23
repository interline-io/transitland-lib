package cmds

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-shapefile"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/schema/ne"
	postgressMigrations "github.com/interline-io/transitland-lib/schema/postgres"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tldb"
	postgressAdapter "github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tt"

	"github.com/spf13/pflag"
)

type DBMigrateCommand struct {
	DBURL      string
	Subcommand string
	Adapter    tldb.Adapter
}

func (cmd *DBMigrateCommand) HelpDesc() (string, string) {
	return "Perform database migrations and load Natural Earth geographies", ""
}

func (cmd *DBMigrateCommand) HelpArgs() string {
	return "[flags] <subcommand>"
}

func (cmd *DBMigrateCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
}

// Parse command line options.
func (cmd *DBMigrateCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() == 0 {
		return errors.New("subcommand required")
	}
	cmd.Subcommand = fl.Arg(0)
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("TL_DATABASE_URL")
	}
	return nil
}

// Run this command.
func (cmd *DBMigrateCommand) Run(ctx context.Context) error {
	atx := postgressAdapter.PostgresAdapter{DBURL: cmd.DBURL}
	db, err := atx.OpenDB()
	if err != nil {
		return err
	}
	defer atx.Close()
	cmd.Adapter = postgressAdapter.NewPostgresAdapterFromDBX(db)

	_ = postgressMigrations.EmbeddedMigrations
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return err
	}
	_ = driver
	source, err := iofs.New(postgressMigrations.EmbeddedMigrations, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	m.Log = &migrationLogger{log: log.Logger.With().Logger()}
	if err != nil {
		return err
	}
	switch cmd.Subcommand {
	case "up":
		log.Info().Msg("Running migrations...")
		err := m.Up()
		if err == nil {
			log.Info().Msg("Migrations complete")
			return nil
		}
		if err == migrate.ErrNoChange {
			log.Info().Msg("No migrations to run")
			return nil
		}
		return err
	case "natural-earth":
		return cmd.neLoad(ctx)
	case "down":
		return errors.New("unsupported command")
	}
	return fmt.Errorf("unknown subcommand: %s", cmd.Subcommand)
}

func (cmd *DBMigrateCommand) neLoad(ctx context.Context) error {
	var err error
	err = fs.WalkDir(ne.EmbeddedNaturalEarthData, ".", func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".zip") {
			return nil
		}
		neZipData, err := ne.EmbeddedNaturalEarthData.ReadFile(path)
		if err != nil {
			return err
		}
		switch path {
		case "ne_10m_admin_1_states_provinces.zip":
			return cmd.neProcess(ctx, neZipData, cmd.neLoadAdmins)
		case "ne_10m_populated_places_simple.zip":
			return cmd.neProcess(ctx, neZipData, cmd.neLoadPlaces)
		}
		return nil
	})
	return err
}

func (cmd *DBMigrateCommand) neProcess(ctx context.Context, neZipFile []byte, cb shpFileHandler) error {
	ret := []neShape{}
	zipReader, err := zip.NewReader(bytes.NewReader(neZipFile), int64(len(neZipFile)))
	if err != nil {
		return err
	}
	scanner, err := shapefile.NewScannerFromZipReader(zipReader, &shapefile.ReadShapefileOptions{
		DBF: &shapefile.ReadDBFOptions{
			SkipBrokenFields: true,
		},
	})
	if err != nil {
		return err
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
	return cb(ctx, cmd.Adapter, ret)
}

type shpFileHandler = func(context.Context, tldb.Adapter, []neShape) error

type neShape struct {
	geometry geom.T
	attrs    map[string]string
}

//////////////////

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

func (cmd *DBMigrateCommand) neLoadAdmins(ctx context.Context, atx tldb.Adapter, shapes []neShape) error {
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

//////////////////

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

func (cmd *DBMigrateCommand) neLoadPlaces(ctx context.Context, atx tldb.Adapter, shapes []neShape) error {
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

/////////

type migrationLogger struct {
	log zerolog.Logger
}

func (l *migrationLogger) Printf(format string, v ...interface{}) {
	format = strings.TrimSuffix(format, "\n")
	l.log.Info().Msgf(format, v...)
}

func (l *migrationLogger) Verbose() bool {
	return false
}
