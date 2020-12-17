package tldb

import (
	"errors"
)

// find a single record.
func find(adapter Adapter, dest interface{}, args ...interface{}) error {
	entid := 0
	if v, ok := dest.(canGetID); ok {
		entid = v.GetID()
	} else {
		return errors.New("cannot get ID")
	}
	qstr, args, err := adapter.Sqrl().Select("*").From(getTableName(dest)).Where("id = ?", entid).ToSql()
	if err != nil {
		return err
	}
	return adapter.Get(dest, qstr, args...)
}

// update a single record.
func update(adapter Adapter, ent interface{}, columns ...string) error {
	entid := 0
	if v, ok := ent.(canGetID); ok {
		entid = v.GetID()
	} else {
		return errors.New("cannot get ID")
	}
	table := getTableName(ent)
	cols, vals, err := getInsert(ent)
	if err != nil {
		return err
	}
	colmap := make(map[string]interface{})
	for i, col := range cols {
		if len(columns) > 0 && !contains(col, columns) {
			continue
		}
		colmap[col] = vals[i]
	}
	_, err2 := adapter.Sqrl().
		Update(table).
		Where("id = ?", entid).
		SetMap(colmap).
		Exec()
	return err2
}
