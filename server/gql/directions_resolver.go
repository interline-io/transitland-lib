package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/finders/directions"
	"github.com/interline-io/transitland-lib/model"
)

type directionsResolver struct{ *Resolver }

// Note: where is not a pointer
func (r *directionsResolver) Directions(ctx context.Context, where model.DirectionRequest) (*model.Directions, error) {
	return directions.HandleRequest(ctx, "", where)
}
