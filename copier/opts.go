package copier

import (
	"fmt"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/filters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/rules"
)

type CommonExtensions struct {
	// Skip most validation filters
	NoValidators bool
	// Normalize timezones, e.g. US/Pacific -> America/Los_Angeles
	NormalizeTimezones bool
	// Convert extended route types to primitives
	UseBasicRouteTypes bool
	// Simplify shapes
	SimplifyShapes float64
	// Convert route network_id to networks.txt/route_networks.txt
	NormalizeNetworks bool
	// Maximum shape segment length in meters
	ShapeMaxSegmentLength float64
	// Exclude stops and shapes with one or both zero coordinates
	NullIslandCheck bool
}

func (opts *CommonExtensions) Extensions() []any {
	// Default set of validators
	var addExts []any

	// Minimal validators
	if !opts.NoValidators {
		addExts = append(addExts,
			&rules.EntityDuplicateIDCheck{},
			&rules.EntityDuplicateKeyCheck{},
			&rules.ValidFarezoneCheck{},
			&rules.AgencyIDConditionallyRequiredCheck{},
			&rules.StopTimeSequenceCheck{},
			&rules.InconsistentTimezoneCheck{},
			&rules.ParentStationLocationTypeCheck{},
			&rules.CalendarDuplicateDates{},
			&rules.FareProductRiderCategoryDefaultCheck{},
			&rules.TransferStopLocationTypeCheck{},
		)
	}

	// Optional rules that are best practices but can
	// have a significant data quality impact
	if opts.ShapeMaxSegmentLength > 0 {
		// Check shape segment lengths
		addExts = append(addExts, &rules.ShapeMaxSegmentLengthCheck{
			MaxAllowedDistance: opts.ShapeMaxSegmentLength,
		})
	}
	if opts.NullIslandCheck {
		// Exclude stops with zero coordinates
		addExts = append(addExts, &rules.NullIslandCheck{})
	}

	// Optional filters for common data transformations
	if opts.UseBasicRouteTypes {
		// Convert extended route types to basic route types
		addExts = append(addExts, &filters.BasicRouteTypeFilter{})
	}
	if opts.NormalizeTimezones {
		// Normalize timezones and apply agency/stop timezones where empty
		addExts = append(addExts, &filters.NormalizeTimezoneFilter{})
		addExts = append(addExts, &filters.ApplyParentTimezoneFilter{})
	}
	if opts.SimplifyShapes > 0 {
		// Simplify shapes.txt
		addExts = append(addExts, &filters.SimplifyShapeFilter{
			SimplifyValue: opts.SimplifyShapes,
		})
	}
	if opts.NormalizeNetworks {
		// Convert routes.txt network_id to networks.txt/route_networks.txt
		addExts = append(addExts, &filters.RouteNetworkIDFilter{})
	} else {
		addExts = append(addExts, &filters.RouteNetworkIDCompatFilter{})
	}

	return addExts
}

// Options defines the settable options for a Copier.
type Options struct {
	// Batch size
	BatchSize int
	// Skip shape cache
	NoShapeCache bool
	// Attempt to save an entity that returns validation errors
	AllowEntityErrors    bool
	AllowReferenceErrors bool
	// Interpolate any missing StopTime values: ArrivalTime/DepartureTime/ShapeDistTraveled
	InterpolateStopTimes bool
	// Create a stop-to-stop Shape for Trips without a ShapeID.
	CreateMissingShapes bool
	// Create missing Calendar entries
	NormalizeServiceIDs bool
	// Simplify Calendars that use mostly CalendarDates
	SimplifyCalendars bool
	// Copy extra files (requires CSV input)
	CopyExtraFiles bool
	// DeduplicateStopTimes
	DeduplicateJourneyPatterns bool
	// Error limit
	ErrorLimit int
	// Logging level
	Quiet bool
	// Default error handler
	ErrorHandler ErrorHandler
	// Entity selection strategy
	Marker Marker
	// Journey Pattern Key Function
	JourneyPatternKey func(*gtfs.Trip) string
	// Named extensions
	ExtensionDefs []string
	// Common extensions
	CommonExtensions
	// Initialized extensions
	exts []optionExtLevel
}

type optionExtLevel struct {
	ext   any
	level int
}

func (opts *Options) AddExtension(ext any) {
	opts.AddExtensionWithLevel(ext, 0)
}

func ParseExtensionDef(extDef string) (ext.Extension, error) {
	extName, extArgs, err := ext.ParseExtensionArgs(extDef)
	if err != nil {
		return nil, err
	}
	e, err := ext.GetExtension(extName, extArgs)
	if err != nil {
		return nil, fmt.Errorf("error creating extension '%s' with args '%s': %s", extName, extArgs, err.Error())
	} else if e == nil {
		return nil, fmt.Errorf("no registered extension for '%s'", extName)
	}
	return e, nil
}

func (opts *Options) AddExtensionWithLevel(e any, level int) {
	opts.exts = append(opts.exts, optionExtLevel{ext: e, level: level})
}
