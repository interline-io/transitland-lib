package mock

import "github.com/interline-io/gotransit"

// DirectCopy does a direct reader->writer copy.
func DirectCopy(reader gotransit.Reader, writer gotransit.Writer) {
	for ent := range reader.Agencies() {
		writer.AddEntity(&ent)
	}
	for ent := range reader.Routes() {
		writer.AddEntity(&ent)
	}
	for ent := range reader.Stops() {
		writer.AddEntity(&ent)
	}
	for ent := range reader.Calendars() {
		writer.AddEntity(&ent)
	}
	for ent := range reader.CalendarDates() {
		writer.AddEntity(&ent)
	}
	for ent := range reader.ShapeLinesByShapeID() {
		writer.AddEntity(&ent)
	}
	for ent := range reader.Trips() {
		writer.AddEntity(&ent)
	}
	for ents := range reader.StopTimesByTripID() {
		ents2 := []gotransit.Entity{}
		for _, st := range ents {
			ents2 = append(ents2, &st)
		}
		writer.AddEntities(ents2)
	}
	for ent := range reader.Frequencies() {
		writer.AddEntity(&ent)
	}
	for ent := range reader.Transfers() {
		writer.AddEntity(&ent)
	}
	for ent := range reader.FareAttributes() {
		writer.AddEntity(&ent)
	}
	for ent := range reader.FareRules() {
		writer.AddEntity(&ent)
	}
	for ent := range reader.FeedInfos() {
		writer.AddEntity(&ent)
	}
}
