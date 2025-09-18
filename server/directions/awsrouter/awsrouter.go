package awsrouter

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/location"
	"github.com/aws/aws-sdk-go-v2/service/location/types"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/server/caches/httpcache"
	"github.com/interline-io/transitland-lib/server/directions"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

type LocationClient interface {
	CalculateRoute(context.Context, *location.CalculateRouteInput, ...func(*location.Options)) (*location.CalculateRouteOutput, error)
}

func init() {
	ctx := context.Background()
	// Get AWS config and register handler factory
	cn := os.Getenv("TL_AWS_LOCATION_CALCULATOR")
	if cn == "" {
		return
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	if os.Getenv("TL_DIRECTIONS_ENABLE_CACHE") != "" {
		// By default use a 1 minute TTL cache
		cache := httpcache.NewTTLCache(16*1024, 1*time.Minute)
		cache.SkipExtension(true) // don't refresh values on get
		client.Transport = httpcache.NewCache(nil, httpcache.NoHeadersKey, cache)

	}
	cfg.HTTPClient = client
	lc := location.NewFromConfig(cfg)
	if err := directions.RegisterRouter("aws", func() directions.Handler {
		return NewRouter(lc, cn)
	}); err != nil {
		panic(err)
	}
}

type Router struct {
	CalculatorName string
	Clock          clock.Clock
	locationClient LocationClient
}

func NewRouter(lc LocationClient, calculator string) *Router {
	return &Router{
		CalculatorName: calculator,
		Clock:          &clock.Real{},
		locationClient: lc,
	}
}

func (h *Router) Request(ctx context.Context, req model.DirectionRequest) (*model.Directions, error) {
	// Input validation
	if err := directions.ValidateDirectionRequest(req); err != nil {
		return &model.Directions{Success: false, Exception: aws.String("invalid input")}, nil
	}

	// Prepare request
	input := location.CalculateRouteInput{
		CalculatorName:      aws.String(h.CalculatorName),
		DeparturePosition:   []float64{req.From.Lon, req.From.Lat},
		DestinationPosition: []float64{req.To.Lon, req.To.Lat},
		DistanceUnit:        types.DistanceUnit("Kilometers"),
		IncludeLegGeometry:  aws.Bool(true),
	}
	if req.Mode == model.StepModeAuto {
		input.TravelMode = types.TravelMode("Car")
	} else if req.Mode == model.StepModeWalk {
		input.TravelMode = types.TravelMode("Walking")
	} else {
		return &model.Directions{Success: false, Exception: aws.String("unsupported travel mode")}, nil
	}
	// Departure time
	now := time.Now().In(time.UTC)
	if h.Clock != nil {
		now = h.Clock.Now()
	}
	var departAt time.Time
	if req.DepartAt == nil {
		departAt = now
		input.DepartNow = aws.Bool(true)
	} else {
		departAt = *req.DepartAt
		input.DepartureTime = req.DepartAt
		input.DepartNow = nil
	}
	// Ugly hack for testing
	// If departAt is in the past, don't send any time info with request
	if departAt.Before(now) {
		input.DepartNow = nil
		input.DepartureTime = nil
	}
	// Ensure we are in UTC
	departAt = departAt.In(time.UTC)

	// Make request
	res, err := h.locationClient.CalculateRoute(ctx, &input)
	if err != nil || res.Summary == nil {
		log.For(ctx).Debug().Err(err).Msg("aws location services error")
		return &model.Directions{Success: false, Exception: aws.String("could not calculate route")}, nil
	}

	// Prepare response
	ret := makeDirections(res, departAt)
	ret.Origin = wpiWaypoint(req.From)
	ret.Destination = wpiWaypoint(req.To)
	ret.Success = true
	ret.Exception = nil
	return ret, nil
}

func makeDirections(res *location.CalculateRouteOutput, departAt time.Time) *model.Directions {
	// Create itinerary summary
	ret := model.Directions{}
	itin := model.Itinerary{}
	distUnits := res.Summary.DistanceUnit
	itin.Duration = makeDuration(res.Summary.DurationSeconds)
	itin.Distance = makeDistance(res.Summary.Distance, distUnits)
	itin.StartTime = departAt
	if res.Summary.DurationSeconds != nil {
		itin.EndTime = departAt.Add(time.Duration(*res.Summary.DurationSeconds) * time.Second)
	}
	// aws responses have single itineraries
	ret.Duration = itin.Duration
	ret.Distance = itin.Distance
	ret.StartTime = &itin.StartTime
	ret.EndTime = &itin.EndTime
	ret.DataSource = res.Summary.DataSource

	// Create legs for itinerary
	prevLegDepartAt := departAt
	for _, awsleg := range res.Legs {
		if awsleg.DurationSeconds == nil {
			return &model.Directions{Success: false, Exception: aws.String("invalid route response")}
		}
		leg := model.Leg{}
		prevStepDepartAt := prevLegDepartAt
		for _, awsstep := range awsleg.Steps {
			step := model.Step{}
			step.Duration = makeDuration(awsstep.DurationSeconds)
			step.Distance = makeDistance(awsstep.Distance, distUnits)
			step.StartTime = prevStepDepartAt
			step.EndTime = prevStepDepartAt.Add(time.Duration(*awsstep.DurationSeconds) * time.Second)
			step.To = awsWaypoint(awsstep.EndPosition)
			step.GeometryOffset = awsInt(awsstep.GeometryOffset)
			prevStepDepartAt = step.EndTime
			leg.Steps = append(leg.Steps, &step)
		}
		leg.Duration = makeDuration(awsleg.DurationSeconds)
		leg.Distance = makeDistance(awsleg.Distance, distUnits)
		leg.StartTime = prevLegDepartAt
		leg.EndTime = prevLegDepartAt.Add(time.Duration(*awsleg.DurationSeconds) * time.Second)
		leg.From = awsWaypoint(awsleg.StartPosition)
		leg.To = awsWaypoint(awsleg.EndPosition)
		prevLegDepartAt = leg.EndTime
		if awsleg.Geometry != nil {
			leg.Geometry = awsLineString(awsleg.Geometry.LineString)
		}
		itin.Legs = append(itin.Legs, &leg)
	}
	if len(itin.Legs) > 0 {
		ret.Itineraries = append(ret.Itineraries, &itin)
	}
	return &ret
}

func awsInt(v *int32) int {
	if v == nil {
		return 0
	}
	return int(*v)
}

func awsLineString(v [][]float64) tt.LineString {
	coords := []float64{}
	for _, coord := range v {
		if len(coord) == 2 {
			coords = append(coords, coord[0], coord[1], 0)
		}
	}
	return tt.NewLineStringFromFlatCoords(coords)
}

func awsWaypoint(v []float64) *model.Waypoint {
	if len(v) != 2 {
		return nil
	}
	return &model.Waypoint{
		Lon: v[0],
		Lat: v[1],
	}
}

func makeDuration(v *float64) *model.Duration {
	if v == nil {
		return nil
	}
	r := model.Duration{
		Duration: *v,
		Units:    model.DurationUnitSeconds,
	}
	return &r
}

func makeDistance(v *float64, units types.DistanceUnit) *model.Distance {
	if v == nil || units == "" {
		return nil
	}
	r := model.Distance{}
	switch units {
	case "Kilometers":
		r.Units = model.DistanceUnitKilometers
	case "Miles":
		r.Units = model.DistanceUnitMiles
	default:
		return nil
	}
	r.Distance = *v
	return &r
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
