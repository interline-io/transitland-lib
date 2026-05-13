package cmds

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog"

	"github.com/interline-io/log"
	postgressMigrations "github.com/interline-io/transitland-lib/schema/postgres"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tldb"
	postgressAdapter "github.com/interline-io/transitland-lib/tldb/postgres"

	"github.com/spf13/pflag"
)

type DBMigrateCommand struct {
	DBURL      string
	Subcommand string
	Adapter    tldb.Adapter
}

func (cmd *DBMigrateCommand) HelpDesc() (string, string) {
	return "Perform database migrations", ""
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
		return errors.New("natural-earth has moved to its own command; use 'transitland dbmigrate-natural-earth' instead")
	case "down":
		return errors.New("unsupported command")
	}
	return fmt.Errorf("unknown subcommand: %s", cmd.Subcommand)
}

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
