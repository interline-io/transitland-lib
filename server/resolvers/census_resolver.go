package resolvers

import (
	"context"
	"strings"

	"github.com/interline-io/transitland-lib/server/find"
	"github.com/interline-io/transitland-lib/server/model"
)

////////////////////////// CENSUS RESOLVERS

type censusGeographyResolver struct{ *Resolver }

func (r *censusGeographyResolver) Values(ctx context.Context, obj *model.CensusGeography, tableNames []string, limit *int) (ents []*model.CensusValue, err error) {
	// dataloader cant easily pass []string
	return find.For(ctx).CensusValuesByGeographyID.Load(model.CensusValueParam{TableNames: strings.Join(tableNames, ","), Limit: limit, GeographyID: obj.ID})
}

type censusValueResolver struct{ *Resolver }

func (r *censusValueResolver) Table(ctx context.Context, obj *model.CensusValue) (*model.CensusTable, error) {
	return find.For(ctx).CensusTableByID.Load(obj.TableID)
}

func (r *censusValueResolver) Values(ctx context.Context, obj *model.CensusValue) (interface{}, error) {
	return obj.TableValues, nil
}

// add geography resolvers to agency, route, stop

func (r *agencyResolver) CensusGeographies(ctx context.Context, obj *model.Agency, layerName string, radius *float64, limit *int) (ents []*model.CensusGeography, err error) {
	return find.For(ctx).CensusGeographiesByEntityID.Load(model.CensusGeographyParam{
		EntityType: "agency",
		EntityID:   obj.ID,
		Radius:     radius,
		LayerName:  layerName,
		Limit:      limit,
	})
}

func (r *routeResolver) CensusGeographies(ctx context.Context, obj *model.Route, layerName string, radius *float64, limit *int) (ents []*model.CensusGeography, err error) {
	return find.For(ctx).CensusGeographiesByEntityID.Load(model.CensusGeographyParam{
		EntityType: "route",
		EntityID:   obj.ID,
		Radius:     radius,
		LayerName:  layerName,
		Limit:      limit,
	})
}

func (r *stopResolver) CensusGeographies(ctx context.Context, obj *model.Stop, layerName string, radius *float64, limit *int) (ents []*model.CensusGeography, err error) {
	return find.For(ctx).CensusGeographiesByEntityID.Load(model.CensusGeographyParam{
		EntityType: "stop",
		EntityID:   obj.ID,
		Radius:     radius,
		LayerName:  layerName,
		Limit:      limit,
	})
}
