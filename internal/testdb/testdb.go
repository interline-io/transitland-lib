package testdb

import (
	"errors"
	"testing"

	"github.com/interline-io/gotransit/gtdb"
)

// MustInsert panics on failure
func MustInsert(atx gtdb.Adapter, ent interface{}) int {
	id, err := atx.Insert(ent)
	if err != nil {
		panic(err)
	}
	return id
}

// MustUpdate panics on failure
func MustUpdate(atx gtdb.Adapter, ent interface{}, columns ...string) {
	err := atx.Update(ent, columns...)
	if err != nil {
		panic(err)
	}
}

// MustFind panics on failure
func MustFind(atx gtdb.Adapter, ent interface{}, qargs ...interface{}) {
	err := atx.Find(ent, qargs...)
	if err != nil {
		panic(err)
	}
}

// MustGet panics on failure
func MustGet(atx gtdb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Get(ent, qstr, qargs...)
	if err != nil {
		panic(err)
	}
}

// MustSelect panics on failure
func MustSelect(atx gtdb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Select(ent, qstr, qargs...)
	if err != nil {
		panic(err)
	}
}

////////////

// ShouldInsert sends a test error on failure
func ShouldInsert(t *testing.T, atx gtdb.Adapter, ent interface{}) int {
	id, err := atx.Insert(ent)
	if err != nil {
		t.Errorf("failed insert: %s", err.Error())
	}
	return id
}

// ShouldUpdate sends a test error on failure
func ShouldUpdate(t *testing.T, atx gtdb.Adapter, ent interface{}, columns ...string) {
	err := atx.Update(ent, columns...)
	if err != nil {
		t.Errorf("failed update: %s", err.Error())
	}
}

// ShouldFind pasends a test error on failure
func ShouldFind(t *testing.T, atx gtdb.Adapter, ent interface{}, qargs ...interface{}) {
	err := atx.Find(ent, qargs...)
	if err != nil {
		t.Errorf("failed find: %s", err.Error())
	}
}

// ShouldGet pansends a test error on failure
func ShouldGet(t *testing.T, atx gtdb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Get(ent, qstr, qargs...)
	if err != nil {
		t.Errorf("failed get: %s", err.Error())
	}
}

// ShouldSelect sends a test error on failure
func ShouldSelect(t *testing.T, atx gtdb.Adapter, ent interface{}, qstr string, qargs ...interface{}) {
	err := atx.Select(ent, qstr, qargs...)
	if err != nil {
		t.Errorf("failed select: %s", err.Error())
	}
}

////////////

// WithAdapterRollback runs a callback inside a Tx and then aborts, returns any error from original callback.
func WithAdapterRollback(cb func(gtdb.Adapter) error) error {
	var err error
	cb2 := func(atx gtdb.Adapter) error {
		err = cb(atx)
		return errors.New("rollback")
	}
	WithAdapterTx(cb2)
	return err
}

// WithAdapterTx runs a callback inside a Tx, commits if callback returns nil.
func WithAdapterTx(cb func(gtdb.Adapter) error) error {
	adapter := gtdb.SQLiteAdapter{DBURL: "sqlite3://:memory:"}
	writer := gtdb.Writer{Adapter: &adapter}
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
	gtdb.Adapter
}

// Tx runs in same tx if tx already open, otherwise runs without tx
func (atx *AdapterIgnoreTx) Tx(cb func(gtdb.Adapter) error) error {
	return cb(atx)
}
