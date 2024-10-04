package out

type WheelchairAccess int32

type BikeAccess int32

type BoardAccess int32

type PickupAccess int32

type StopLocationType int32

type RouteType int32

type TripDirection int32

type StopTimepoint int32

type CalendarExceptionType int32

type FrequencyExactTime int32

type TransferType int32

type PathwayDirectionality int32

type PathwayMode int32

type BookingRuleType int32

type FareMediaType int32

type FareTransferType int32

type DurationLimitType int32

type FareAttributeTransferType int32

type PaymentMethod int32

type AttributionRole int32

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
	ID             int64
	FeedVersionID  int64
	AgencyID       EntityID
	AgencyName     string
	AgencyUrl      Url
	AgencyTimezone Timezone
	AgencyLang     Language
	AgencyPhone    Phone
	AgencyFareUrl  Url
	AgencyEmail    Email
}

type Stop struct {
	ID                 int64
	FeedVersionID      int64
	StopID             EntityID
	StopCode           string
	StopName           string
	TtsStopName        string
	StopDesc           string
	StopLat            float64
	StopLon            float64
	ZoneID             string
	StopUrl            Url
	LocationType       StopLocationType
	ParentStation      Reference
	StopTimezone       Timezone
	WheelchairBoarding WheelchairAccess
	LevelID            Reference
	PlatformCode       string
}

type Route struct {
	ID                int64
	FeedVersionID     int64
	RouteID           EntityID
	AgencyID          Reference
	RouteShortName    string
	RouteLongName     string
	RouteDesc         string
	RouteType         RouteType
	RouteUrl          Url
	RouteColor        Color
	RouteTextColor    Color
	RouteSortOrder    int32
	ContinuousPickup  PickupAccess
	ContinuousDropOff PickupAccess
	NetworkID         string
}

type Trip struct {
	ID                   int64
	FeedVersionID        int64
	RouteID              Reference
	ServiceID            Reference
	TripID               EntityID
	TripHeadsign         string
	TripShortName        string
	DirectionID          TripDirection
	BlockID              string
	ShapeID              Reference
	WheelchairAccessible WheelchairAccess
	BikesAllowed         BikeAccess
}

type StopTime struct {
	ID                       int64
	FeedVersionID            int64
	TripID                   Reference
	ArrivalTime              Seconds
	DepartureTime            Seconds
	StopID                   Reference
	StopSequence             int32
	StopHeadsign             string
	ContinuousPickup         PickupAccess
	ContinuousDropOff        PickupAccess
	ShapeDistTraveled        float64
	Timepoint                StopTimepoint
	LocationID               Reference
	LocationGroupID          Reference
	StartPickupDropOffWindow Seconds
	EndPickupDropOffWindow   Seconds
	PickupType               PickupAccess
	DropOffType              PickupAccess
	PickupBookingRuleID      Reference
	DropOffBookingRuleID     Reference
}

type Calendar struct {
	ID            int64
	FeedVersionID int64
	ServiceID     EntityID
	StartDate     Date
	EndDate       Date
	Monday        bool
	Tuesday       bool
	Wednesday     bool
	Thursday      bool
	Friday        bool
	Saturday      bool
	Sunday        bool
}

type CalendarDate struct {
	ID            int64
	FeedVersionID int64
	ServiceID     Reference
	Date          Date
	ExceptionType CalendarExceptionType
}

type FareAttribute struct {
	ID               int64
	FeedVersionID    int64
	FareID           EntityID
	Price            Money
	CurrencyType     Currency
	PaymentMethod    PaymentMethod
	Transfers        FareAttributeTransferType
	AgencyID         Reference
	TransferDuration int32
}

type FareRule struct {
	ID            int64
	FeedVersionID int64
	FareID        EntityID
	RouteID       Reference
	OriginID      Reference
	DestinationID Reference
	ContainsID    Reference
}

type Timeframe struct {
	ID               int64
	FeedVersionID    int64
	TimeframeGroupID EntityID
	StartTime        Seconds
	EndTime          Seconds
	ServiceID        Reference
}

type FareMedia struct {
	ID            int64
	FeedVersionID int64
	FareMediaID   EntityID
	FareMediaName string
	FareMediaType FareMediaType
}

type FareProduct struct {
	ID              int64
	FeedVersionID   int64
	FareProductID   EntityID
	FareProductName string
	FareMediaID     Reference
	Amount          Money
	Currency        Currency
}

type FareLegRule struct {
	ID                   int64
	FeedVersionID        int64
	LegGroupID           EntityID
	NetworkID            Reference
	FromAreaID           Reference
	ToAreaID             Reference
	FromTimeframeGroupID Reference
	ToTimeframeGroupID   Reference
	FareProductID        Reference
	RuleProirity         int32
}

