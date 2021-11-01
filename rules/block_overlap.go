package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

// BlockOverlapError reports when two block_id's with the same service_id overlap in time.
type BlockOverlapError struct {
	BlockID        string
	ServiceID      string
	TripID         string
	StartTime      tl.WideTime
	EndTime        tl.WideTime
	OtherTripID    string
	OtherStartTime tl.WideTime
	OtherEndTime   tl.WideTime
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
func (e *BlockOverlapCheck) Validate(ent tl.Entity) []error {
	trip, ok := ent.(*tl.Trip)
	if !ok || trip.BlockID == "" || len(trip.StopTimes) < 2 {
		return nil
	}
	if e.blocks == nil {
		e.blocks = map[string][]*tripBlockInfo{}
	}
	var errs []error
	// To make life easy, we only care about when the vehicle is moving.
	// intervals are: (first departure, last arrival)
	tf := tripBlockInfo{
		trip:    trip.TripID,
		service: trip.ServiceID,
		start:   trip.StopTimes[0].DepartureTime.Seconds,
		end:     trip.StopTimes[len(trip.StopTimes)-1].ArrivalTime.Seconds,
	}
	for _, hit := range e.blocks[trip.BlockID] {
		// fmt.Println(
		// 	"block:", trip.BlockID,
		// 	"overlap?", tf,
		// 	"hit:", hit,
		// 	"service:", trip.ServiceID == hit.service,
		// 	"start:", tf.start <= hit.end,
		// 	"end:", tf.end >= hit.start,
		// )
		if trip.ServiceID == hit.service && tf.start < hit.end && tf.end > hit.start {
			errs = append(errs, &BlockOverlapError{
				TripID:         tf.trip,
				BlockID:        trip.BlockID,
				ServiceID:      tf.service,
				StartTime:      tl.NewWideTimeFromSeconds(tf.start),
				EndTime:        tl.NewWideTimeFromSeconds(tf.end),
				OtherTripID:    hit.trip,
				OtherStartTime: tl.NewWideTimeFromSeconds(hit.start),
				OtherEndTime:   tl.NewWideTimeFromSeconds(hit.end),
			})
		}
	}
	e.blocks[trip.BlockID] = append(e.blocks[trip.BlockID], &tf)
	return errs
}
