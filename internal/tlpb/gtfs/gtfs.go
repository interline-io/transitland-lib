package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

type WheelchairAccessEnum int32

type BikeAccessEnum int32

type BoardAccessEnum int32

type PickupAccessEnum int32

type StopLocationTypeEnum int32

type RouteTypeEnum int32

type TripDirectionEnum int32

type StopTimepointEnum int32

type CalendarExceptionTypeEnum int32

type FrequencyExactTimeEnum int32

type TransferTypeEnum int32

type PathwayDirectionalityEnum int32

type PathwayModeEnum int32

type BookingRuleTypeEnum int32

type FareMediaTypeEnum int32

type FareTransferTypeEnum int32

type DurationLimitTypeEnum int32

type FareAttributeTransferTypeEnum int32

type PaymentMethodEnum int32

type AttributionRoleEnum int32

type FeedEntity struct {
	Agency   Agency
	Stop     Stop
	Route    Route
	Trip     Trip
	StopTime StopTime
	Shape    Shape
	Service  Service
}

type Agency struct {
	DatabaseEntity
	AgencyID       tt.Key
	AgencyName     tt.String
	AgencyUrl      tt.Url
	AgencyTimezone tt.Timezone
	AgencyLang     tt.Language
	AgencyPhone    tt.Phone
	AgencyFareUrl  tt.Url
	AgencyEmail    tt.Email
}

type Stop struct {
	DatabaseEntity
	StopID             tt.Key
	StopCode           tt.String
	StopName           tt.String
	TtsStopName        tt.String
	StopDesc           tt.String
	StopLat            tt.Float
	StopLon            tt.Float
	ZoneID             tt.String
	StopUrl            tt.Url
	LocationType       StopLocationType
	ParentStation      tt.Reference
	StopTimezone       tt.Timezone
	WheelchairBoarding WheelchairAccess
	LevelID            tt.Reference
	PlatformCode       tt.String
}

type Route struct {
	DatabaseEntity
	RouteID           tt.Key
	AgencyID          tt.Reference
	RouteShortName    tt.String
	RouteLongName     tt.String
	RouteDesc         tt.String
	RouteType         RouteType
	RouteUrl          tt.Url
	RouteColor        tt.Color
	RouteTextColor    tt.Color
	RouteSortOrder    tt.Int
	ContinuousPickup  PickupAccess
	ContinuousDropOff PickupAccess
	NetworkID         tt.String
}

type Trip struct {
	DatabaseEntity
	RouteID              tt.Reference
	ServiceID            tt.Reference
	TripID               tt.Key
	TripHeadsign         tt.String
	TripShortName        tt.String
	DirectionID          TripDirection
	BlockID              tt.String
	ShapeID              tt.Reference
	WheelchairAccessible WheelchairAccess
	BikesAllowed         BikeAccess
}

type StopTime struct {
	DatabaseEntity
	TripID                   tt.Reference
	ArrivalTime              tt.Seconds
	DepartureTime            tt.Seconds
	StopID                   tt.Reference
	StopSequence             tt.Int
	StopHeadsign             tt.String
	ContinuousPickup         PickupAccess
	ContinuousDropOff        PickupAccess
	ShapeDistTraveled        tt.Float
	Timepoint                StopTimepoint
	LocationID               tt.Reference
	LocationGroupID          tt.Reference
	StartPickupDropOffWindow tt.Seconds
	EndPickupDropOffWindow   tt.Seconds
	PickupType               PickupAccess
	DropOffType              PickupAccess
	PickupBookingRuleID      tt.Reference
	DropOffBookingRuleID     tt.Reference
}

type Calendar struct {
	DatabaseEntity
	ServiceID tt.Key
	StartDate tt.Date
	EndDate   tt.Date
	Monday    tt.Bool
	Tuesday   tt.Bool
	Wednesday tt.Bool
	Thursday  tt.Bool
	Friday    tt.Bool
	Saturday  tt.Bool
	Sunday    tt.Bool
}

