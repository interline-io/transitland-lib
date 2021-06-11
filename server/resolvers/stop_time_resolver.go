package resolvers

import (
	"context"

	"github.com/interline-io/transitland-lib/server/find"
	"github.com/interline-io/transitland-lib/server/model"
)

// STOP TIME

type stopTimeResolver struct{ *Resolver }

func (r *stopTimeResolver) Stop(ctx context.Context, obj *model.StopTime) (*model.Stop, error) {
	return find.For(ctx).StopsByID.Load(atoi(obj.StopID))
}

func (r *stopTimeResolver) Trip(ctx context.Context, obj *model.StopTime) (*model.Trip, error) {
	return find.For(ctx).TripsByID.Load(atoi(obj.TripID))
}
