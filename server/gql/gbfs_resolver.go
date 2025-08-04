package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/model"
)

func (r *queryResolver) Bikes(ctx context.Context, limit *int, where *model.GbfsBikeRequest) ([]*model.GbfsFreeBikeStatus, error) {
	return model.ForContext(ctx).GbfsFinder.FindBikes(ctx, checkLimit(limit), where)
}

func (r *queryResolver) Docks(ctx context.Context, limit *int, where *model.GbfsDockRequest) ([]*model.GbfsStationInformation, error) {
	return model.ForContext(ctx).GbfsFinder.FindDocks(ctx, checkLimit(limit), where)
}
