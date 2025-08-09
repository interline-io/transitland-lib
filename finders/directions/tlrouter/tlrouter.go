package tlrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/finders/directions"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/interline-io/transitland-mw/caches/httpcache"
)

func init() {
	apikey := os.Getenv("TL_TLROUTER_APIKEY")
	endpoint := os.Getenv("TL_TLROUTER_ENDPOINT")
	if endpoint == "" {
		return
	}
	// TODO: configurable timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	if os.Getenv("TL_DIRECTIONS_ENABLE_CACHE") != "" {
		client.Transport = httpcache.NewCache(nil, nil, httpcache.NewTTLCache(16*1024, 24*time.Hour))
	}
	if err := directions.RegisterRouter("tlrouter", func() directions.Handler {
		return NewRouter(client, endpoint, apikey)
	}); err != nil {
		panic(err)
	}
}

type Router struct {
	Clock    clock.Clock
	client   *http.Client
	endpoint string
	apikey   string
}

func NewRouter(client *http.Client, endpoint string, apikey string) *Router {
	if client == nil {
		client = http.DefaultClient
	}
	return &Router{
		client:   client,
		endpoint: endpoint,
		apikey:   apikey,
	}
}

func (h *Router) Request(ctx context.Context, req model.DirectionRequest) (*model.Directions, error) {
	if err := directions.ValidateDirectionRequest(req); err != nil {
		return &model.Directions{Success: false, Exception: aws.String("invalid input")}, nil
	}

	// Prepare request
	input := Request{}
	input.FromPlace = RequestLocation{Lat: req.From.Lat, Lon: req.From.Lon}
	input.ToPlace = RequestLocation{Lat: req.To.Lat, Lon: req.To.Lon}
	if req.Mode == model.StepModeTransit {
		input.Mode = "TRANSIT"
	} else if req.Mode == model.StepModeBicycle {
		input.Mode = "BICYCLE"
	} else if req.Mode == model.StepModeWalk {
		input.Mode = "WALK"
	} else {
		return &model.Directions{Success: false, Exception: aws.String("unsupported travel mode")}, nil
	}

	// Prepare departure time
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
	input.UnixTime = departAt.Unix()

	// Make request
	res, err := makeRequest(ctx, input, h.client, h.endpoint, h.apikey)
	if err != nil || len(res.Plan.Itineraries) == 0 {
		log.For(ctx).Error().Err(err).Msg("tlrouter: failed to calculate route")
		return &model.Directions{Success: false, Exception: aws.String("could not calculate route")}, nil
	}
	// Prepare response
	ret := makeDirections(res)
	ret.Origin = wpiWaypoint(req.From)
	ret.Destination = wpiWaypoint(req.To)
	ret.Success = true
	ret.Exception = nil
	return ret, nil
}

func makeRequest(ctx context.Context, req Request, client *http.Client, endpoint string, apikey string) (*PlanResponse, error) {
	parsedUrl, err := url.Parse(fmt.Sprintf("%s/plan", endpoint))
	if err != nil {
		return nil, err
	}
	q := parsedUrl.Query()
	q.Add("fromPlace", fmt.Sprintf("%f,%f", req.FromPlace.Lat, req.FromPlace.Lon))
	q.Add("toPlace", fmt.Sprintf("%f,%f", req.ToPlace.Lat, req.ToPlace.Lon))
	q.Add("unixTime", fmt.Sprintf("%d", req.UnixTime))
	q.Add("mode", req.Mode)
	q.Add("includeWalkingItinerary", "true")
	if req.UseFallbackDates {
		q.Add("useFallbackDates", "true")
	}
	parsedUrl.RawQuery = q.Encode()
	reqUrl := parsedUrl.String()

	reqJson, _ := json.Marshal(req)

	// Make request
	hreq, err := http.NewRequest("GET", reqUrl, bytes.NewReader(reqJson))
	if err != nil {
		return nil, errors.Join(errors.New("failed to create request"), err)
	}

	hreq.Header.Add("api_key", apikey)
	log.TraceCheck(func() {
		log.For(ctx).Trace().Str("url", hreq.URL.String()).Msg("tlrouter: request")
	})
	resp, err := client.Do(hreq)
	if err != nil {
		return nil, errors.Join(errors.New("request failed"), err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Join(errors.New("failed to read response"), err)
	}
	res := PlanResponse{}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, errors.Join(errors.New("failed to read response as JSON"), err)
	}
	return &res, nil
}

func legLocationToWaypoint(v LegLocation) *model.Waypoint {
	wp := model.Waypoint{
		Lon:  v.Lon,
		Lat:  v.Lat,
		Name: aws.String(v.Name),
	}
	if v.StopId != "" {
		wps := model.WaypointStop{}
		wps.Lat = v.Lat
		wps.Lon = v.Lon
		wps.StopID = v.StopId
		wps.StopName = v.Name
		wps.StopCode = v.StopCode
		wps.StopOnestopID = v.StopOnestopId
		wps.Departure = otpMs(v.Departure)
		wp.Stop = &wps
	}
	return &wp
}

