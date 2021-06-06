package resolvers

import (
	"context"

	"github.com/interline-io/transitland-lib/server/find"
	"github.com/interline-io/transitland-lib/server/model"
)

// OPERATOR

type operatorResolver struct{ *Resolver }

func (r *operatorResolver) Agency(ctx context.Context, obj *model.Operator) (*model.Agency, error) {
	if obj.AgencyID != nil {
		return find.For(ctx).AgenciesByID.Load(*obj.AgencyID)
	}
	return nil, nil
}

func (r *operatorResolver) OperatorTags(ctx context.Context, obj *model.Operator) (interface{}, error) {
	return obj.OperatorTags, nil
}

func (r *operatorResolver) OperatorAssociatedFeeds(ctx context.Context, obj *model.Operator) (interface{}, error) {
	return obj.OperatorAssociatedFeeds, nil
}

func (r *operatorResolver) PlacesCache(ctx context.Context, obj *model.Operator) ([]string, error) {
	if obj.PlacesCache != nil {
		return *obj.PlacesCache, nil
	}
	return []string{}, nil
}
