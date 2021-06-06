//go:generate go run github.com/99designs/gqlgen

package resolvers

import (
	"strconv"

	"github.com/interline-io/transitland-lib/server/config"
	"github.com/interline-io/transitland-lib/server/generated/gqlgen"
)

func atoi(v string) int {
	a, _ := strconv.Atoi(v)
	return a
}

// Resolver .
type Resolver struct {
	cfg config.Config
}

// Query .
func (r *Resolver) Query() gqlgen.QueryResolver { return &queryResolver{r} }

// Mutation .
func (r *Resolver) Mutation() gqlgen.MutationResolver { return &mutationResolver{r} }

// Agency .
func (r *Resolver) Agency() gqlgen.AgencyResolver { return &agencyResolver{r} }

// Feed .
func (r *Resolver) Feed() gqlgen.FeedResolver { return &feedResolver{r} }

// FeedState .
func (r *Resolver) FeedState() gqlgen.FeedStateResolver { return &feedStateResolver{r} }

// FeedVersion .
func (r *Resolver) FeedVersion() gqlgen.FeedVersionResolver { return &feedVersionResolver{r} }

// Route .
func (r *Resolver) Route() gqlgen.RouteResolver { return &routeResolver{r} }

// RouteStop .
func (r *Resolver) RouteStop() gqlgen.RouteStopResolver { return &routeStopResolver{r} }

// RouteHeadway .
func (r *Resolver) RouteHeadway() gqlgen.RouteHeadwayResolver { return &routeHeadwayResolver{r} }

// Stop .
func (r *Resolver) Stop() gqlgen.StopResolver { return &stopResolver{r} }

// Trip .
func (r *Resolver) Trip() gqlgen.TripResolver { return &tripResolver{r} }

// StopTime .
func (r *Resolver) StopTime() gqlgen.StopTimeResolver { return &stopTimeResolver{r} }

// Operator .
func (r *Resolver) Operator() gqlgen.OperatorResolver { return &operatorResolver{r} }

// FeedVersionGtfsImport .
func (r *Resolver) FeedVersionGtfsImport() gqlgen.FeedVersionGtfsImportResolver {
	return &feedVersionGtfsImportResolver{r}
}

// CensusGeography .
func (r *Resolver) CensusGeography() gqlgen.CensusGeographyResolver {
	return &censusGeographyResolver{r}
}

// CensusValue .
func (r *Resolver) CensusValue() gqlgen.CensusValueResolver {
	return &censusValueResolver{r}
}

// Pathway .
func (r *Resolver) Pathway() gqlgen.PathwayResolver {
	return &pathwayResolver{r}
}
