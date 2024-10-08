package gtfs

type EnumValue int32

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
	Agency Agency
	Stop Stop
	Route Route
	Trip Trip
	StopTime StopTime
	Shape Shape
	Service Service
}

type Agency struct {
	DatabaseEntity
	AgencyID Key
	AgencyName String
	AgencyUrl Url
	AgencyTimezone Timezone
	AgencyLang Language
	AgencyPhone Phone
	AgencyFareUrl Url
	AgencyEmail Email
}

type Stop struct {
	DatabaseEntity
	StopID Key
	StopCode String
	StopName String
	TtsStopName String
	StopDesc String
	StopLat Float
	StopLon Float
	ZoneID String
	StopUrl Url
	LocationType StopLocationType
	ParentStation Reference
	StopTimezone Timezone
	WheelchairBoarding WheelchairAccess
	LevelID Reference
	PlatformCode String
}

type Route struct {
	DatabaseEntity
	RouteID Key
	AgencyID Reference
	RouteShortName String
	RouteLongName String
	RouteDesc String
	RouteType RouteType
	RouteUrl Url
	RouteColor Color
	RouteTextColor Color
	RouteSortOrder Int
	ContinuousPickup PickupAccess
	ContinuousDropOff PickupAccess
	NetworkID String
}

type Trip struct {
	DatabaseEntity
	RouteID Reference
	ServiceID Reference
	TripID Key
	TripHeadsign String
	TripShortName String
	DirectionID TripDirection
	BlockID String
	ShapeID Reference
	WheelchairAccessible WheelchairAccess
	BikesAllowed BikeAccess
}

type StopTime struct {
	DatabaseEntity
	TripID Reference
	ArrivalTime Seconds
	DepartureTime Seconds
	StopID Reference
	StopSequence Int
	StopHeadsign String
	ContinuousPickup PickupAccess
	ContinuousDropOff PickupAccess
	ShapeDistTraveled Float
	Timepoint StopTimepoint
	LocationID Reference
	LocationGroupID Reference
	StartPickupDropOffWindow Seconds
	EndPickupDropOffWindow Seconds
	PickupType PickupAccess
	DropOffType PickupAccess
	PickupBookingRuleID Reference
	DropOffBookingRuleID Reference
}

type Calendar struct {
	DatabaseEntity
	ServiceID Key
	StartDate Date
	EndDate Date
	Monday Bool
	Tuesday Bool
	Wednesday Bool
	Thursday Bool
	Friday Bool
	Saturday Bool
	Sunday Bool
}

type CalendarDate struct {
	DatabaseEntity
	ServiceID Reference
	Date Date
	ExceptionType CalendarExceptionType
}

type FareAttribute struct {
	DatabaseEntity
	FareID Key
	Price Money
	CurrencyType Currency
	PaymentMethod PaymentMethod
	Transfers FareAttributeTransferType
	AgencyID Reference
	TransferDuration Int
}

type FareRule struct {
	DatabaseEntity
	FareID Key
	RouteID Reference
	OriginID Reference
	DestinationID Reference
	ContainsID Reference
}

type Timeframe struct {
	DatabaseEntity
	TimeframeGroupID Key
	StartTime Seconds
	EndTime Seconds
	ServiceID Reference
}

type FareMedia struct {
	DatabaseEntity
	FareMediaID Key
	FareMediaName String
	FareMediaType FareMediaType
}

type FareProduct struct {
	DatabaseEntity
	FareProductID Key
	FareProductName String
	FareMediaID Reference
	Amount Money
	Currency Currency
}

type FareLegRule struct {
	DatabaseEntity
	LegGroupID Key
	NetworkID Reference
	FromAreaID Reference
	ToAreaID Reference
	FromTimeframeGroupID Reference
	ToTimeframeGroupID Reference
	FareProductID Reference
	RuleProirity Int
}

type FareTransferRule struct {
	DatabaseEntity
	FromLegGroupID Reference
	ToLegGroupID Reference
	TransferCount Int
	DurationLimitType DurationLimitType
	FareTransferType FareTransferType
	FareProductID Reference
}

type Area struct {
	DatabaseEntity
	AreaID Key
	AreaName String
}

type StopArea struct {
	DatabaseEntity
	AreaID Reference
	StopID Reference
}

type Network struct {
	DatabaseEntity
	NetworkID Key
	NetworkName String
}

type RouteNetwork struct {
	DatabaseEntity
	NetworkID Reference
	RouteID Reference
}

type Frequency struct {
	DatabaseEntity
	TripID Reference
	StartTime Seconds
	EndTime Seconds
	HeadwaySecs Int
	ExactTime FrequencyExactTime
}

type Transfer struct {
	DatabaseEntity
	FromStopID Reference
	ToStopID Reference
	FromRouteID Reference
	ToRouteID Reference
	FromTripID Reference
	ToTripID Reference
	TransferType TransferType
	MinTransferTime Int
}

