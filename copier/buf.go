package copier

import (
	"fmt"

	"github.com/interline-io/gotransit"
)

////////////////////////

// BufferedWriter .
type BufferedWriter struct {
	bufferSize int
	emap       *gotransit.EntityMap // shared
	buffer     []gotransit.Entity
	gotransit.Writer
}

// AddEntity .
func (w *BufferedWriter) AddEntity(ent gotransit.Entity) (string, error) {
	if len(w.buffer) > 0 {
		if ent.Filename() != w.buffer[len(w.buffer)-1].Filename() {
			w.Flush()
		}
	}
	w.buffer = append(w.buffer, ent)
	if w.bufferSize > 0 && len(w.buffer) > w.bufferSize {
		w.Flush()
	}
	return "", nil
}

// AddEntities .
func (w *BufferedWriter) AddEntities(ents []gotransit.Entity) error {
	w.buffer = append(w.buffer, ents...)
	if w.bufferSize > 0 && len(w.buffer) > w.bufferSize {
		w.Flush()
	}
	return nil
}

// Flush .
func (w *BufferedWriter) Flush() error {
	fmt.Println("FLUSH", len(w.buffer))
	if len(w.buffer) == 0 {
		return nil
	}
	efn := w.buffer[0].Filename()
	sids := []string{}
	for _, ent := range w.buffer {
		if ent.Filename() != efn {
			panic("buffer must contain only one type of entity")
		}
		sids = append(sids, ent.EntityID())
	}
	if err := w.Writer.AddEntities(w.buffer); err != nil {
		panic(err)
	}
	for i, ent := range w.buffer {
		eid := ent.EntityID()
		if eid != "" {
			w.emap.Set(ent.Filename(), sids[i], eid)
		}
	}
	w.buffer = nil
	return nil
}
