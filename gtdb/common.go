package gtdb

import (
	"errors"
)

func find(adapter Adapter, dest interface{}, args ...interface{}) error {
	eid, err := getID(dest)
	if err != nil {
		return err
	}
	qstr, args, err := adapter.Sqrl().Select("*").From(getTableName(dest)).Where("id = ?", eid).ToSql()
	if err != nil {
		return err
	}
	return adapter.Get(dest, qstr, args...)
}

// update a single record.
func update(adapter Adapter, ent interface{}, columns ...string) error {
	entid, err := getID(ent)
	if err != nil {
		return errors.New("cant set ID")
	}
	if v, ok := ent.(canUpdateTimestamps); ok {
		v.UpdateTimestamps()
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
