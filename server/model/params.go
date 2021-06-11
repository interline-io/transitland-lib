package model

// This file contains parameters that can be passed to methods for finding/selecting/grouping entities
// These are distinct from WHERE graphql input filters, which are available to users.

type FrequencyParam struct {
	TripID int
	Limit  *int
}

type FeedVersionFileInfoParam struct {
	FeedVersionID int
	Limit         *int
}

type FeedVersionParam struct {
	FeedID int
	Limit  *int
	Where  *FeedVersionFilter
}

type FeedVersionServiceLevelParam struct {
	FeedVersionID int
	Limit         *int
	Where         *FeedVersionServiceLevelFilter
}

type FeedInfoParam struct {
	FeedVersionID int
	Limit         *int
}

type PathwayParam struct {
	FeedVersionID int
	FromStopID    int
	ToStopID      int
	Limit         *int
	Where         *PathwayFilter
}

type StopTimeParam struct {
	TripID int
	StopID int
	Limit  *int
	Where  *StopTimeFilter
}

type AgencyParam struct {
	FeedVersionID int
	Limit         *int
	Where         *AgencyFilter
}

type RouteParam struct {
	AgencyID      int
	FeedVersionID int
	Limit         *int
	Where         *RouteFilter
}

type RouteStopParam struct {
	RouteID int
	StopID  int
	Limit   *int
}

type RouteHeadwayParam struct {
	RouteID int
	Limit   *int
}

type RouteGeometryParam struct {
	RouteID int
	Limit   *int
}

type TripParam struct {
	FeedVersionID int
	RouteID       int
	Limit         *int
	Where         *TripFilter
}

type StopParam struct {
	FeedVersionID int
	ParentStopID  int
	AgencyID      int
	Limit         *int
	Where         *StopFilter
}

type AgencyPlaceParam struct {
	AgencyID int
	Limit    *int
	Where    *AgencyPlaceFilter
}

type OperatorParam struct {
	FeedID int
	Limit  *int
	Where  *OperatorFilter
}

type CalendarDateParam struct {
	ServiceID int
	Limit     *int
	Where     *CalendarDateFilter
}

type CensusGeographyParam struct {
	Radius     *float64
	LayerName  string
	EntityType string
	EntityID   int
	Limit      *int
}

type CensusValueParam struct {
	GeographyID int
	TableNames  string // these have to be comma joined for now, []string cant be used as map key
	Limit       *int
}

type CensusTableParam struct {
	Limit *int
}

type RouteStopBufferParam struct {
	EntityID int
	Radius   *float64
	Limit    *int
}
