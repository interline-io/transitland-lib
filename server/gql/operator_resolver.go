package gql

import (
	"context"
	"encoding/json"

	"github.com/interline-io/transitland-lib/server/model"
)

// OPERATOR

type operatorResolver struct{ *Resolver }

func (r *operatorResolver) Cursor(ctx context.Context, obj *model.Operator) (*model.Cursor, error) {
	c := model.NewCursor(0, obj.ID)
	return &c, nil
}

func (r *operatorResolver) Agencies(ctx context.Context, obj *model.Operator) ([]*model.Agency, error) {
	return LoaderFor(ctx).AgenciesByOnestopIDs.Load(ctx, agencyLoaderParam{OnestopID: &obj.OnestopID.Val})()
}

func (r *operatorResolver) AssociatedFeeds(ctx context.Context, obj *model.Operator) (interface{}, error) {
	a, err := json.Marshal(obj.AssociatedFeeds)
	return json.RawMessage(a), err
}

func (r *operatorResolver) Generated(ctx context.Context, obj *model.Operator) (bool, error) {
	if obj.Generated {
		return true, nil
	}
	return false, nil
}

func (r *operatorResolver) Feeds(ctx context.Context, obj *model.Operator, limit *int, where *model.FeedFilter) ([]*model.Feed, error) {
	return LoaderFor(ctx).FeedsByOperatorOnestopIDs.Load(ctx, feedLoaderParam{OperatorOnestopID: obj.OnestopID.Val, Where: where, Limit: resolverCheckLimit(limit)})()
}
