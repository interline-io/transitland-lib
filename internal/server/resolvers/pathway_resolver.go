package resolvers

import (
	"context"

	"github.com/interline-io/transitland-lib/internal/server/find"
	"github.com/interline-io/transitland-lib/internal/server/model"
)

// PATHWAYS

type pathwayResolver struct{ *Resolver }

func (r *pathwayResolver) FromStop(ctx context.Context, obj *model.Pathway) (*model.Stop, error) {
	return find.For(ctx).StopsByID.Load(atoi(obj.FromStopID))
}

func (r *pathwayResolver) ToStop(ctx context.Context, obj *model.Pathway) (*model.Stop, error) {
	return find.For(ctx).StopsByID.Load(atoi(obj.ToStopID))
}
