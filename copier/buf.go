package copier

import (
	"fmt"

	"github.com/interline-io/gotransit"
)

////////////////////////

// bufferedWriter .
type bufferedWriter struct {
	bufferSize int
	emap       *gotransit.EntityMap // shared
	buffer     []gotransit.Entity
	writeError error
	gotransit.Writer
}

func (w *bufferedWriter) Close() error {
	if err := w.Flush(); err != nil {
		return err
	}
	return w.Writer.Close()
}

// AddEntity .
func (w *bufferedWriter) AddEntity(ent gotransit.Entity) (string, error) {
	if err := w.AddEntities([]gotransit.Entity{ent}); err != nil {
		return "", err
	}
	// Return a temporary ID
	return ent.EntityID(), nil
}

// AddEntities .
func (w *bufferedWriter) AddEntities(ents []gotransit.Entity) error {
	if w.writeError != nil {
		return w.writeError
	}
	if len(ents) == 0 {
		return nil
	}
	efn := ents[0].Filename()
	if len(w.buffer) > 0 {
		efn = w.buffer[len(w.buffer)-1].Filename()
	}
	for _, ent := range ents {
		if ent.Filename() != efn {
			return fmt.Errorf("buffer must contain only one type of entity, last was %s", efn)
		}
	}
	w.buffer = append(w.buffer, ents...)
	if w.bufferSize > 0 && len(w.buffer) >= w.bufferSize {
		if err := w.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// Flush .
func (w *bufferedWriter) Flush() error {
	if len(w.buffer) == 0 {
		return nil
	}
	sids := []string{}
	for _, ent := range w.buffer {
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
