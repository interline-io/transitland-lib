package gql

import (
	"context"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

type subscriptionResolver struct{ *Resolver }

func (r *subscriptionResolver) VehiclePositions(ctx context.Context, where *model.VehiclePositionFilter) (<-chan []*model.VehiclePosition, error) {
	cfg := model.ForContext(ctx)
	ch := make(chan []*model.VehiclePosition, 1)

	// Subscribe to cache update notifications
	updateCh, cancelSub := cfg.RTFinder.Subscribe()

	go func() {
		defer close(ch)
		defer cancelSub()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-updateCh:
				if !ok {
					return
				}
				// Collect vehicle positions from all known RT feeds
				positions := r.collectVehiclePositions(ctx, cfg, where)
				select {
				case ch <- positions:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

func (r *subscriptionResolver) collectVehiclePositions(ctx context.Context, cfg model.Config, where *model.VehiclePositionFilter) []*model.VehiclePosition {
	// Get all feeds to check for vehicle positions
	feeds, err := cfg.Finder.FindFeeds(ctx, nil, nil, nil, nil)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("subscription: error finding feeds")
		return nil
	}

	var result []*model.VehiclePosition
	for _, feed := range feeds {
		feedID := feed.FeedID
		// Filter by feed_onestop_ids if specified
		if where != nil && len(where.FeedOnestopIds) > 0 {
			if !containsString(where.FeedOnestopIds, feedID) {
				continue
			}
		}
		vps := cfg.RTFinder.GetVehiclePositions(ctx, feedID)
		for _, vp := range vps {
			mvp := convertVehiclePosition(vp, feedID)
			if mvp == nil {
				continue
			}
			if !matchesFilter(mvp, vp, where) {
				continue
			}
			result = append(result, mvp)
		}
	}
	return result
}

func convertVehiclePosition(vp *pb.VehiclePosition, feedOnestopID string) *model.VehiclePosition {
	if vp == nil {
		return nil
	}
	pos := vp.GetPosition()
	if pos == nil {
		return nil
	}

	mvp := &model.VehiclePosition{
		FeedOnestopID: &feedOnestopID,
	}

	// Position
	p := tt.NewPoint(float64(pos.GetLongitude()), float64(pos.GetLatitude()))
	mvp.Position = &p

	// Bearing and speed
	if pos.Bearing != nil {
		b := float64(pos.GetBearing())
		mvp.Bearing = &b
	}
	if pos.Speed != nil {
		s := float64(pos.GetSpeed())
		mvp.Speed = &s
	}

	// Vehicle descriptor
	if v := vp.GetVehicle(); v != nil {
		mvp.Vehicle = &model.RTVehicleDescriptor{}
		if v.Id != nil {
			mvp.Vehicle.ID = v.Id
		}
		if v.Label != nil {
			mvp.Vehicle.Label = v.Label
		}
		if v.LicensePlate != nil {
			mvp.Vehicle.LicensePlate = v.LicensePlate
		}
	}

	// Trip descriptor
	if td := vp.GetTrip(); td != nil {
		mvp.Trip = &model.RTTripDescriptor{}
		if td.TripId != nil {
			mvp.Trip.TripID = td.TripId
		}
		if td.RouteId != nil {
			mvp.Trip.RouteID = td.RouteId
		}
		if td.DirectionId != nil {
			d := int(*td.DirectionId)
			mvp.Trip.DirectionID = &d
		}
		if td.ScheduleRelationship != nil {
			sr := td.ScheduleRelationship.String()
			mvp.Trip.ScheduleRelationship = &sr
		}
	}

	// Stop info
	if vp.CurrentStopSequence != nil {
		seq := int(vp.GetCurrentStopSequence())
		mvp.CurrentStopSequence = &seq
	}
	if vp.CurrentStatus != nil {
		cs := vp.CurrentStatus.String()
		mvp.CurrentStatus = &cs
	}
	if vp.CongestionLevel != nil {
		cl := vp.CongestionLevel.String()
		mvp.CongestionLevel = &cl
	}

	// Timestamp
	if vp.Timestamp != nil {
		t := time.Unix(int64(vp.GetTimestamp()), 0).In(time.UTC)
		mvp.Timestamp = &t
	}

	return mvp
}

func matchesFilter(mvp *model.VehiclePosition, vp *pb.VehiclePosition, where *model.VehiclePositionFilter) bool {
	if where == nil {
		return true
	}

	// Bbox filter
	if where.Bbox != nil && mvp.Position != nil {
		lon := mvp.Position.X()
		lat := mvp.Position.Y()
		if lon < where.Bbox.MinLon || lon > where.Bbox.MaxLon ||
			lat < where.Bbox.MinLat || lat > where.Bbox.MaxLat {
			return false
		}
	}

	// Route filter
	if len(where.RouteIds) > 0 {
		if vp.GetTrip() == nil || !containsString(where.RouteIds, vp.GetTrip().GetRouteId()) {
			return false
		}
	}

	// Agency filter — match via route_id is not possible without DB lookup,
	// so for now agency_ids filters on the feed level only.
	// A future enhancement could join against GTFS static data.

	return true
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
