package testutil

import (
	"fmt"

	"github.com/interline-io/gotransit"
)

// DirectCopy does a direct reader->writer copy, with minimal validation and changes.
func DirectCopy(reader gotransit.Reader, writer gotransit.Writer) error {
	emap := gotransit.NewEntityMap()
	cp := func(ent gotransit.Entity) error {
		sid := ent.EntityID()
		if err := ent.UpdateKeys(emap); err != nil {
			return fmt.Errorf("entity: %#v error: %s", ent, err)
		}
		eid, err := writer.AddEntity(ent)
		if err != nil {
			return fmt.Errorf("entity: %#v error: %s", ent, err)
		}
		return emap.Set(ent, sid, eid)
	}
	for ent := range reader.Agencies() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	for ent := range reader.Routes() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	for ent := range reader.Stops() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	for ent := range reader.Calendars() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	for ent := range reader.CalendarDates() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	for ent := range reader.Shapes() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	for ent := range reader.Trips() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	for ents := range reader.StopTimesByTripID() {
		for _, ent := range ents {
			if err := cp(&ent); err != nil {
				return err
			}
		}
	}
	for ent := range reader.Frequencies() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	for ent := range reader.Transfers() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	for ent := range reader.FareAttributes() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	for ent := range reader.FareRules() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	for ent := range reader.FeedInfos() {
		if err := cp(&ent); err != nil {
			return err
		}
	}
	return nil
}
