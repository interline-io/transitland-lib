package gql

import (
	"context"
	"strings"

	"github.com/interline-io/transitland-lib/server/model"
)

////////////////////////// CENSUS RESOLVERS

type censusDatasetResolver struct{ *Resolver }

func (r *censusDatasetResolver) Geographies(ctx context.Context, obj *model.CensusDataset, limit *int, where *model.CensusDatasetGeographyFilter) ([]*model.CensusGeography, error) {
	return LoaderFor(ctx).CensusGeographiesByDatasetIDs.Load(ctx, censusDatasetGeographyLoaderParam{DatasetID: obj.ID, Limit: limit, Where: where})()
}

func (r *censusDatasetResolver) Sources(ctx context.Context, obj *model.CensusDataset, limit *int, where *model.CensusSourceFilter) ([]*model.CensusSource, error) {
	return LoaderFor(ctx).CensusSourcesByDatasetIDs.Load(ctx, censusSourceLoaderParam{DatasetID: obj.ID, Limit: limit, Where: where})()
}

func (r *censusDatasetResolver) Tables(ctx context.Context, obj *model.CensusDataset, limit *int, where *model.CensusTableFilter) ([]*model.CensusTable, error) {
	return nil, nil
}

func (r *censusDatasetResolver) Layers(ctx context.Context, obj *model.CensusDataset) (ret []*model.CensusLayer, err error) {
	return LoaderFor(ctx).CensusDatasetLayersByDatasetIDs.Load(ctx, obj.ID)()
}

type censusSourceResolver struct{ *Resolver }

func (r *censusSourceResolver) Layers(ctx context.Context, obj *model.CensusSource) (ret []*model.CensusLayer, err error) {
	return LoaderFor(ctx).CensusSourceLayersBySourceIDs.Load(ctx, obj.ID)()
}

func (r *censusSourceResolver) Geographies(ctx context.Context, obj *model.CensusSource, limit *int, where *model.CensusSourceGeographyFilter) (ret []*model.CensusGeography, err error) {
	return LoaderFor(ctx).CensusGeographiesBySourceIDs.Load(ctx, censusSourceGeographyLoaderParam{SourceID: obj.ID, Limit: limit, Where: where})()
}

type censusGeographyResolver struct{ *Resolver }

func (r *censusGeographyResolver) Values(ctx context.Context, obj *model.CensusGeography, tableNames []string, datasetName *string, limit *int) ([]*model.CensusValue, error) {
	// dataloader cant easily pass []string
	return LoaderFor(ctx).CensusValuesByGeographyIDs.Load(ctx, censusValueLoaderParam{Dataset: datasetName, TableNames: strings.Join(tableNames, ","), Limit: limit, Geoid: *obj.Geoid})()
}

func (r *censusGeographyResolver) Layer(ctx context.Context, obj *model.CensusGeography) (*model.CensusLayer, error) {
	return LoaderFor(ctx).CensusLayersByIDs.Load(ctx, obj.LayerID)()
}

func (r *censusGeographyResolver) Source(ctx context.Context, obj *model.CensusGeography) (*model.CensusSource, error) {
	return LoaderFor(ctx).CensusSourcesByIDs.Load(ctx, obj.SourceID)()
}

type censusValueResolver struct{ *Resolver }

func (r *censusValueResolver) Table(ctx context.Context, obj *model.CensusValue) (*model.CensusTable, error) {
	return LoaderFor(ctx).CensusTableByIDs.Load(ctx, obj.TableID)()
}

type censusTableResolver struct{ *Resolver }

func (r *censusTableResolver) Fields(ctx context.Context, obj *model.CensusTable) ([]*model.CensusField, error) {
	return LoaderFor(ctx).CensusFieldsByTableIDs.Load(ctx, censusFieldLoaderParam{TableID: obj.ID})()
}

type censusLayerResolver struct{ *Resolver }

func (r *censusLayerResolver) Geographies(ctx context.Context, obj *model.CensusLayer, limit *int, where *model.CensusSourceGeographyFilter) (ret []*model.CensusGeography, err error) {
	return LoaderFor(ctx).CensusGeographiesByLayerIDs.Load(ctx, censusSourceGeographyLoaderParam{LayerID: obj.ID, Limit: limit, Where: where})()
}

// add geography resolvers to agency, route, stop

func (r *agencyResolver) CensusGeographies(ctx context.Context, obj *model.Agency, limit *int, where *model.CensusGeographyFilter) ([]*model.CensusGeography, error) {
	return LoaderFor(ctx).CensusGeographiesByEntityIDs.Load(ctx, censusGeographyLoaderParam{
		EntityType: "agency",
		EntityID:   obj.ID,
		Limit:      limit,
		Where:      where,
	})()
}

func (r *routeResolver) CensusGeographies(ctx context.Context, obj *model.Route, limit *int, where *model.CensusGeographyFilter) ([]*model.CensusGeography, error) {
	return LoaderFor(ctx).CensusGeographiesByEntityIDs.Load(ctx, censusGeographyLoaderParam{
		EntityType: "route",
		EntityID:   obj.ID,
		Limit:      limit,
		Where:      where,
	})()
}

func (r *stopResolver) CensusGeographies(ctx context.Context, obj *model.Stop, limit *int, where *model.CensusGeographyFilter) ([]*model.CensusGeography, error) {
	return LoaderFor(ctx).CensusGeographiesByEntityIDs.Load(ctx, censusGeographyLoaderParam{
		EntityType: "stop",
		EntityID:   obj.ID,
		Limit:      limit,
		Where:      where,
	})()
}
