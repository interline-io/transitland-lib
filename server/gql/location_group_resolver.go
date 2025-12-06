package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

type locationGroupResolver struct{ *Resolver }

func (r *locationGroupResolver) FeedVersion(ctx context.Context, obj *model.LocationGroup) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *locationGroupResolver) Stops(ctx context.Context, obj *model.LocationGroup, limit *int) ([]*model.Stop, error) {
	// TODO: Implement
	return nil, nil
}