func makeDirections(res *PlanResponse) *model.Directions {
	// Map PlanResponse to Directions
	ret := model.Directions{}
	ret.DataSource = aws.String("OSM, Transitland")
	for _, vitin := range res.Plan.Itineraries {
		itin := model.Itinerary{}
		itin.From = &model.Waypoint{Lon: res.Plan.From.Lon, Lat: res.Plan.From.Lat, Name: aws.String(res.Plan.From.Name)}
		itin.To = &model.Waypoint{Lon: res.Plan.To.Lon, Lat: res.Plan.To.Lat, Name: aws.String(res.Plan.To.Name)}
		itin.Duration = makeDuration(float64(vitin.Duration))
		itin.Distance = makeDistance(vitin.Distance, "km")

		// Convert times
		itin.StartTime = otpMs(vitin.StartTime)
		itin.EndTime = otpMs(vitin.EndTime)

		// Create legs for itinerary
		for _, vleg := range vitin.Legs {
			leg := model.Leg{}

			// Setup leg
			var sm model.StepMode
			switch vleg.Mode {
			case "BICYCLE":
				sm = model.StepMode(model.StepModeBicycle.String())
			case "TRANSIT":
				sm = model.StepMode(model.StepModeTransit.String())
			default:
				sm = model.StepMode(model.StepModeWalk.String())
			}
			leg.Mode = &sm
			leg.From = legLocationToWaypoint(vleg.From)
			leg.To = legLocationToWaypoint(vleg.To)

			// Process transit trip
			if vleg.TripId != "" {
				leg.Trip = &model.LegTrip{
					TripID:          vleg.TripId,
					TripShortName:   vleg.RouteShortName,
					Headsign:        vleg.Headsign,
					FeedID:          vleg.FeedId,
					FeedVersionSha1: vleg.FeedVersionSha1,
					// BlockID todo
					Route: &model.LegRoute{
						RouteID:        vleg.RouteId,
						RouteShortName: vleg.RouteShortName,
						RouteLongName:  vleg.RouteLongName,
						RouteType:      vleg.RouteType,
						RouteColor:     aws.String(vleg.RouteColor),
						RouteTextColor: aws.String(vleg.RouteTextColor),
						RouteOnestopID: vleg.RouteOnestopId,
						Agency: &model.LegRouteAgency{
							AgencyID:        vleg.AgencyId,
							AgencyName:      vleg.AgencyName,
							AgencyOnestopID: "", // TODO: vleg.AgencyOnestopId,
						},
					},
				}
				// Process stops
				leg.Stops = append(leg.Stops, legLocationToWaypointDeparture(vleg.From))
				for _, vstop := range vleg.IntermediateStops {
					leg.Stops = append(leg.Stops, stopToWaypointDeparture(vstop))
				}
				leg.Stops = append(leg.Stops, legLocationToWaypointDeparture(vleg.To))
			}

			// Process steps
			for _, vstep := range vleg.Steps {
				_ = vstep
				step := model.Step{}
				leg.Steps = append(leg.Steps, &step)
			}

			leg.Duration = makeDuration(float64(vleg.Duration))
			leg.Distance = makeDistance(vleg.Distance, "km")
			leg.StartTime = otpMs(vleg.StartTime)
			leg.EndTime = otpMs(vleg.EndTime)

			// TODO: decode points
			if c, err := tlxy.DecodePolylineString(vleg.LegGeometry.Points); err == nil {
				var coords []float64
				for _, v := range c {
					coords = append(coords, v.Lon, v.Lat, 0)
				}
				leg.Geometry = tt.NewLineStringFromFlatCoords(coords)
			}

			// Append leg
			itin.Legs = append(itin.Legs, &leg)
		}
		if len(itin.Legs) > 0 {
			ret.Itineraries = append(ret.Itineraries, &itin)
		}
	}
	if len(ret.Itineraries) > 0 {
		r0 := ret.Itineraries[0]
		ret.Duration = r0.Duration
		ret.Distance = r0.Distance
		ret.StartTime = &r0.StartTime
		ret.EndTime = &r0.EndTime
	}
	return &ret
}

func legLocationToWaypointDeparture(v LegLocation) *model.WaypointDeparture {
	wp := model.WaypointDeparture{}
	wp.Lat = v.Lat
	wp.Lon = v.Lon
	wp.StopID = v.StopId
	wp.StopName = v.Name
	wp.StopCode = v.StopCode
	wp.StopOnestopID = v.StopOnestopId
	wp.Departure = otpMs(v.Departure)
	wp.StopIndex = aws.Int(v.StopIndex)
	wp.StopSequence = aws.Int(v.StopSequence)
	return &wp
}

func stopToWaypointDeparture(v Stop) *model.WaypointDeparture {
	wp := model.WaypointDeparture{}
	wp.Lat = v.Lat
	wp.Lon = v.Lon
	wp.StopID = v.StopId
	wp.StopName = v.Name
	wp.StopCode = v.StopCode
	wp.StopOnestopID = v.StopOnestopId
	wp.Departure = otpMs(v.Departure)
	wp.StopIndex = aws.Int(v.StopIndex)
	wp.StopSequence = aws.Int(v.StopSequence)
	return &wp
}

