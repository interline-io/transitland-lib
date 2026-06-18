package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

type locationGroupStopResolver struct{ *Resolver }

func (r *locationGroupStopResolver) LocationGroup(ctx context.Context, obj *model.LocationGroupStop) (*model.LocationGroup, error) {
	return LoaderFor(ctx).LocationGroupsByIDs.Load(ctx, obj.LocationGroupID.Int())()
}

func (r *locationGroupStopResolver) Stop(ctx context.Context, obj *model.LocationGroupStop) (*model.Stop, error) {
	return LoaderFor(ctx).StopsByIDs.Load(ctx, obj.StopID.Int())()
}
