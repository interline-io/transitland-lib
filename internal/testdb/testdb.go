package testdb

import (
	"errors"
	"testing"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tlsql"
)

// MustInsert panics on failure
func MustInsert(atx tldb.Adapter, ent interface{}) int {
	id, err := atx.Insert(ent)
	if err != nil {
		panic(err)
	}
	return id
}

// MustUpdate panics on failure
func MustUpdate(atx tldb.Adapter, ent interface{}, columns ...string) {
	err := atx.Update(ent, columns...)
	if err != nil {
		panic(err)
	}
}

// MustFind panics on failure
func MustFind(atx tldb.Adapter, ent interface{}, qargs ...interface{}) {
	err := atx.Find(ent, qargs...)
	if err != nil {
		panic(err)
	}
}

// MustGet panics on failure
func MustGet(atx tldb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Get(ent, qstr, qargs...)
	if err != nil {
		panic(err)
	}
}

// MustSelect panics on failure
func MustSelect(atx tldb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Select(ent, qstr, qargs...)
	if err != nil {
		panic(err)
	}
}

////////////

// ShouldInsert sends a test error on failure
func ShouldInsert(t *testing.T, atx tldb.Adapter, ent interface{}) int {
	id, err := atx.Insert(ent)
	if err != nil {
		t.Errorf("failed insert: %s", err.Error())
	}
	return id
}

// ShouldUpdate sends a test error on failure
func ShouldUpdate(t *testing.T, atx tldb.Adapter, ent interface{}, columns ...string) {
	err := atx.Update(ent, columns...)
	if err != nil {
		t.Errorf("failed update: %s", err.Error())
	}
}

// ShouldFind pasends a test error on failure
func ShouldFind(t *testing.T, atx tldb.Adapter, ent interface{}, qargs ...interface{}) {
	err := atx.Find(ent, qargs...)
	if err != nil {
		t.Errorf("failed find: %s", err.Error())
	}
}

// ShouldGet pansends a test error on failure
func ShouldGet(t *testing.T, atx tldb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Get(ent, qstr, qargs...)
	if err != nil {
		t.Errorf("failed get: %s", err.Error())
	}
}

// ShouldSelect sends a test error on failure
func ShouldSelect(t *testing.T, atx tldb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Select(ent, qstr, qargs...)
	if err != nil {
		t.Errorf("failed select: %s", err.Error())
	}
}

////////////

// WithAdapterRollback runs a callback inside a Tx and then aborts, returns any error from original callback.
func WithAdapterRollback(cb func(tldb.Adapter) error) error {
	var err error
	cb2 := func(atx tldb.Adapter) error {
		err = cb(atx)
		return errors.New("rollback")
	}
	WithAdapterTx(cb2)
	return err
}

// WithAdapterTx runs a callback inside a Tx, commits if callback returns nil.
func WithAdapterTx(cb func(tldb.Adapter) error) error {
	adapter := tlsql.SQLiteAdapter{DBURL: "sqlite3://:memory:"}
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

// AdapterIgnoreTx .
type AdapterIgnoreTx struct {
	tldb.Adapter
}

// Tx runs in same tx if tx already open, otherwise runs without tx
func (atx *AdapterIgnoreTx) Tx(cb func(tldb.Adapter) error) error {
	return cb(atx)
}

// CreateTestFeed returns a simple feed inserted into a database.
func CreateTestFeed(atx tldb.Adapter, url string) tl.Feed {
	// Create dummy feed
	tlfeed := tl.Feed{}
	tlfeed.FeedID = url
	tlfeed.URLs.StaticCurrent = url
	tlfeed.ID = MustInsert(atx, &tlfeed)
	return tlfeed
}
