package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// BlockOverlapError reports when two block_id's with the same service_id overlap in time.
type BlockOverlapError struct {
	BlockID        string
	ServiceID      string
	TripID         string
	StartTime      tt.Seconds
	EndTime        tt.Seconds
	OtherTripID    string
	OtherStartTime tt.Seconds
	OtherEndTime   tt.Seconds
	bc
}

func (e *BlockOverlapError) Error() string {
	return fmt.Sprintf(
		"trip '%s' with block_id '%s' and service_id '%s' has interval %s -> %s which overlaps another trip '%s' in the same block with interval %s -> %s",
		e.TripID,
		e.BlockID,
		e.ServiceID,
		e.StartTime.String(),
		e.EndTime.String(),
		e.OtherTripID,
		e.OtherStartTime.String(),
		e.OtherEndTime.String(),
	)
}

type tripBlockInfo struct {
	trip    string
	service string
	start   int
	end     int
}

// BlockOverlapCheck checks for BlockOverlapErrors.
type BlockOverlapCheck struct {
	blocks map[string][]*tripBlockInfo
}

// Validate .
func (e *BlockOverlapCheck) Validate(ent tt.Entity) []error {
	trip, ok := ent.(*gtfs.Trip)
	if !ok || !trip.BlockID.Valid || len(trip.StopTimes) < 2 {
		return nil
	}
	if e.blocks == nil {
		e.blocks = map[string][]*tripBlockInfo{}
	}
	var errs []error
	// To make life easy, we only care about when the vehicle is moving.
	// intervals are: (first departure, last arrival)
	tf := tripBlockInfo{
		trip:    trip.TripID.Val,
		service: trip.ServiceID.Val,
		start:   trip.StopTimes[0].DepartureTime.Int(),
		end:     trip.StopTimes[len(trip.StopTimes)-1].ArrivalTime.Int(),
	}
	for _, hit := range e.blocks[trip.BlockID.Val] {
		// log.Log(
		// 	"block:", trip.BlockID,
		// 	"overlap?", tf,
		// 	"hit:", hit,
		// 	"service:", trip.ServiceID == hit.service,
		// 	"start:", tf.start <= hit.end,
		// 	"end:", tf.end >= hit.start,
		// )
		if trip.ServiceID.Val == hit.service && tf.start < hit.end && tf.end > hit.start {
			errs = append(errs, &BlockOverlapError{
				TripID:         tf.trip,
				BlockID:        trip.BlockID.Val,
				ServiceID:      tf.service,
				StartTime:      tt.NewSeconds(tf.start),
				EndTime:        tt.NewSeconds(tf.end),
				OtherTripID:    hit.trip,
				OtherStartTime: tt.NewSeconds(hit.start),
				OtherEndTime:   tt.NewSeconds(hit.end),
			})
		}
	}
	e.blocks[trip.BlockID.Val] = append(e.blocks[trip.BlockID.Val], &tf)
	return errs
}
