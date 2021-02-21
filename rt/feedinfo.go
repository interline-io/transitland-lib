package rt

import (
	"github.com/interline-io/transitland-lib/tl"
)

// FeedInfo .
type FeedInfo struct {
	ids               map[string]map[string]int
	tripFrequency     map[string]bool
	tripDirections    map[string]int
	stopLocationTypes map[string]int
}

// NewFeedInfo .
func NewFeedInfo() *FeedInfo {
	return &FeedInfo{
		ids:               map[string]map[string]int{},
		tripFrequency:     map[string]bool{},
		tripDirections:    map[string]int{},
		stopLocationTypes: map[string]int{},
	}
}

// NewFeedInfoFromReader .
func NewFeedInfoFromReader(reader tl.Reader) (*FeedInfo, error) {
	fi := NewFeedInfo()
	add := func(efn string, eid string) {
		m, ok := fi.ids[efn]
		if !ok {
			m = map[string]int{}
			fi.ids[efn] = m
		}
		m[eid]++
	}
	for ent := range reader.Stops() {
		add(ent.Filename(), ent.EntityID())
		fi.stopLocationTypes[ent.StopID] = ent.LocationType
	}
	for ent := range reader.Routes() {
		add(ent.Filename(), ent.EntityID())
	}
	for ent := range reader.Trips() {
		add(ent.Filename(), ent.EntityID())
		fi.tripDirections[ent.TripID] = ent.DirectionID
	}
	for ent := range reader.Frequencies() {
		fi.tripFrequency[ent.TripID] = true
	}
	return fi, nil
}

// Contains .
func (fi *FeedInfo) Contains(efn string, eid string) bool {
	m, ok := fi.ids[efn]
	if ok {
		_, ok2 := m[eid]
		return ok2
	}
	return false
}
