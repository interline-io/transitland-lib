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
