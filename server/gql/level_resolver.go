package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

type levelResolver struct {
	*Resolver
}

func (r *levelResolver) Stops(ctx context.Context, obj *model.Level) ([]*model.Stop, error) {
	return LoaderFor(ctx).StopsByLevelIDs.Load(ctx, stopLoaderParam{LevelID: obj.ID})()
}
