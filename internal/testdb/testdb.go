package testdb

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tldb/sqlite"
)

func MustOpenWriter(dburl string, create bool) *tldb.Writer {
	w, err := tldb.OpenWriter(dburl, create)
	if err != nil {
		panic(err)
	}
	return w
}

// MustInsert panics on failure
func MustInsert(atx tldb.Adapter, ent interface{}) int {
	id, err := atx.Insert(context.Background(), ent)
	if err != nil {
		panic(err)
	}
	return id
}

// MustUpdate panics on failure
func MustUpdate(atx tldb.Adapter, ent interface{}, columns ...string) {
	err := atx.Update(context.Background(), ent, columns...)
	if err != nil {
		panic(err)
	}
}

// MustFind panics on failure
func MustFind(atx tldb.Adapter, ent interface{}) {
	err := atx.Find(context.Background(), ent)
	if err != nil {
		panic(err)
	}
}

// MustGet panics on failure
func MustGet(atx tldb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Get(context.Background(), ent, qstr, qargs...)
	if err != nil {
		panic(err)
	}
}

// MustSelect panics on failure
func MustSelect(atx tldb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Select(context.Background(), ent, qstr, qargs...)
	if err != nil {
		panic(err)
	}
}

////////////

// ShouldInsert sends a test error on failure
func ShouldInsert(t *testing.T, atx tldb.Adapter, ent interface{}) int {
	id, err := atx.Insert(context.Background(), ent)
	if err != nil {
		t.Errorf("failed insert: %s", err.Error())
	}
	return id
}

// ShouldUpdate sends a test error on failure
func ShouldUpdate(t *testing.T, atx tldb.Adapter, ent interface{}, columns ...string) {
	err := atx.Update(context.Background(), ent, columns...)
	if err != nil {
		t.Errorf("failed update: %s", err.Error())
	}
}

// ShouldFind sends a test error on failure
func ShouldFind(t *testing.T, atx tldb.Adapter, ent interface{}) {
	err := atx.Find(context.Background(), ent)
	if err != nil {
		t.Errorf("failed find: %s", err.Error())
	}
}

// ShouldGet sends a test error on failure
func ShouldGet(t *testing.T, atx tldb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Get(context.Background(), ent, qstr, qargs...)
	if err != nil {
		t.Errorf("failed get: %s", err.Error())
	}
}

// ShouldSelect sends a test error on failure
func ShouldSelect(t *testing.T, atx tldb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Select(context.Background(), ent, qstr, qargs...)
	if err != nil {
		t.Errorf("failed select: %s", err.Error())
	}
}

////////////

func TempPostgres(dburl string, cb func(tldb.Adapter) error) error {
	adapter := postgres.PostgresAdapter{DBURL: dburl}
	if err := adapter.Open(); err != nil {
		panic(err)
	}
	writer := tldb.Writer{Adapter: &adapter}
	if err := writer.Open(); err != nil {
		panic(err)
	}
	defer writer.Close()
	if err := writer.Create(); err != nil {
		panic(err)
	}
	return writer.Adapter.Tx(cb)
}

// TempSqlite creates a temporary in-memory database and runs the callback inside a tx.
func TempSqlite(cb func(tldb.Adapter) error) error {
	adapter := sqlite.SQLiteAdapter{DBURL: "sqlite3://:memory:"}
	writer := tldb.Writer{Adapter: &adapter}
	if err := writer.Open(); err != nil {
		panic(err)
	}
	defer writer.Close()
	if err := writer.Create(); err != nil {
		panic(err)
	}
	return writer.Adapter.Tx(cb)
}

func TempSqliteAdapter() tldb.Adapter {
	adapter := sqlite.SQLiteAdapter{DBURL: "sqlite3://:memory:"}
	writer := tldb.Writer{Adapter: &adapter}
	if err := writer.Open(); err != nil {
		panic(err)
	}
	defer writer.Close()
	if err := writer.Create(); err != nil {
		panic(err)
	}
	return writer.Adapter
}

// AdapterIgnoreTx .
type AdapterIgnoreTx struct {
	tldb.Adapter
}

// Tx runs in same tx if tx already open, otherwise runs without tx
func (atx *AdapterIgnoreTx) Tx(cb func(tldb.Adapter) error) error {
	return cb(atx)
}

// CreateTestFeed returns a simple feed inserted into a database.
func CreateTestFeed(atx tldb.Adapter, url string) dmfr.Feed {
	// Create dummy feed
	tlfeed := dmfr.Feed{}
	tlfeed.FeedID = url
	tlfeed.URLs.StaticCurrent = url
	tlfeed.ID = MustInsert(atx, &tlfeed)
	return tlfeed
}