type FareTransferRule struct {
	ID                int64
	FeedVersionID     int64
	FromLegGroupID    Reference
	ToLegGroupID      Reference
	TransferCount     TransferCount
	DurationLimitType DurationLimitType
	FareTransferType  FareTransferType
	FareProductID     Reference
}

type Area struct {
	ID            int64
	FeedVersionID int64
	AreaID        EntityID
	AreaName      string
}

type StopArea struct {
	ID            int64
	FeedVersionID int64
	AreaID        Reference
	StopID        Reference
}

type Network struct {
	ID            int64
	FeedVersionID int64
	NetworkID     EntityID
	NetworkName   string
}

type RouteNetwork struct {
	ID            int64
	FeedVersionID int64
	NetworkID     Reference
	RouteID       Reference
}

type Frequency struct {
	ID            int64
	FeedVersionID int64
	TripID        Reference
	StartTime     Seconds
	EndTime       Seconds
	HeadwaySecs   int32
	ExactTime     FrequencyExactTime
}

type Transfer struct {
	ID              int64
	FeedVersionID   int64
	FromStopID      Reference
	ToStopID        Reference
	FromRouteID     Reference
	ToRouteID       Reference
	FromTripID      Reference
	ToTripID        Reference
	TransferType    TransferType
	MinTransferTime int32
}

type Pathway struct {
	ID                  int64
	FeedVersionID       int64
	PathwayID           EntityID
	FromStopID          Reference
	ToStopID            Reference
	PathwayMode         PathwayMode
	IsBidirectional     PathwayDirectionality
	Length              float64
	TraversalTime       int32
	StairCount          int32
	MaxSlope            float64
	MinWidth            float64
	SignpostedAs        string
	ReverseSignpostedAs string
}

type Level struct {
	ID            int64
	FeedVersionID int64
	LevelID       EntityID
	LevelIndex    float64
	LevelName     string
}

type LocationGroup struct {
	ID                int64
	FeedVersionID     int64
	LocationGroupID   EntityID
	LocationGroupName string
}

type LocationGroupStop struct {
	ID              int64
	FeedVersionID   int64
	LocationGroupID Reference
	StopID          Reference
}

type BookingRule struct {
	ID                     int64
	FeedVersionID          int64
	BookingRuleID          EntityID
	BookingType            BookingRuleType
	PriorNoticeDurationMin int32
	PriorNoticeDurationMax int32
	PriorNoticeLastDay     int32
	PriorNoticeLastTime    Seconds
	PriorNoticeStartDay    int32
	PriorNoticeStartTime   Seconds
	PriorNoticeServiceID   Reference
	Message                string
	PickupMessage          string
	DropOffMessage         string
	PhoneNumber            string
	InfoUrl                Url
	BookingUrl             Url
}

type Translation struct {
	ID            int64
	FeedVersionID int64
	TableName     string
	FieldName     string
	Language      Language
	Translation   string
	RecordID      string
	RecordSubID   string
	FieldValue    string
}

type FeedInfo struct {
	ID                int64
	FeedVersionID     int64
	FeedPublisherName string
	FeedPublisherUrl  Url
	FeedLang          Language
	DefaultLang       Language
	FeedStartDate     Date
	FeedEndDate       Date
	FeedContactEmail  Email
	FeedContactUrl    Url
}

type Attribution struct {
	ID               int64
	FeedVersionID    int64
	AttributionID    EntityID
	AgencyID         Reference
	RouteID          Reference
	TripID           Reference
	OrganizationName string
	IsProducer       AttributionRole
	IsOperator       AttributionRole
	IsAuthority      AttributionRole
	AttributionUrl   Url
	AttributionEmail Email
	AttributionPhone Phone
}

type Service struct {
	ID            int64
	FeedVersionID int64
	ServiceID     EntityID
	StartDate     Date
	EndDate       Date
	Added         Date
	Removed       Date
	Monday        bool
	Tuesday       bool
	Wednesday     bool
	Thursday      bool
	Friday        bool
	Saturday      bool
	Sunday        bool
}

type ShapePoint struct {
	ShapeID           EntityID
	ShapePtLat        float64
	ShapePtLon        float64
	ShapePtSequence   int32
	ShapeDistTraveled float64
}

type Shape struct {
	ShapeID EntityID
	Shape   LineString
}

type Point struct {
	Lon float64
	Lat float64
}

type LineString struct {
	Stride      uint32
	Coordinates float64
}

type Date struct {
	Year  int32
	Month int32
	Day   int32
}

type Time Option[int64]

type Seconds Option[int64]

type EntityID Option[string]

type Timezone Option[string]

type Reference Option[string]

type Url Option[string]

type Email Option[string]

type Color Option[string]

type Money struct {
	Units int64
	Nanos int64
}

type Currency Option[string]

type Language Option[string]

type Phone Option[string]

type FareAttributeTransfer Option[FareAttributeTransferType]

type TransferCount Option[int32]
