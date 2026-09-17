package gql

import (
	"context"

	dataloader "github.com/graph-gophers/dataloader/v7"
	"github.com/interline-io/transitland-lib/server/model"
)

type placeResolver struct{ *Resolver }

func (r *placeResolver) Operators(ctx context.Context, obj *model.Place) ([]*model.Operator, error) {
	var ret []*model.Operator
	var thunks []dataloader.Thunk[[]*model.Operator]
	for _, oid := range obj.AgencyIDs.Val {
		thunks = append(thunks, LoaderFor(ctx).OperatorsByAgencyIDs.Load(ctx, int(oid)))
	}
	for _, thunk := range thunks {
		ops, err := thunk()
		if err != nil {
			return nil, err
		}
		ret = append(ret, ops...)
	}
	// By OnestopID
	byOsid := map[string]bool{}
	var retfilt []*model.Operator
	for _, o := range ret {
		if _, ok := byOsid[o.OnestopID.Val]; !ok {
			byOsid[o.OnestopID.Val] = true
			retfilt = append(retfilt, o)
		}
	}
	return retfilt, nil
}

func (r *placeResolver) Count(ctx context.Context, obj *model.Place) (int, error) {
	operators, err := r.Operators(ctx, obj)
	if err != nil {
		return 0, err
	}
	return len(operators), nil
}