type CalendarDate struct {
	DatabaseEntity
	ServiceID     tt.Reference
	Date          tt.Date
	ExceptionType CalendarExceptionType
}

type FareAttribute struct {
	DatabaseEntity
	FareID           tt.Key
	Price            Money
	CurrencyType     tt.Currency
	PaymentMethod    PaymentMethod
	Transfers        FareAttributeTransferType
	AgencyID         tt.Reference
	TransferDuration tt.Int
}

type FareRule struct {
	DatabaseEntity
	FareID        tt.Key
	RouteID       tt.Reference
	OriginID      tt.Reference
	DestinationID tt.Reference
	ContainsID    tt.Reference
}

type Timeframe struct {
	DatabaseEntity
	TimeframeGroupID tt.Key
	StartTime        tt.Seconds
	EndTime          tt.Seconds
	ServiceID        tt.Reference
}

type FareMedia struct {
	DatabaseEntity
	FareMediaID   tt.Key
	FareMediaName tt.String
	FareMediaType FareMediaType
}

type FareProduct struct {
	DatabaseEntity
	FareProductID   tt.Key
	FareProductName tt.String
	FareMediaID     tt.Reference
	Amount          Money
	Currency        tt.Currency
}

type FareLegRule struct {
	DatabaseEntity
	LegGroupID           tt.Key
	NetworkID            tt.Reference
	FromAreaID           tt.Reference
	ToAreaID             tt.Reference
	FromTimeframeGroupID tt.Reference
	ToTimeframeGroupID   tt.Reference
	FareProductID        tt.Reference
	RuleProirity         tt.Int
}

type FareTransferRule struct {
	DatabaseEntity
	FromLegGroupID    tt.Reference
	ToLegGroupID      tt.Reference
	TransferCount     tt.Int
	DurationLimitType DurationLimitType
	FareTransferType  FareTransferType
	FareProductID     tt.Reference
}

type Area struct {
	DatabaseEntity
	AreaID   tt.Key
	AreaName tt.String
}

type StopArea struct {
	DatabaseEntity
	AreaID tt.Reference
	StopID tt.Reference
}

type Network struct {
	DatabaseEntity
	NetworkID   tt.Key
	NetworkName tt.String
}

type RouteNetwork struct {
	DatabaseEntity
	NetworkID tt.Reference
	RouteID   tt.Reference
}

type Frequency struct {
	DatabaseEntity
	TripID      tt.Reference
	StartTime   tt.Seconds
	EndTime     tt.Seconds
	HeadwaySecs tt.Int
	ExactTime   FrequencyExactTime
}

type Transfer struct {
	DatabaseEntity
	FromStopID      tt.Reference
	ToStopID        tt.Reference
	FromRouteID     tt.Reference
	ToRouteID       tt.Reference
	FromTripID      tt.Reference
	ToTripID        tt.Reference
	TransferType    TransferType
	MinTransferTime tt.Int
}

type Pathway struct {
	DatabaseEntity
	PathwayID           tt.Key
	FromStopID          tt.Reference
	ToStopID            tt.Reference
	PathwayMode         PathwayMode
	IsBidirectional     PathwayDirectionality
	Length              tt.Float
	TraversalTime       tt.Int
	StairCount          tt.Int
	MaxSlope            tt.Float
	MinWidth            tt.Float
	SignpostedAs        tt.String
	ReverseSignpostedAs tt.String
}

type Level struct {
	DatabaseEntity
	LevelID    tt.Key
	LevelIndex tt.Float
	LevelName  tt.String
}

type LocationGroup struct {
	DatabaseEntity
	LocationGroupID   tt.Key
	LocationGroupName tt.String
}

type LocationGroupStop struct {
	DatabaseEntity
	LocationGroupID tt.Reference
	StopID          tt.Reference
}

