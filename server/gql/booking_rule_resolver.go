package gql

import (
	"context"

	"github.com/interline-io/transitland-lib/server/model"
)

type bookingRuleResolver struct{ *Resolver }

func (r *bookingRuleResolver) FeedVersion(ctx context.Context, obj *model.BookingRule) (*model.FeedVersion, error) {
	return LoaderFor(ctx).FeedVersionsByIDs.Load(ctx, obj.FeedVersionID)()
}

func (r *bookingRuleResolver) PriorNoticeServiceID(ctx context.Context, obj *model.BookingRule) (*string, error) {
	return obj.PriorNoticeServiceID.Ptr(), nil
}

func (r *bookingRuleResolver) PriorNoticeService(ctx context.Context, obj *model.BookingRule) (*model.Calendar, error) {
	// TODO: Implement loader for Calendar by ServiceID
	return nil, nil
}
