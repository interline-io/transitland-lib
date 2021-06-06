package resolvers

import (
	"context"
	"time"

	"github.com/interline-io/transitland-lib/server/find"
	"github.com/interline-io/transitland-lib/server/model"
)

// FEED

type feedResolver struct{ *Resolver }

func (r *feedResolver) FeedState(ctx context.Context, obj *model.Feed) (*model.FeedState, error) {
	return find.For(ctx).FeedStatesByFeedID.Load(obj.ID)
}

func (r *feedResolver) FeedVersions(ctx context.Context, obj *model.Feed, limit *int, where *model.FeedVersionFilter) ([]*model.FeedVersion, error) {
	return find.For(ctx).FeedVersionsByFeedID.Load(model.FeedVersionParam{
		FeedID: obj.ID,
		Limit:  limit,
		Where:  where,
	})
}

func (r *feedResolver) License(ctx context.Context, obj *model.Feed) (*model.FeedLicense, error) {
	return &model.FeedLicense{FeedLicense: obj.License}, nil
}

func (r *feedResolver) Languages(ctx context.Context, obj *model.Feed) ([]string, error) {
	return obj.Languages, nil
}

func (r *feedResolver) AssociatedFeeds(ctx context.Context, obj *model.Feed) ([]string, error) {
	return obj.AssociatedFeeds, nil
}

func (r *feedResolver) Urls(ctx context.Context, obj *model.Feed) (*model.FeedUrls, error) {
	return &model.FeedUrls{FeedUrls: obj.URLs}, nil
}

func (r *feedResolver) AssociatedOperators(ctx context.Context, obj *model.Feed) ([]*model.Operator, error) {
	return find.For(ctx).OperatorsByFeedID.Load(model.OperatorParam{FeedID: obj.ID})
}

func (r *feedResolver) Authorization(ctx context.Context, obj *model.Feed) (*model.FeedAuthorization, error) {
	return &model.FeedAuthorization{FeedAuthorization: obj.Authorization}, nil
}

// FEED STATE

type feedStateResolver struct{ *Resolver }

func (r *feedStateResolver) LastFetchedAt(ctx context.Context, obj *model.FeedState) (*time.Time, error) {
	// TODO: Add Custom Marshaler
	if obj.LastFetchedAt.Valid {
		return &obj.LastFetchedAt.Time, nil
	}
	return nil, nil
}

func (r *feedStateResolver) LastSuccessfulFetchAt(ctx context.Context, obj *model.FeedState) (*time.Time, error) {
	// TODO: Add Custom Marshaler
	if obj.LastSuccessfulFetchAt.Valid {
		return &obj.LastSuccessfulFetchAt.Time, nil
	}
	return nil, nil
}
