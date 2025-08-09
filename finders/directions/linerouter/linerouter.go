package linerouter

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/interline-io/transitland-lib/finders/directions"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
)

func init() {
	if err := directions.RegisterRouter("line", func() directions.Handler {
		return &Router{}
	}); err != nil {
		panic(err)
	}
}

// Router is a simple point-to-point handler for testing purposes
type Router struct {
	Clock clock.Clock
}

func (h *Router) Request(ctx context.Context, req model.DirectionRequest) (*model.Directions, error) {
	// Prepare response
	ret := model.Directions{
		Origin:      wpiWaypoint(req.From),
		Destination: wpiWaypoint(req.To),
		Success:     true,
		Exception:   nil,
	}
	if err := directions.ValidateDirectionRequest(req); err != nil {
		ret.Success = false
		ret.Exception = aws.String("invalid input")
		return &ret, nil
	}

	departAt := time.Now().In(time.UTC)
	if h.Clock != nil {
		departAt = h.Clock.Now()
	}
	if req.DepartAt == nil {
		req.DepartAt = &departAt
	} else {
		departAt = *req.DepartAt
	}
	// Ensure we are in UTC
	departAt = departAt.In(time.UTC)

	distance := tlxy.DistanceHaversine(tlxy.Point{Lon: req.From.Lon, Lat: req.From.Lat}, tlxy.Point{Lon: req.To.Lon, Lat: req.To.Lat}) / 1000.0
	speed := 1.0 // m/s
	switch req.Mode {
	case model.StepModeAuto:
		speed = 10
	case model.StepModeBicycle:
		speed = 4
	case model.StepModeWalk:
		speed = 1
	case model.StepModeTransit:
		speed = 5
	}
	duration := float64(distance * 1000 / speed)

	// Create itinerary summary
	itin := model.Itinerary{}
	itin.Duration = makeDuration(duration)
	itin.Distance = makeDistance(distance, "")
	itin.StartTime = departAt
	itin.EndTime = departAt.Add(time.Duration(duration) * time.Second)

	ret.Duration = itin.Duration
	ret.Distance = itin.Distance
	ret.StartTime = &itin.StartTime
	ret.EndTime = &itin.EndTime
	ret.DataSource = aws.String("LINE")

	// Create legs and steps for itinerary
	step := model.Step{}
	step.Duration = makeDuration(duration)
	step.Distance = makeDistance(distance, "")
	step.StartTime = departAt
	step.EndTime = departAt.Add(time.Duration(duration) * time.Second)
	step.GeometryOffset = 0

	leg := model.Leg{}
	leg.Steps = append(leg.Steps, &step)
	leg.Duration = makeDuration(duration)
	leg.Distance = makeDistance(distance, "")
	leg.StartTime = departAt
	leg.EndTime = departAt.Add(time.Duration(duration) * time.Second)
	leg.Geometry = tt.NewLineStringFromFlatCoords([]float64{
		req.From.Lon, req.From.Lat, 0.0,
		req.To.Lon, req.To.Lat, 0.0,
	})

	itin.Legs = append(itin.Legs, &leg)
	if len(itin.Legs) > 0 {
		ret.Itineraries = append(ret.Itineraries, &itin)
	}
	return &ret, nil
}

func wpiWaypoint(w *model.WaypointInput) *model.Waypoint {
	if w == nil {
		return nil
	}
	return &model.Waypoint{
		Lon:  w.Lon,
		Lat:  w.Lat,
		Name: w.Name,
	}
}

func makeDuration(t float64) *model.Duration {
	return &model.Duration{Duration: float64(t), Units: model.DurationUnitSeconds}
}

func makeDistance(v float64, units string) *model.Distance {
	_ = units
	return &model.Distance{Distance: v, Units: model.DistanceUnitKilometers}
}
