package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/interline-io/transitland-lib/tldb"
	postgresAdapter "github.com/interline-io/transitland-lib/tldb/postgres"

	"github.com/spf13/pflag"
)

// Command runs schema migrations against a Postgres database using the
// migrations embedded in this package. Lives next to the embedded migrations
// so consumers that don't need to run dbmigrate don't transitively pull them
// in by importing transitland-lib/cmds.
type Command struct {
	DBURL      string
	Subcommand string
	Adapter    tldb.Adapter
}

func (cmd *Command) HelpDesc() (string, string) {
	return "Perform database migrations", ""
}

func (cmd *Command) HelpArgs() string {
	return "[flags] <subcommand>"
}

func (cmd *Command) AddFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.DBURL, "dburl", "", "Database URL (default: $TL_DATABASE_URL)")
}

func (cmd *Command) Parse(args []string) error {
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

func (cmd *Command) Run(ctx context.Context) error {
	atx := postgresAdapter.PostgresAdapter{DBURL: cmd.DBURL}
	db, err := atx.OpenDB()
	if err != nil {
		return err
	}
	defer atx.Close()
	cmd.Adapter = postgresAdapter.NewPostgresAdapterFromDBX(db)

	driver, err := migratepg.WithInstance(db.DB, &migratepg.Config{})
	if err != nil {
		return err
	}
	source, err := iofs.New(EmbeddedMigrations, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return err
	}
	m.Log = &migrationLogger{log: log.Logger.With().Logger()}

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
