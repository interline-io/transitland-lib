package gql

import (
	"context"
	"errors"

	"github.com/interline-io/transitland-lib/model"
)

// Entity editing
func (r *mutationResolver) StopCreate(ctx context.Context, obj model.StopSetInput) (*model.Stop, error) {
	finder := model.ForContext(ctx).Finder
	entId, err := finder.StopCreate(ctx, obj)
	if err != nil {
		return nil, err
	}
	ents, errs := finder.StopsByIDs(ctx, []int{entId})
	return first(errs, ents)
}

func (r *mutationResolver) StopUpdate(ctx context.Context, obj model.StopSetInput) (*model.Stop, error) {
	finder := model.ForContext(ctx).Finder
	entId, err := finder.StopUpdate(ctx, obj)
	if err != nil {
		return nil, err
	}
	ents, errs := finder.StopsByIDs(ctx, []int{entId})
	return first(errs, ents)
}

func (r *mutationResolver) StopDelete(ctx context.Context, id int) (*model.EntityDeleteResult, error) {
	finder := model.ForContext(ctx).Finder
	if err := finder.StopDelete(ctx, id); err != nil {
		return nil, err
	}
	return &model.EntityDeleteResult{ID: id}, nil
}

func (r *mutationResolver) LevelCreate(ctx context.Context, obj model.LevelSetInput) (*model.Level, error) {
	finder := model.ForContext(ctx).Finder
	entId, err := finder.LevelCreate(ctx, obj)
	if err != nil {
		return nil, err
	}
	ents, errs := finder.LevelsByIDs(ctx, []int{entId})
	return first(errs, ents)
}

func (r *mutationResolver) LevelUpdate(ctx context.Context, obj model.LevelSetInput) (*model.Level, error) {
	finder := model.ForContext(ctx).Finder
	entId, err := finder.LevelUpdate(ctx, obj)
	if err != nil {
		return nil, err
	}
	ents, errs := finder.LevelsByIDs(ctx, []int{entId})
	return first(errs, ents)
}

func (r *mutationResolver) LevelDelete(ctx context.Context, id int) (*model.EntityDeleteResult, error) {
	finder := model.ForContext(ctx).Finder
	if err := finder.LevelDelete(ctx, id); err != nil {
		return nil, err
	}
	return &model.EntityDeleteResult{ID: id}, nil
}

func (r *mutationResolver) PathwayCreate(ctx context.Context, obj model.PathwaySetInput) (*model.Pathway, error) {
	finder := model.ForContext(ctx).Finder
	entId, err := finder.PathwayCreate(ctx, obj)
	if err != nil {
		return nil, err
	}
	ents, errs := finder.PathwaysByIDs(ctx, []int{entId})
	return first(errs, ents)
}

func (r *mutationResolver) PathwayUpdate(ctx context.Context, obj model.PathwaySetInput) (*model.Pathway, error) {
	finder := model.ForContext(ctx).Finder
	entId, err := finder.PathwayUpdate(ctx, obj)
	if err != nil {
		return nil, err
	}
	ents, errs := finder.PathwaysByIDs(ctx, []int{entId})
	return first(errs, ents)
}

func (r *mutationResolver) PathwayDelete(ctx context.Context, id int) (*model.EntityDeleteResult, error) {
	finder := model.ForContext(ctx).Finder
	if err := finder.PathwayDelete(ctx, id); err != nil {
		return nil, err
	}
	return &model.EntityDeleteResult{ID: id}, nil
}

func first[T any](errs []error, v []T) (T, error) {
	var ret T
	if len(errs) > 0 {
		return ret, errs[0]
	}
	if len(v) == 0 {
		return ret, errors.New("not found")
	}
	return v[0], nil
}
