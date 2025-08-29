package gql

import (
	"context"

	dataloader "github.com/graph-gophers/dataloader/v7"
	"github.com/interline-io/transitland-lib/server/model"
)

type placeResolver struct{ *Resolver }

func (r *placeResolver) Operators(ctx context.Context, obj *model.Place) ([]*model.Operator, error) {
	var ret []*model.Operator
	var thunks []dataloader.Thunk[*model.Operator]
	for _, oid := range obj.AgencyIDs.Val {
		// fmt.Println("creating thunk for operator:", oid)
		t := LoaderFor(ctx).OperatorsByAgencyIDs.Load(ctx, int(oid))
		thunks = append(thunks, t)
	}
	for i := 0; i < len(obj.AgencyIDs.Val); i++ {
		// oid := obj.AgencyIDs.Val[i]
		o, err := thunks[i]()
		if err != nil {
			return nil, err
		}
		if o != nil {
			// oj, _ := json.Marshal(o)
			// fmt.Println("got operator for:", oid, "json:", string(oj))
			ret = append(ret, o)
		}
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
