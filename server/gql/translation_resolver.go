package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

// TRANSLATION

type translationResolver struct{ *Resolver }

func (r *translationResolver) FeedVersion(ctx context.Context, obj *model.Translation) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

// Translation returns the translations.translation column, which the embedded
// gtfs.Translation struct shadows on model.Translation.
func (r *translationResolver) Translation(ctx context.Context, obj *model.Translation) (*string, error) {
	if !obj.Translation.Translation.Valid {
		return nil, nil
	}
	return &obj.Translation.Translation.Val, nil
}
