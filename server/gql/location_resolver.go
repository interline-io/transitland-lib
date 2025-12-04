package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

type locationResolver struct{ *Resolver }

func (r *locationResolver) FeedVersion(ctx context.Context, obj *model.Location) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *locationResolver) StopTimes(ctx context.Context, obj *model.Location, limit *int, where *model.StopTimeFilter) ([]*model.FlexStopTime, error) {
	return LoaderFor(ctx).FlexStopTimesByStopIDs.Load(ctx, stopTimeLoaderParam{
		FeedVersionID: obj.FeedVersionID,
		StopID:        obj.ID,
		Limit:         resolverCheckLimit(limit),
		Where:         where,
	})()
}
