package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

type locationGroupStopResolver struct{ *Resolver }

func (r *locationGroupStopResolver) LocationGroup(ctx context.Context, obj *model.LocationGroupStop) (*model.LocationGroup, error) {
	// TODO
	return nil, nil
}

func (r *locationGroupStopResolver) Stop(ctx context.Context, obj *model.LocationGroupStop) (*model.Stop, error) {
	// TODO
	return nil, nil
}