func otpMs(v int64) time.Time {
	return time.Unix(v/1000, 0).In(time.UTC)
}

type Request struct {
	// Required options
	FromPlace RequestLocation `json:"fromPlace"`
	ToPlace   RequestLocation `json:"toPlace"`
	UnixTime  int64           `json:"unixTime"`
	Time      string          `json:"time"`
	Date      string          `json:"date"`

	// Advanced options
	MaxItineraries      int     `json:"maxItineraries"`
	Mode                string  `json:"mode"`
	ArriveBy            bool    `json:"arriveBy"`
	MaxWalkingDistance  float64 `json:"maxWalkingDistance"`
	WalkingSpeed        float64 `json:"walkingSpeed"`
	MaxK                int     `json:"maxK"`
	MaxTripTime         int     `json:"maxTripTime"`
	TransferTimePenalty int     `json:"transferTimePenalty"`

	// Fallbacks
	UseFallbackDates bool `json:"useFallbackDates"`
	// includeWalkingItinerary  bool `json:"includeWalkingItinerary"`
	// fallbackWalkingItinerary bool `json:"fallbackWalkingItinerary"`
	// allowWalkingItinerary    bool `json:"allowWalkingItinerary"`
	// includeEarliestArrivals  bool `json:"includeEarliestArrivals"`
	// useTargetStopPruining    bool `json:"useTargetStopPruining"`
}

type RequestLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
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
	// Input distance is m
	if units == "km" {
		v = v / 1000.0
	}
	return &model.Distance{Distance: v, Units: model.DistanceUnitKilometers}
}

// Generated from example.json

type PlanResponse struct {
	Plan Plan `json:"plan"`
}

type Plan struct {
	Date        int64       `json:"date"`
	From        Location    `json:"from"`
	To          Location    `json:"to"`
	Itineraries []Itinerary `json:"itineraries"`
}

type Location struct {
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	Name          string  `json:"name"`
	StopOnestopId string  `json:"stopOnestopId"`
}

type Itinerary struct {
	Duration        int64   `json:"duration"`
	Distance        float64 `json:"distance"`
	StartTime       int64   `json:"startTime"`
	EndTime         int64   `json:"endTime"`
	WalkTime        int64   `json:"walkTime"`
	WalkDistance    float64 `json:"walkDistance"`
	TransitTime     int64   `json:"transitTime"`
	TransitDistance float64 `json:"transitDistance"`
	WaitingTime     int64   `json:"waitingTime"`
	Transfers       int     `json:"transfers"`
	Legs            []Leg   `json:"legs"`
}

type Leg struct {
	StartTime         int64       `json:"startTime"`
	EndTime           int64       `json:"endTime"`
	Distance          float64     `json:"distance"`
	Duration          int64       `json:"duration"`
	Mode              string      `json:"mode"`
	TransitLeg        bool        `json:"transitLeg"`
	From              LegLocation `json:"from"`
	To                LegLocation `json:"to"`
	Steps             []Step      `json:"steps"`
	LegGeometry       Geometry    `json:"legGeometry"`
	AgencyId          string      `json:"agencyId,omitempty"`
	AgencyName        string      `json:"agencyName,omitempty"`
	RouteShortName    string      `json:"routeShortName,omitempty"`
	RouteLongName     string      `json:"routeLongName,omitempty"`
	RouteType         int         `json:"routeType,omitempty"`
	RouteId           string      `json:"routeId,omitempty"`
	RouteColor        string      `json:"routeColor,omitempty"`
	RouteTextColor    string      `json:"routeTextColor,omitempty"`
	RouteOnestopId    string      `json:"routeOnestopId,omitempty"`
	TripId            string      `json:"tripId,omitempty"`
	Headsign          string      `json:"headsign,omitempty"`
	FeedId            string      `json:"feedId,omitempty"`
	FeedVersionSha1   string      `json:"feedVersionSha1,omitempty"`
	IntermediateStops []Stop      `json:"intermediateStops,omitempty"`
}

type LegLocation struct {
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	Name          string  `json:"name"`
	Departure     int64   `json:"departure"`
	StopId        string  `json:"stopId,omitempty"`
	StopCode      string  `json:"stopCode,omitempty"`
	StopIndex     int     `json:"stopIndex,omitempty"`
	StopSequence  int     `json:"stopSequence,omitempty"`
	StopOnestopId string  `json:"stopOnestopId"`
}

type Step struct {
	// Define fields for steps if needed
}

type Geometry struct {
	Points string `json:"points"`
	Length int    `json:"length"`
}

type Stop struct {
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	Name          string  `json:"name"`
	Departure     int64   `json:"departure"`
	StopId        string  `json:"stopId"`
	StopCode      string  `json:"stopCode"`
	StopIndex     int     `json:"stopIndex"`
	StopSequence  int     `json:"stopSequence"`
	StopOnestopId string  `json:"stopOnestopId"`
}