type Pathway struct {
	DatabaseEntity
	PathwayID Key
	FromStopID Reference
	ToStopID Reference
	PathwayMode PathwayMode
	IsBidirectional PathwayDirectionality
	Length Float
	TraversalTime Int
	StairCount Int
	MaxSlope Float
	MinWidth Float
	SignpostedAs String
	ReverseSignpostedAs String
}

type Level struct {
	DatabaseEntity
	LevelID Key
	LevelIndex Float
	LevelName String
}

type LocationGroup struct {
	DatabaseEntity
	LocationGroupID Key
	LocationGroupName String
}

type LocationGroupStop struct {
	DatabaseEntity
	LocationGroupID Reference
	StopID Reference
}

type BookingRule struct {
	DatabaseEntity
	BookingRuleID Key
	BookingType BookingRuleType
	PriorNoticeDurationMin Int
	PriorNoticeDurationMax Int
	PriorNoticeLastDay Int
	PriorNoticeLastTime Seconds
	PriorNoticeStartDay Int
	PriorNoticeStartTime Seconds
	PriorNoticeServiceID Reference
	Message String
	PickupMessage String
	DropOffMessage String
	PhoneNumber String
	InfoUrl Url
	BookingUrl Url
}

type Translation struct {
	DatabaseEntity
	TableName String
	FieldName String
	Language Language
	Translation String
	RecordID String
	RecordSubID String
	FieldValue String
}

type FeedInfo struct {
	DatabaseEntity
	FeedPublisherName String
	FeedPublisherUrl Url
	FeedLang Language
	DefaultLang Language
	FeedStartDate Date
	FeedEndDate Date
	FeedContactEmail Email
	FeedContactUrl Url
}

type Attribution struct {
	DatabaseEntity
	AttributionID Key
	AgencyID Reference
	RouteID Reference
	TripID Reference
	OrganizationName String
	IsProducer AttributionRole
	IsOperator AttributionRole
	IsAuthority AttributionRole
	AttributionUrl Url
	AttributionEmail Email
	AttributionPhone Phone
}

type ShapePoint struct {
	ShapeID Key
	ShapePtLat float64
	ShapePtLon float64
	ShapePtSequence int32
	ShapeDistTraveled float64
}

type Service struct {
	DatabaseEntity
	ServiceID Key
	StartDate Date
	EndDate Date
	Added Date
	Removed Date
	Monday Bool
	Tuesday Bool
	Wednesday Bool
	Thursday Bool
	Friday Bool
	Saturday Bool
	Sunday Bool
}

type Shape struct {
	DatabaseEntity
	ShapeID Key
	Geometry LineString
}

type Point struct {
	Lon float64
	Lat float64
}

type LineString struct {
	Stride uint32
	Coordinates float64
}

type DatabaseEntity struct {
	ID int64
	FeedVersionID int64
}

type Date struct {
	Year int32
	Month int32
	Day int32
}

type Timestamp struct { tt.Option[int64] }

type Seconds struct { tt.Option[int64] }

type Key struct { tt.Option[string] }

type Timezone struct { tt.Option[string] }

type Reference struct { tt.Option[string] }

type Url struct { tt.Option[string] }

type Email struct { tt.Option[string] }

type Color struct { tt.Option[string] }

type Money struct {
	Units int64
	Nanos int64
}

type Currency struct { tt.Option[string] }

type Language struct { tt.Option[string] }

type Phone struct { tt.Option[string] }

type Float struct { tt.Option[float64] }

type String struct { tt.Option[string] }

type Int struct { tt.Option[int64] }

type Bool struct { tt.Option[bool] }

type WheelchairAccess struct { tt.Option[WheelchairAccessEnum] }

type BikeAccess struct { tt.Option[BikeAccessEnum] }

type BoardAccess struct { tt.Option[BoardAccessEnum] }

type PickupAccess struct { tt.Option[PickupAccessEnum] }

type StopLocationType struct { tt.Option[StopLocationTypeEnum] }

type RouteType struct { tt.Option[RouteTypeEnum] }

type TripDirection struct { tt.Option[TripDirectionEnum] }

type StopTimepoint struct { tt.Option[StopTimepointEnum] }

type CalendarExceptionType struct { tt.Option[CalendarExceptionTypeEnum] }

type FrequencyExactTime struct { tt.Option[FrequencyExactTimeEnum] }

type TransferType struct { tt.Option[TransferTypeEnum] }

type PathwayDirectionality struct { tt.Option[PathwayDirectionalityEnum] }

type PathwayMode struct { tt.Option[PathwayModeEnum] }

type BookingRuleType struct { tt.Option[BookingRuleTypeEnum] }

type FareMediaType struct { tt.Option[FareMediaTypeEnum] }

type FareTransferType struct { tt.Option[FareTransferTypeEnum] }

type DurationLimitType struct { tt.Option[DurationLimitTypeEnum] }

type FareAttributeTransferType struct { tt.Option[FareAttributeTransferTypeEnum] }

type PaymentMethod struct { tt.Option[PaymentMethodEnum] }

type AttributionRole struct { tt.Option[AttributionRoleEnum] }