type BookingRule struct {
	DatabaseEntity
	BookingRuleID          tt.Key
	BookingType            BookingRuleType
	PriorNoticeDurationMin tt.Int
	PriorNoticeDurationMax tt.Int
	PriorNoticeLastDay     tt.Int
	PriorNoticeLastTime    tt.Seconds
	PriorNoticeStartDay    tt.Int
	PriorNoticeStartTime   tt.Seconds
	PriorNoticeServiceID   tt.Reference
	Message                tt.String
	PickupMessage          tt.String
	DropOffMessage         tt.String
	PhoneNumber            tt.String
	InfoUrl                tt.Url
	BookingUrl             tt.Url
}

type Translation struct {
	DatabaseEntity
	TableName   tt.String
	FieldName   tt.String
	Language    tt.Language
	Translation tt.String
	RecordID    tt.String
	RecordSubID tt.String
	FieldValue  tt.String
}

type FeedInfo struct {
	DatabaseEntity
	FeedPublisherName tt.String
	FeedPublisherUrl  tt.Url
	FeedLang          tt.Language
	DefaultLang       tt.Language
	FeedStartDate     tt.Date
	FeedEndDate       tt.Date
	FeedContactEmail  tt.Email
	FeedContactUrl    tt.Url
}

type Attribution struct {
	DatabaseEntity
	AttributionID    tt.Key
	AgencyID         tt.Reference
	RouteID          tt.Reference
	TripID           tt.Reference
	OrganizationName tt.String
	IsProducer       AttributionRole
	IsOperator       AttributionRole
	IsAuthority      AttributionRole
	AttributionUrl   tt.Url
	AttributionEmail tt.Email
	AttributionPhone tt.Phone
}

type Shape struct {
	ShapeID           tt.Key
	ShapePtLat        float64
	ShapePtLon        float64
	ShapePtSequence   int32
	ShapeDistTraveled float64
}

type Service struct {
	DatabaseEntity
	ServiceID tt.Key
	StartDate tt.Date
	EndDate   tt.Date
	Added     tt.Date
	Removed   tt.Date
	Monday    tt.Bool
	Tuesday   tt.Bool
	Wednesday tt.Bool
	Thursday  tt.Bool
	Friday    tt.Bool
	Saturday  tt.Bool
	Sunday    tt.Bool
}

type ShapeLine struct {
	DatabaseEntity
	ShapeID  tt.Key
	Geometry LineString
}

type Point struct {
	Lon float64
	Lat float64
}

type LineString struct {
	Stride      uint32
	Coordinates float64
}

type DatabaseEntity struct {
	ID            int64
	FeedVersionID int64
}

type Money struct {
	Units int64
	Nanos int64
}

type WheelchairAccess struct {
	tt.Option[WheelchairAccessEnum]
}

type BikeAccess struct{ tt.Option[BikeAccessEnum] }

type BoardAccess struct{ tt.Option[BoardAccessEnum] }

type PickupAccess struct{ tt.Option[PickupAccessEnum] }

type StopLocationType struct {
	tt.Option[StopLocationTypeEnum]
}

type RouteType struct{ tt.Option[RouteTypeEnum] }

type TripDirection struct{ tt.Option[TripDirectionEnum] }

type StopTimepoint struct{ tt.Option[StopTimepointEnum] }

type CalendarExceptionType struct {
	tt.Option[CalendarExceptionTypeEnum]
}

type FrequencyExactTime struct {
	tt.Option[FrequencyExactTimeEnum]
}

type TransferType struct{ tt.Option[TransferTypeEnum] }

type PathwayDirectionality struct {
	tt.Option[PathwayDirectionalityEnum]
}

type PathwayMode struct{ tt.Option[PathwayModeEnum] }

type BookingRuleType struct{ tt.Option[BookingRuleTypeEnum] }

type FareMediaType struct{ tt.Option[FareMediaTypeEnum] }

type FareTransferType struct {
	tt.Option[FareTransferTypeEnum]
}

type DurationLimitType struct {
	tt.Option[DurationLimitTypeEnum]
}

type FareAttributeTransferType struct {
	tt.Option[FareAttributeTransferTypeEnum]
}

type PaymentMethod struct{ tt.Option[PaymentMethodEnum] }

type AttributionRole struct{ tt.Option[AttributionRoleEnum] }
