package copier

import (
	"errors"
	"fmt"

	"github.com/interline-io/gotransit"
)

////////////////////////

// BufferedWriter .
type BufferedWriter struct {
	bufferSize int
	emap       *gotransit.EntityMap // shared
	buffer     []gotransit.Entity
	writeError error
	gotransit.Writer
}

// AddEntity .
func (w *BufferedWriter) AddEntity(ent gotransit.Entity) (string, error) {
	if w.writeError != nil {
		return "", w.writeError
	}
	if len(w.buffer) > 0 {
		if ent.Filename() != w.buffer[len(w.buffer)-1].Filename() {
			if err := w.Flush(); err != nil {
				return "", err
			}
		}
	}
	w.buffer = append(w.buffer, ent)
	if w.bufferSize > 0 && len(w.buffer) >= w.bufferSize {
		if err := w.Flush(); err != nil {
			return "", err
		}
	}
	return "", nil
}

// AddEntities .
func (w *BufferedWriter) AddEntities(ents []gotransit.Entity) error {
	for _, ent := range ents {
		if _, err := w.AddEntity(ent); err != nil {
			return err
		}
	}
	return nil
}

// Flush .
func (w *BufferedWriter) Flush() error {
	if len(w.buffer) == 0 {
		return nil
	}
	fmt.Println("FLUSH", len(w.buffer))
	efn := w.buffer[0].Filename()
	sids := []string{}
	for _, ent := range w.buffer {
		if ent.Filename() != efn {
			w.writeError = errors.New("buffer must contain only one type of entity")
			return w.writeError
		}
		sids = append(sids, ent.EntityID())
	}
	if err := w.Writer.AddEntities(w.buffer); err != nil {
		w.writeError = err
		return err
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
