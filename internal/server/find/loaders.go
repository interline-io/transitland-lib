package find

import (
	"context"
	"net/http"
	"time"

	sq "github.com/Masterminds/squirrel"
	dl "github.com/interline-io/transitland-lib/internal/server/generated/dataloader"
	"github.com/interline-io/transitland-lib/internal/server/model"
	"github.com/jmoiron/sqlx"
)

// MAXBATCH is maximum batch size
const MAXBATCH = 1000

// WAIT is the time to wait
const WAIT = 2 * time.Millisecond

const loadersKey = "dataloaders"

// TODO: Use code generation!

// Loaders .
type Loaders struct {
	// ID Loaders
	AgenciesByID     dl.AgencyLoader
	CalendarsByID    dl.CalendarLoader
	FeedsByID        dl.FeedLoader
	RoutesByID       dl.RouteLoader
	ShapesByID       dl.ShapeLoader
	StopsByID        dl.StopLoader
	FeedVersionsByID dl.FeedVersionLoader
	LevelsByID       dl.LevelLoader
	TripsByID        dl.TripLoader
	// Other ID loaders
	FeedStatesByFeedID  dl.FeedStateLoader
	OperatorsByAgencyID dl.OperatorLoader
	// FeedVersionID Loaders
	FeedVersionGtfsImportsByFeedVersionID   dl.FeedVersionGtfsImportLoader
	FeedVersionServiceLevelsByFeedVersionID dl.FeedVersionServiceLevelWhereLoader
	FeedVersionFileInfosByFeedVersionID     dl.FeedVersionFileInfoWhereLoader
	AgenciesByFeedVersionID                 dl.AgencyWhereLoader
	RoutesByFeedVersionID                   dl.RouteWhereLoader
	StopsByFeedVersionID                    dl.StopWhereLoader
	TripsByFeedVersionID                    dl.TripWhereLoader
	FeedInfosByFeedVersionID                dl.FeedInfoWhereLoader
	// Where Loaders
	StopsByParentStopID      dl.StopWhereLoader
	AgencyPlacesByAgencyID   dl.AgencyPlaceWhereLoader
	RouteGeometriesByRouteID dl.RouteGeometryWhereLoader
	TripsByRouteID           dl.TripWhereLoader
	FrequenciesByTripID      dl.FrequencyWhereLoader
	StopTimesByTripID        dl.StopTimeWhereLoader
	StopTimesByStopID        dl.StopTimeWhereLoader
	RouteStopsByRouteID      dl.RouteStopWhereLoader
	RouteStopsByStopID       dl.RouteStopWhereLoader
	RouteHeadwaysByRouteID   dl.RouteHeadwayWhereLoader
	RoutesByAgencyID         dl.RouteWhereLoader
	FeedVersionsByFeedID     dl.FeedVersionWhereLoader
	OperatorsByFeedID        dl.OperatorWhereLoader
	PathwaysByFromStopID     dl.PathwayWhereLoader
	PathwaysByToStopID       dl.PathwayWhereLoader
	CalendarDatesByServiceID dl.CalendarDateWhereLoader
	// Census
	CensusTableByID             dl.CensusTableLoader
	CensusGeographiesByEntityID dl.CensusGeographyWhereLoader
	CensusValuesByGeographyID   dl.CensusValueWhereLoader
}

