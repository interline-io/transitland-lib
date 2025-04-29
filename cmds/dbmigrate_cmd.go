package cmds

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/jonas-p/go-shp"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/schema/ne"
	postgressMigrations "github.com/interline-io/transitland-lib/schema/postgres"
	"github.com/interline-io/transitland-lib/tlcli"
	postgressAdapter "github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tt"

	"github.com/spf13/pflag"
)

type DBMigrateCommand struct {
	DBURL      string
	Subcommand string
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
	if err != nil {
		return err
	}
	switch cmd.Subcommand {
	case "up":
		return m.Up()
	case "natural-earth":
		return cmd.neLoad(ctx, db)
	case "down":
		return errors.New("unsupported command")
	}
	return fmt.Errorf("unknown subcommand: %s", cmd.Subcommand)
}

func (cmd *DBMigrateCommand) neLoad(ctx context.Context, db *sqlx.DB) error {
	// Teporarily write out to zip
	var err error
	err = fs.WalkDir(ne.EmbeddedNaturalEarthData, ".", func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".zip") {
			return nil
		}
		fmt.Println("Processing:", path)
		neZipData, err := ne.EmbeddedNaturalEarthData.ReadFile(path)
		if err != nil {
			return err
		}
		switch path {
		case "ne_10m_admin_1_states_provinces.zip":
			return cmd.neProcess(ctx, db, neZipData, cmd.neLoadAdmins)
		case "ne_10m_populated_places_simple.zip":
		}
		return nil
	})
	return err
}

type shpFileHandler = func(context.Context, *sqlx.DB, *shp.Reader) error

func (cmd *DBMigrateCommand) neProcess(ctx context.Context, db *sqlx.DB, neZipFile []byte, cb shpFileHandler) error {
	neTempDir, err := os.MkdirTemp("", "ne-shape")
	if err != nil {
		return err
	}
	defer os.RemoveAll(neTempDir)
	buf := bytes.NewReader(neZipFile)
	zipReader, err := zip.NewReader(buf, buf.Size())
	shpFileName := ""
	for _, zipFile := range zipReader.File {
		log.Infof("copying file %s", zipFile.Name)
		neOutTmp, err := os.Create(filepath.Join(neTempDir, zipFile.Name))
		if err != nil {
			return err
		}
		zipFileReader, err := zipFile.Open()
		if _, err := io.Copy(neOutTmp, zipFileReader); err != nil {
			return err
		}
		if err := neOutTmp.Close(); err != nil {
			return err
		}
		if strings.HasSuffix(neOutTmp.Name(), ".shp") {
			shpFileName = neOutTmp.Name()
		}
	}
	shpSrc, err := shp.Open(shpFileName)
	if err != nil {
		return err
	}
	return cb(ctx, db, shpSrc)
}

type neAdmin struct {
	Name     tt.String   `db:"name"`
	IsoName  tt.String   `db:"iso_3166_2"`
	IsoA2    tt.String   `db:"iso_a2"`
	Geometry tt.Geometry `db:"geometry"`
}

func (cmd *DBMigrateCommand) neLoadAdmins(ctx context.Context, db *sqlx.DB, shpSrc *shp.Reader) error {
	shpFields := shpSrc.Fields()
	defer shpSrc.Close()
	for shpSrc.Next() {
		n, p := shpSrc.Shape()

		fmt.Println(reflect.TypeOf(p).Elem(), p.BBox())

		// print attributes
		ent := neAdmin{}
		for k, f := range shpFields {
			val := shpSrc.ReadAttribute(n, k)
			switch f.String() {
			case "name":
				ent.Name.Set(val)
			case "iso_3166_2":
				ent.IsoName.Set(val)
			case "iso_a2":
				ent.IsoA2.Set(val)
			}
		}
		fmt.Println("ent:", ent)
	}
	return nil
}
