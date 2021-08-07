package tldb

import (
	"database/sql"
	"net/url"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlsql"
	"github.com/mattn/go-sqlite3"
)

type Adapter = tlsql.Adapter

const bufferSize = 1000

func check(err error) {
	if err != nil {
		log.Debug("Error: %s", err)
	}
}

// Register.
func init() {
	initPostgres()
	initSqlite()
}

type canSetID interface {
	SetID(int)
}

type canUpdateTimestamps interface {
	UpdateTimestamps()
}

type canSetFeedVersion interface {
	SetFeedVersionID(int)
}

var adapters = map[string]func(string) Adapter{}

// newAdapter returns a Adapter for the given dburl.
func newAdapter(dburl string) Adapter {
	u, err := url.Parse(dburl)
	if err != nil {
		return nil
	}
	fn, ok := adapters[u.Scheme]
	if !ok {
		return nil
	}
	return fn(dburl)
}

func initPostgres() {
	// Register driver
	adapters["postgres"] = func(dburl string) Adapter { return &tlsql.PostgresAdapter{DBURL: dburl} }
	// Register readers and writers
	r := func(url string) (tl.Reader, error) { return NewReader(url) }
	ext.RegisterReader("postgres", r)
	w := func(url string) (tl.Writer, error) { return NewWriter(url) }
	ext.RegisterWriter("postgres", w)
}

func initSqlite() {
	// Register test adapter
	adapters["sqlite3"] = func(dburl string) Adapter { return &tlsql.SQLiteAdapter{DBURL: dburl} }
	// Register readers and writers
	r := func(url string) (tl.Reader, error) { return NewReader(url) }
	ext.RegisterReader("sqlite3", r)
	w := func(url string) (tl.Writer, error) { return NewWriter(url) }
	ext.RegisterWriter("sqlite3", w)
	// Dummy handlers for SQL functions.
	dummy := func(fvid int) int {
		return 0
	}
	sqlfuncs := []string{
		"tl_generate_agency_geometries",
		"tl_generate_agency_places",
		"tl_generate_feed_version_geometries",
		"tl_generate_onestop_ids",
		"tl_generate_route_geometries",
		"tl_generate_route_headways",
		"tl_generate_route_stops",
	}
	sql.Register("sqlite3_w_funcs",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				for _, f := range sqlfuncs {
					if err := conn.RegisterFunc(f, dummy, true); err != nil {
						return err
					}
				}
				return nil
			},
		},
	)
}