// Middleware .
func Middleware(atx sqlx.Ext, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loadersKey, &Loaders{
			LevelsByID: *dl.NewLevelLoader(dl.LevelLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.Level, errs []error) {
					MustSelect(
						atx,
						quickSelect("gtfs_levels", nil, nil, ids),
						&ents,
					)
					byid := map[int]*model.Level{}
					for _, ent := range ents {
						byid[ent.ID] = ent
					}
					ents2 := make([]*model.Level, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
			TripsByID: *dl.NewTripLoader(dl.TripLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.Trip, errs []error) {
					MustSelect(
						atx,
						quickSelect("gtfs_trips", nil, nil, ids),
						&ents,
					)
					byid := map[int]*model.Trip{}
					for _, ent := range ents {
						byid[ent.ID] = ent
					}
					ents2 := make([]*model.Trip, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
			CalendarsByID: *dl.NewCalendarLoader(dl.CalendarLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.Calendar, errs []error) {
					MustSelect(
						atx,
						quickSelect("gtfs_calendars", nil, nil, ids),
						&ents,
					)
					byid := map[int]*model.Calendar{}
					for _, ent := range ents {
						byid[ent.ID] = ent
					}
					ents2 := make([]*model.Calendar, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
			ShapesByID: *dl.NewShapeLoader(dl.ShapeLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.Shape, errs []error) {
					MustSelect(
						atx,
						quickSelect("gtfs_shapes", nil, nil, ids),
						&ents,
					)
					byid := map[int]*model.Shape{}
					for _, ent := range ents {
						byid[ent.ID] = ent
					}
					ents2 := make([]*model.Shape, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
			FeedVersionsByID: *dl.NewFeedVersionLoader(dl.FeedVersionLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.FeedVersion, errs []error) {
					MustSelect(
						atx,
						FeedVersionSelect(nil, nil, ids, nil),
						&ents,
					)
					byid := map[int]*model.FeedVersion{}
					for _, ent := range ents {
						byid[ent.ID] = ent
					}
					ents2 := make([]*model.FeedVersion, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
			FeedsByID: *dl.NewFeedLoader(dl.FeedLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.Feed, errs []error) {
					MustSelect(
						atx,
						FeedSelect(nil, nil, ids, nil),
						&ents,
					)
					byid := map[int]*model.Feed{}
					for _, ent := range ents {
						byid[ent.ID] = ent
					}
					ents2 := make([]*model.Feed, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
			AgenciesByID: *dl.NewAgencyLoader(dl.AgencyLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.Agency, errs []error) {
					MustSelect(
						atx,
						AgencySelect(nil, nil, ids, nil),
						&ents,
					)
					byid := map[int]*model.Agency{}
					for _, ent := range ents {
						byid[ent.ID] = ent
					}
					ents2 := make([]*model.Agency, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
			StopsByID: *dl.NewStopLoader(dl.StopLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.Stop, errs []error) {
					MustSelect(
						atx,
						StopSelect(nil, nil, ids, nil),
						&ents,
					)
					byid := map[int]*model.Stop{}
					for _, ent := range ents {
						byid[ent.ID] = ent
					}
					ents2 := make([]*model.Stop, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
			RoutesByID: *dl.NewRouteLoader(dl.RouteLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.Route, errs []error) {
					MustSelect(
						atx,
						RouteSelect(nil, nil, ids, nil),
						&ents,
					)
					byid := map[int]*model.Route{}
					for _, ent := range ents {
						byid[ent.ID] = ent
					}
					ents2 := make([]*model.Route, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
			// Other ID loaders
			FeedVersionGtfsImportsByFeedVersionID: *dl.NewFeedVersionGtfsImportLoader(dl.FeedVersionGtfsImportLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.FeedVersionGtfsImport, errs []error) {
					MustSelect(
						atx,
						quickSelect("feed_version_gtfs_imports", nil, nil, nil).Where(sq.Eq{"feed_version_id": ids}),
						&ents,
					)
					byid := map[int]*model.FeedVersionGtfsImport{}
					for _, ent := range ents {
						byid[ent.FeedVersionID] = ent
					}
					ents2 := make([]*model.FeedVersionGtfsImport, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
			FeedStatesByFeedID: *dl.NewFeedStateLoader(dl.FeedStateLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.FeedState, errs []error) {
					MustSelect(
						atx,
						quickSelect("feed_states", nil, nil, nil).Where(sq.Eq{"feed_id": ids}),
						&ents,
					)
					byid := map[int]*model.FeedState{}
					for _, ent := range ents {
						byid[ent.FeedID] = ent
					}
					ents2 := make([]*model.FeedState, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
			OperatorsByAgencyID: *dl.NewOperatorLoader(dl.OperatorLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.Operator, errs []error) {
					MustSelect(
						atx,
						quickSelect("tl_mv_active_agency_operators", nil, nil, nil).Where(sq.Eq{"feed_id": ids}),
						&ents,
					)
					byid := map[int]*model.Operator{}
					for _, ent := range ents {
						byid[*ent.AgencyID] = ent
					}
					ents2 := make([]*model.Operator, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),

			// Where loaders
			FrequenciesByTripID: *dl.NewFrequencyWhereLoader(dl.FrequencyWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.FrequencyParam) (ents [][]*model.Frequency, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.TripID)
					}
					qents := []*model.Frequency{}
					MustSelect(
						atx,
						lateralWrap(quickSelect("gtfs_frequencies", params[0].Limit, nil, nil), "gtfs_trips", "id", "trip_id", ids),
						&qents,
					)
					group := map[int][]*model.Frequency{}
					for _, ent := range qents {
						group[atoi(ent.TripID)] = append(group[atoi(ent.TripID)], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			StopTimesByTripID: *dl.NewStopTimeWhereLoader(dl.StopTimeWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.StopTimeParam) (ents [][]*model.StopTime, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.TripID)
					}
					qents := []*model.StopTime{}
					MustSelect(
						atx,
						lateralWrap(StopTimeSelect(params[0].Limit, nil, nil, params[0].Where).Where(sq.Eq{"id": ids}), "gtfs_trips", "id", "trip_id", ids),
						&qents,
					)
					group := map[int][]*model.StopTime{}
					for _, ent := range qents {
						group[atoi(ent.TripID)] = append(group[atoi(ent.TripID)], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			StopTimesByStopID: *dl.NewStopTimeWhereLoader(dl.StopTimeWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.StopTimeParam) (ents [][]*model.StopTime, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					qents := []*model.StopTime{}
					for _, p := range params {
						ids = append(ids, p.StopID)
					}
					if p := params[0].Where; p != nil && p.ServiceDate != nil {
						q := StopDeparturesSelect(nil, nil, ids, p)
						qstr, qargs, err := q.ToSql()
						if err != nil {
							panic(err)
						}
						if err := sqlx.Select(atx, &qents, atx.Rebind(qstr), qargs...); err != nil {
							panic(err)
						}
					} else {
						// Otherwise get all stop_times for stop
						MustSelect(
							atx,
							lateralWrap(StopTimeSelect(params[0].Limit, nil, nil, params[0].Where), "gtfs_stops", "id", "stop_id", ids),
							&qents,
						)
					}
					group := map[int][]*model.StopTime{}
					for _, ent := range qents {
						group[atoi(ent.StopID)] = append(group[atoi(ent.StopID)], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			RouteStopsByStopID: *dl.NewRouteStopWhereLoader(dl.RouteStopWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.RouteStopParam) (ents [][]*model.RouteStop, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.StopID)
					}
					qents := []*model.RouteStop{}
					MustSelect(
						atx,
						lateralWrap(quickSelectOrder("tl_route_stops", params[0].Limit, nil, nil, "stop_id"), "gtfs_stops", "id", "stop_id", ids),
						&qents,
					)
					group := map[int][]*model.RouteStop{}
					for _, ent := range qents {
						group[ent.StopID] = append(group[ent.StopID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			RouteStopsByRouteID: *dl.NewRouteStopWhereLoader(dl.RouteStopWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.RouteStopParam) (ents [][]*model.RouteStop, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.RouteID)
					}
					qents := []*model.RouteStop{}
					MustSelect(
						atx,
						lateralWrap(quickSelectOrder("tl_route_stops", params[0].Limit, nil, nil, "stop_id"), "gtfs_routes", "id", "route_id", ids),
						&qents,
					)
					group := map[int][]*model.RouteStop{}
					for _, ent := range qents {
						group[ent.RouteID] = append(group[ent.RouteID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			RouteHeadwaysByRouteID: *dl.NewRouteHeadwayWhereLoader(dl.RouteHeadwayWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.RouteHeadwayParam) (ents [][]*model.RouteHeadway, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.RouteID)
					}
					qents := []*model.RouteHeadway{}
					MustSelect(
						atx,
						lateralWrap(quickSelectOrder("tl_route_headways", params[0].Limit, nil, nil, "route_id"), "gtfs_routes", "id", "route_id", ids),
						&qents,
					)
					group := map[int][]*model.RouteHeadway{}
					for _, ent := range qents {
						group[ent.RouteID] = append(group[ent.RouteID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			FeedVersionFileInfosByFeedVersionID: *dl.NewFeedVersionFileInfoWhereLoader(dl.FeedVersionFileInfoWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.FeedVersionFileInfoParam) (ents [][]*model.FeedVersionFileInfo, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.FeedVersionID)
					}
					qents := []*model.FeedVersionFileInfo{}
					MustSelect(
						atx,
						lateralWrap(quickSelectOrder("feed_version_file_infos", params[0].Limit, nil, nil, "id"), "feed_versions", "id", "feed_version_id", ids),
						&qents,
					)
					group := map[int][]*model.FeedVersionFileInfo{}
					for _, ent := range qents {
						group[ent.FeedVersionID] = append(group[ent.FeedVersionID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			// Has a select method
			StopsByParentStopID: *dl.NewStopWhereLoader(dl.StopWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.StopParam) (ents [][]*model.Stop, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.ParentStopID)
					}
					qents := []*model.Stop{}
					MustSelect(
						atx,
						lateralWrap(StopSelect(params[0].Limit, nil, nil, params[0].Where), "gtfs_stops", "id", "parent_station", ids),
						&qents,
					)
					group := map[int][]*model.Stop{}
					for _, ent := range qents {
						group[ent.ParentStation.Int()] = append(group[ent.ParentStation.Int()], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),

			FeedVersionsByFeedID: *dl.NewFeedVersionWhereLoader(dl.FeedVersionWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.FeedVersionParam) (ents [][]*model.FeedVersion, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.FeedID)
					}
					qents := []*model.FeedVersion{}
					MustSelect(
						atx,
						lateralWrap(FeedVersionSelect(params[0].Limit, nil, nil, params[0].Where), "current_feeds", "id", "feed_id", ids),
						&qents,
					)
					group := map[int][]*model.FeedVersion{}
					for _, ent := range qents {
						group[ent.FeedID] = append(group[ent.FeedID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			AgencyPlacesByAgencyID: *dl.NewAgencyPlaceWhereLoader(dl.AgencyPlaceWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.AgencyPlaceParam) (ents [][]*model.AgencyPlace, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					minRank := 0.0
					for _, p := range params {
						ids = append(ids, p.AgencyID)
						if p.Where != nil && p.Where.MinRank != nil {
							minRank = *p.Where.MinRank
						}
					}
					qents := []*model.AgencyPlace{}
					MustSelect(
						atx,
						lateralWrap(quickSelectOrder("tl_agency_places", params[0].Limit, nil, nil, "agency_id").Where(sq.GtOrEq{"rank": minRank}), "gtfs_agencies", "id", "agency_id", ids),
						&qents,
					)
					group := map[int][]*model.AgencyPlace{}
					for _, ent := range qents {
						group[ent.AgencyID] = append(group[ent.AgencyID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			RouteGeometriesByRouteID: *dl.NewRouteGeometryWhereLoader(dl.RouteGeometryWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.RouteGeometryParam) (ents [][]*model.RouteGeometry, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.RouteID)
					}
					qents := []*model.RouteGeometry{}
					MustSelect(
						atx,
						lateralWrap(quickSelectOrder("tl_route_geometries", params[0].Limit, nil, nil, "route_id"), "gtfs_routes", "id", "route_id", ids),
						&qents,
					)
					group := map[int][]*model.RouteGeometry{}
					for _, ent := range qents {
						group[ent.RouteID] = append(group[ent.RouteID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			TripsByRouteID: *dl.NewTripWhereLoader(dl.TripWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.TripParam) (ents [][]*model.Trip, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.RouteID)
					}
					qents := []*model.Trip{}
					MustSelect(
						atx,
						lateralWrap(TripSelect(params[0].Limit, nil, nil, params[0].Where), "gtfs_routes", "id", "route_id", ids),
						&qents,
					)
					group := map[int][]*model.Trip{}
					for _, ent := range qents {
						group[atoi(ent.RouteID)] = append(group[atoi(ent.RouteID)], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			RoutesByAgencyID: *dl.NewRouteWhereLoader(dl.RouteWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.RouteParam) (ents [][]*model.Route, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.AgencyID)
					}
					qents := []*model.Route{}
					MustSelect(
						atx,
						lateralWrap(RouteSelect(params[0].Limit, nil, nil, params[0].Where), "gtfs_agencies", "id", "agency_id", ids),
						&qents,
					)
					group := map[int][]*model.Route{}
					for _, ent := range qents {
						group[atoi(ent.AgencyID)] = append(group[atoi(ent.AgencyID)], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			AgenciesByFeedVersionID: *dl.NewAgencyWhereLoader(dl.AgencyWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.AgencyParam) (ents [][]*model.Agency, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.FeedVersionID)
					}
					qents := []*model.Agency{}
					MustSelect(
						atx,
						lateralWrap(AgencySelect(params[0].Limit, nil, nil, params[0].Where), "feed_versions", "id", "feed_version_id", ids),
						&qents,
					)
					group := map[int][]*model.Agency{}
					for _, ent := range qents {
						group[ent.FeedVersionID] = append(group[ent.FeedVersionID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			StopsByFeedVersionID: *dl.NewStopWhereLoader(dl.StopWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.StopParam) (ents [][]*model.Stop, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.FeedVersionID)
					}
					qents := []*model.Stop{}
					MustSelect(
						atx,
						lateralWrap(StopSelect(params[0].Limit, nil, nil, params[0].Where), "feed_versions", "id", "feed_version_id", ids),
						&qents,
					)
					group := map[int][]*model.Stop{}
					for _, ent := range qents {
						group[ent.FeedVersionID] = append(group[ent.FeedVersionID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			TripsByFeedVersionID: *dl.NewTripWhereLoader(dl.TripWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.TripParam) (ents [][]*model.Trip, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.FeedVersionID)
					}
					qents := []*model.Trip{}
					MustSelect(
						atx,
						lateralWrap(TripSelect(params[0].Limit, nil, nil, params[0].Where), "feed_versions", "id", "feed_version_id", ids),
						&qents,
					)
					group := map[int][]*model.Trip{}
					for _, ent := range qents {
						group[ent.FeedVersionID] = append(group[ent.FeedVersionID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			FeedInfosByFeedVersionID: *dl.NewFeedInfoWhereLoader(dl.FeedInfoWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.FeedInfoParam) (ents [][]*model.FeedInfo, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.FeedVersionID)
					}
					qents := []*model.FeedInfo{}
					MustSelect(
						atx,
						lateralWrap(quickSelectOrder("gtfs_feed_infos", params[0].Limit, nil, nil, "id"), "feed_versions", "id", "feed_version_id", ids),
						&qents,
					)
					group := map[int][]*model.FeedInfo{}
					for _, ent := range qents {
						group[ent.FeedVersionID] = append(group[ent.FeedVersionID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			RoutesByFeedVersionID: *dl.NewRouteWhereLoader(dl.RouteWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.RouteParam) (ents [][]*model.Route, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.FeedVersionID)
					}
					qents := []*model.Route{}
					MustSelect(
						atx,
						lateralWrap(RouteSelect(params[0].Limit, nil, nil, params[0].Where), "feed_versions", "id", "feed_version_id", ids),
						&qents,
					)
					group := map[int][]*model.Route{}
					for _, ent := range qents {
						group[ent.FeedVersionID] = append(group[ent.FeedVersionID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			FeedVersionServiceLevelsByFeedVersionID: *dl.NewFeedVersionServiceLevelWhereLoader(dl.FeedVersionServiceLevelWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.FeedVersionServiceLevelParam) (ents [][]*model.FeedVersionServiceLevel, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.FeedVersionID)
					}
					qents := []*model.FeedVersionServiceLevel{}
					MustSelect(
						atx,
						lateralWrap(FeedVersionServiceLevelSelect(params[0].Limit, nil, nil, params[0].Where), "feed_versions", "id", "feed_version_id", ids),
						&qents,
					)
					group := map[int][]*model.FeedVersionServiceLevel{}
					for _, ent := range qents {
						group[ent.FeedVersionID] = append(group[ent.FeedVersionID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			OperatorsByFeedID: *dl.NewOperatorWhereLoader(dl.OperatorWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.OperatorParam) (ents [][]*model.Operator, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.FeedID)
					}
					qents := []*model.Operator{}
					MustSelect(
						atx,
						lateralWrap(OperatorSelect(params[0].Limit, nil, nil, params[0].Where), "current_feeds", "id", "feed_id", ids),
						&qents,
					)
					group := map[int][]*model.Operator{}
					for _, ent := range qents {
						if v := ent.FeedID; v != nil {
							group[*v] = append(group[*v], ent)
						}

					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			PathwaysByFromStopID: *dl.NewPathwayWhereLoader(dl.PathwayWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.PathwayParam) (ents [][]*model.Pathway, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.FromStopID)
					}
					qents := []*model.Pathway{}
					MustSelect(
						atx,
						lateralWrap(PathwaySelect(params[0].Limit, nil, nil, params[0].Where), "gtfs_stops", "id", "from_stop_id", ids),
						&qents,
					)
					group := map[int][]*model.Pathway{}
					for _, ent := range qents {
						group[atoi(ent.FromStopID)] = append(group[atoi(ent.FromStopID)], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			PathwaysByToStopID: *dl.NewPathwayWhereLoader(dl.PathwayWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.PathwayParam) (ents [][]*model.Pathway, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.ToStopID)
					}
					qents := []*model.Pathway{}
					MustSelect(
						atx,
						lateralWrap(PathwaySelect(params[0].Limit, nil, nil, params[0].Where), "gtfs_stops", "id", "to_stop_id", ids),
						&qents,
					)
					group := map[int][]*model.Pathway{}
					for _, ent := range qents {
						group[atoi(ent.ToStopID)] = append(group[atoi(ent.ToStopID)], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			CalendarDatesByServiceID: *dl.NewCalendarDateWhereLoader(dl.CalendarDateWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.CalendarDateParam) (ents [][]*model.CalendarDate, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.ServiceID)
					}
					qents := []*model.CalendarDate{}
					MustSelect(
						atx,
						quickSelectOrder("gtfs_calendar_dates", nil, nil, nil, "date").Where(sq.Eq{"service_id": ids}),
						&qents,
					)
					group := map[int][]*model.CalendarDate{}
					for _, ent := range qents {
						group[atoi(ent.ServiceID)] = append(group[atoi(ent.ServiceID)], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			CensusGeographiesByEntityID: *dl.NewCensusGeographyWhereLoader(dl.CensusGeographyWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.CensusGeographyParam) (ents [][]*model.CensusGeography, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.EntityID)
					}
					qents := []*model.CensusGeography{}
					MustSelect(
						atx,
						CensusGeographySelect(&params[0], ids),
						&qents,
					)
					group := map[int][]*model.CensusGeography{}
					for _, ent := range qents {
						group[ent.MatchEntityID] = append(group[ent.MatchEntityID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			CensusValuesByGeographyID: *dl.NewCensusValueWhereLoader(dl.CensusValueWhereLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(params []model.CensusValueParam) (ents [][]*model.CensusValue, errs []error) {
					if len(params) == 0 {
						return nil, nil
					}
					ids := []int{}
					for _, p := range params {
						ids = append(ids, p.GeographyID)
					}
					a := 1000
					params[0].Limit = &a // only a single result allowed
					qents := []*model.CensusValue{}
					MustSelect(
						atx,
						// lateralWrap(CensusValueSelect(&params[0], ids), "tl_census_geographies", "id", "geography_id", ids),
						CensusValueSelect(&params[0], ids),
						&qents,
					)
					group := map[int][]*model.CensusValue{}
					for _, ent := range qents {
						group[ent.GeographyID] = append(group[ent.GeographyID], ent)
					}
					for _, id := range ids {
						ents = append(ents, group[id])
					}
					return ents, nil
				},
			}),
			CensusTableByID: *dl.NewCensusTableLoader(dl.CensusTableLoaderConfig{
				MaxBatch: MAXBATCH,
				Wait:     WAIT,
				Fetch: func(ids []int) (ents []*model.CensusTable, errs []error) {
					MustSelect(
						atx,
						quickSelect("tl_census_tables", nil, nil, ids),
						&ents,
					)
					byid := map[int]*model.CensusTable{}
					for _, ent := range ents {
						byid[ent.ID] = ent
					}
					ents2 := make([]*model.CensusTable, len(ids))
					for i, id := range ids {
						ents2[i] = byid[id]
					}
					return ents2, nil
				},
			}),
		})
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// For .
func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}
