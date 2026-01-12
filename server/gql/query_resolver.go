package gql

import (
	"context"
	"errors"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

// query root

type queryResolver struct{ *Resolver }

func (r *queryResolver) Me(ctx context.Context) (*model.Me, error) {
	cfg := model.ForContext(ctx)
	me := model.Me{}
	me.ExternalData = tt.NewMap(map[string]any{})
	if checker := cfg.Checker; checker != nil {
		// Use checker if available
		cm, err := checker.Me(ctx, &authz.MeRequest{})
		if err != nil {
			return nil, err
		}
		me.ID = cm.User.Id
		me.Email = &cm.User.Email
		me.Name = &cm.User.Name
		me.Roles = cm.Roles
		for k, v := range cm.ExternalData {
			me.ExternalData.Val[k] = v
		}
	} else if user := authn.ForContext(ctx); user != nil {
		// Fallback to user context
		um := user.Email()
		un := user.Name()
		me.ID = user.ID()
		me.Name = &un
		me.Email = &um
		me.Roles = user.Roles()
	} else {
		return nil, errors.New("no user")
	}
	return &me, nil
}

func (r *queryResolver) Agencies(ctx context.Context, limit *int, after *int, ids []int, where *model.AgencyFilter) ([]*model.Agency, error) {
	cfg := model.ForContext(ctx)
	ctx = addMetric(ctx, "agencies")
	if where != nil {
		if err := checkGeo(cfg.MaxRadius, where.Near, where.Bbox); err != nil {
			return nil, err
		}
	}
	return cfg.Finder.FindAgencies(ctx, resolverCheckLimit(limit), checkCursor(after), ids, where)
}

func (r *queryResolver) Routes(ctx context.Context, limit *int, after *int, ids []int, where *model.RouteFilter) ([]*model.Route, error) {
	cfg := model.ForContext(ctx)
	ctx = addMetric(ctx, "routes")
	if where != nil {
		if err := checkGeo(cfg.MaxRadius, where.Near, where.Bbox); err != nil {
			return nil, err
		}
	}
	return cfg.Finder.FindRoutes(ctx, resolverCheckLimit(limit), checkCursor(after), ids, where)
}

func (r *queryResolver) Stops(ctx context.Context, limit *int, after *int, ids []int, where *model.StopFilter) ([]*model.Stop, error) {
	cfg := model.ForContext(ctx)
	ctx = addMetric(ctx, "stops")
	if where != nil {
		if err := checkGeo(cfg.MaxRadius, where.Near, where.Bbox); err != nil {
			return nil, err
		}
	}
	return cfg.Finder.FindStops(ctx, resolverCheckLimit(limit), checkCursor(after), ids, where)
}

func (r *queryResolver) Trips(ctx context.Context, limit *int, after *int, ids []int, where *model.TripFilter) ([]*model.Trip, error) {
	cfg := model.ForContext(ctx)
	ctx = addMetric(ctx, "trips")
	return cfg.Finder.FindTrips(ctx, resolverCheckLimit(limit), checkCursor(after), ids, where)
}

func (r *queryResolver) FeedVersions(ctx context.Context, limit *int, after *int, ids []int, where *model.FeedVersionFilter) ([]*model.FeedVersion, error) {
	cfg := model.ForContext(ctx)
	ctx = addMetric(ctx, "feedVersions")
	if where != nil {
		if err := checkGeo(cfg.MaxRadius, where.Near, where.Bbox); err != nil {
			return nil, err
		}
	}
	return cfg.Finder.FindFeedVersions(ctx, resolverCheckLimit(limit), checkCursor(after), ids, where)
}

func (r *queryResolver) Feeds(ctx context.Context, limit *int, after *int, ids []int, where *model.FeedFilter) ([]*model.Feed, error) {
	cfg := model.ForContext(ctx)
	ctx = addMetric(ctx, "feeds")
	if where != nil {
		if err := checkGeo(cfg.MaxRadius, where.Near, where.Bbox); err != nil {
			return nil, err
		}
	}
	return cfg.Finder.FindFeeds(ctx, resolverCheckLimitMax(limit, RESOLVER_FEED_MAXLIMIT), checkCursor(after), ids, where)
}

func (r *queryResolver) Operators(ctx context.Context, limit *int, after *int, ids []int, where *model.OperatorFilter) ([]*model.Operator, error) {
	cfg := model.ForContext(ctx)
	ctx = addMetric(ctx, "operators")
	if where != nil {
		if err := checkGeo(cfg.MaxRadius, where.Near, where.Bbox); err != nil {
			return nil, err
		}
	}
	return cfg.Finder.FindOperators(ctx, resolverCheckLimit(limit), checkCursor(after), ids, where)
}

func (r *queryResolver) Places(ctx context.Context, limit *int, after *int, level *model.PlaceAggregationLevel, where *model.PlaceFilter) ([]*model.Place, error) {
	cfg := model.ForContext(ctx)
	return cfg.Finder.FindPlaces(ctx, resolverCheckLimit(limit), checkCursor(after), nil, level, where)
}

func (r *queryResolver) CensusDatasets(ctx context.Context, limit *int, after *int, ids []int, where *model.CensusDatasetFilter) ([]*model.CensusDataset, error) {
	cfg := model.ForContext(ctx)
	return cfg.Finder.FindCensusDatasets(ctx, resolverCheckLimit(limit), checkCursor(after), nil, where)
}
