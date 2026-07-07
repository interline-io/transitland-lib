package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
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
	return "Perform database migrations", "Subcommands: up (apply pending migrations); check (exit non-zero if the database is dirty or behind this binary's embedded migrations, for use as a deploy gate); reset-dirty (clear a dirty flag left by a failed migration so it can be retried)."
}

func (cmd *Command) HelpArgs() string {
	return "[flags] <up|check|reset-dirty>"
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
	src, err := iofs.New(EmbeddedMigrations, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
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
	case "check":
		return checkMigrations(m, src)
	case "reset-dirty":
		return resetDirty(m, src)
	case "natural-earth":
		return errors.New("natural-earth has moved to its own command; use 'transitland dbmigrate-natural-earth' instead")
	case "down":
		return errors.New("unsupported command")
	}
	return fmt.Errorf("unknown subcommand: %s", cmd.Subcommand)
}

// checkMigrations returns an error (non-zero exit) when the database is dirty
// or has migrations embedded in this binary that are not yet applied. Intended
// as a deploy gate so new code is never rolled out ahead of its schema. A
// database that is ahead of this binary (e.g. an image rollback) passes.
func checkMigrations(m *migrate.Migrate, src source.Driver) error {
	current, dirty, err := currentVersion(m)
	if err != nil {
		return err
	}
	available, err := availableVersions(src)
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("database is dirty at version %d; run 'dbmigrate reset-dirty' then 'dbmigrate up' before deploying", current)
	}
	if pending := pendingVersions(current, available); len(pending) > 0 {
		applied := fmt.Sprintf("%d", current)
		if current < 0 {
			applied = "none"
		}
		return fmt.Errorf("database is behind: %d migration(s) pending (applied through %s, latest available %d)", len(pending), applied, available[len(available)-1])
	}
	log.Info().Msgf("Database schema is up to date (version %d)", current)
	if len(available) > 0 && current > available[len(available)-1] {
		log.Info().Msgf("Database version %d is ahead of this binary's latest embedded migration %d", current, available[len(available)-1])
	}
	return nil
}

// resetDirty clears a dirty flag left by a failed migration by stepping the
// recorded version back to the preceding migration, so 'up' re-runs the failed
// one. This is golang-migrate's Force in the backward (safe) direction; it never
// records an unapplied migration as done. Idempotent: a no-op when not dirty.
func resetDirty(m *migrate.Migrate, src source.Driver) error {
	current, dirty, err := currentVersion(m)
	if err != nil {
		return err
	}
	if current < 0 {
		log.Info().Msg("No migrations applied; nothing to reset")
		return nil
	}
	if !dirty {
		log.Info().Msgf("Database is not dirty (version %d); nothing to reset", current)
		return nil
	}
	// Step back to the version before the dirty one. If it was the first
	// migration, reset to NilVersion so 'up' re-runs from the start.
	target := -1
	if prev, err := src.Prev(uint(current)); err == nil {
		target = int(prev)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	log.Info().Msgf("Database dirty at version %d; resetting recorded version to %d and clearing dirty flag", current, target)
	if err := m.Force(target); err != nil {
		return err
	}
	log.Info().Msgf("Dirty flag cleared. Re-run 'dbmigrate up' to retry migration %d. For non-transactional migrations (e.g. CREATE INDEX CONCURRENTLY) undo any partial effects first.", current)
	return nil
}

// currentVersion reports the applied schema version and dirty flag, returning
// -1 when no migration has been applied.
func currentVersion(m *migrate.Migrate) (int, bool, error) {
	v, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return -1, false, nil
		}
		return 0, false, err
	}
	return int(v), dirty, nil
}

// availableVersions returns the migration versions in the source, ascending.
func availableVersions(src source.Driver) ([]int, error) {
	first, err := src.First()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	versions := []int{int(first)}
	for cur := first; ; {
		next, err := src.Next(cur)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			return nil, err
		}
		versions = append(versions, int(next))
		cur = next
	}
	return versions, nil
}

// pendingVersions returns the available versions newer than current, which is
// -1 when no migration has been applied. available must be ascending.
func pendingVersions(current int, available []int) []int {
	var pending []int
	for _, v := range available {
		if v > current {
			pending = append(pending, v)
		}
	}
	return pending
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
