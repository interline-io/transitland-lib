package causes

import (
	"fmt"
	"strconv"
)

// TODO
//
// Causes that are implemented and include some tests:
//   FileRequiredError
//   FileDuplicateFieldError
//   DuplicateIDError
//   FieldParseError
//   RequiredFieldError
//   ConditionallyRequiredFieldError
//   InvalidFieldError
//   InvalidReferenceError
//   SequenceError
//   InvalidParentStationError
//   InvalidFarezoneError
//   EmptyTripError
//   InconsistentTimezoneError
//
// Causes that are implemented but not fully tested:
//   FileRequiredFieldError
//   SourceUnreadableError
//   FileNotPresentError
//   RowParseError
//   FileUnreadableError
//
// Causes that exist but are not implemented:
//   UnusedEntityError - warning that an named entity (agency, route, stop, calendar, etc.) is not referenced
//
// Best Practice Warnings implemented:
//   NoScheduledServiceError - all days are 0, no type 1 exceptions
//   StopTooFarError - stop too far from a related stop
//   StopTooFarFromShapeError - stop_time too far from associated shape
//   StopTooCloseError - stop too close to another stop
//   FastTravelError - what it says
//
// Best practice warnings that are implemented as warnings but need named types:
//   Duplicate fare_rules
//   Duplicate route route_short_name, route_long_name, etc. combinations
//   NoServiceError - warning that service contains no positive days
//   OverlappingFrequencyError
//
// Best Practice Warning that do not exist but should:
//   StationVisitError - stop_time visits location_type != 0
//   ShapeReversedError - shape is valid and stops are close but reverse direction
//   StopTimeSequenceConsecutive - >2 visits to the same stop in a row, or time not increasing
//   InsufficientRouteColorContrast - colors not distinguishable
//   TravelDistanceError - shape_dist_travel and shape length mismatch, or related
//   FeedServiceDurationError - feed covers < 30 days (note: can only be checked after all others)
//   InconsistentLanguageError - feed_info or agency contains more than 1 language
//   OverlappingBlockError - maybe?
//   Some feed_info rules

// Context adds structured context.
type Context struct {
	Filename   string
	Line       int
	EntityID   string
	Field      string
	Value      string
	Message    string
	errorLevel int
	cause      error
}

// bc avoids the problem of having a method
// with the same name as the embedded struct.
type bc = Context

// Context returns the base context
func (e *Context) Context() *Context {
	return e
}

// ErrorLevel 0 = error, 1 = warning, 2 = best practice
func (e *Context) ErrorLevel() int {
	return e.errorLevel
}

// ErrorLevel returns the error level

// Cause returns the underlying error and implements the Causer interface.
func (e *Context) Cause() error {
	return e.cause
}

// Update sets new values, if present
func (e *Context) Update(v *Context) {
	if v == nil {
		return
	}
	if v.Filename != "" {
		e.Filename = v.Filename
	}
	if v.Line > 0 {
		e.Line = v.Line
	}
	if v.EntityID != "" {
		e.EntityID = v.EntityID
	}
	if v.Field != "" {
		e.Field = v.Field
	}
	if v.Value != "" {
		e.Value = v.Value
	}
	if v.Message != "" {
		e.Message = v.Message
	}
	if v.cause != nil {
		e.cause = v.cause
	}
}

func (e *Context) Error() string {
	return ""
}

////////////////////////////
// Error wrappers
////////////////////////////

// Warning wraps an error.
type Warning struct {
	cause error
}

// ErrorLevel returns type 1.
func (Warning) ErrorLevel() int {
	return 1
}

// BestPractice wraps an error.
type BestPractice struct {
	cause error
}

// ErrorLevel returns type 2.
func (BestPractice) ErrorLevel() int {
	return 2
}

////////////////////////////
// Feed level errors
////////////////////////////

// SourceUnreadableError reports when the archive itself cannot be read
type SourceUnreadableError struct {
	bc
}

// NewSourceUnreadableError returns a new SourceUnreadableError
func NewSourceUnreadableError(message string, err error) *SourceUnreadableError {
	return &SourceUnreadableError{bc: bc{Message: message, cause: err}}
}

func (e *SourceUnreadableError) Error() string {
	return fmt.Sprintf("could not read file '%s'", e.Filename)
}

////////////////////////////

// FileRequiredError reports a required file is not present
type FileRequiredError struct {
	bc
}

// NewFileRequiredError returns a new FileRequiredError
func NewFileRequiredError(filename string) *FileRequiredError {
	return &FileRequiredError{bc: bc{Filename: filename}}
}

func (e *FileRequiredError) Error() string {
	return fmt.Sprintf("required file '%s' not present or could not be read", e.Filename)
}

////////////////////////////

// FileNotPresentError is returned when a requested file is not present
type FileNotPresentError struct {
	bc
}

// NewFileNotPresentError returns a new FileNotPresentError
func NewFileNotPresentError(filename string) *FileNotPresentError {
	return &FileNotPresentError{bc: bc{Filename: filename}}
}

func (e *FileNotPresentError) Error() string {
	return fmt.Sprintf("file '%s' not present", e.Filename)
}

////////////////////////////

// RowParseError reports an error parsing a CSV row
type RowParseError struct {
	bc
}

// NewRowParseError returns a new RowParseError
func NewRowParseError(line int, err error) *RowParseError {
	return &RowParseError{bc: bc{Line: line, cause: err}}
}

func (e *RowParseError) Error() string {
	return fmt.Sprintf("could not parse row %d", e.Line)
}

////////////////////////////

// FileUnreadableError reports a file that could not be read
type FileUnreadableError struct {
	bc
}

// NewFileUnreadableError returns a new FileUnreadableError
func NewFileUnreadableError(filename string, err error) *FileUnreadableError {
	return &FileUnreadableError{bc: bc{Filename: filename, cause: err}}
}

func (e *FileUnreadableError) Error() string {
	return fmt.Sprintf("could not read file '%s'", e.Filename)
}

////////////////////////////

// FileDuplicateFieldError reports when a file contains multiple columns with the same name
type FileDuplicateFieldError struct {
	bc
}

// NewFileDuplicateFieldError returns a new DuplicateFieldError
func NewFileDuplicateFieldError(filename string, field string) *FileDuplicateFieldError {
	return &FileDuplicateFieldError{bc: bc{Filename: filename, Field: field}}
}

func (e *FileDuplicateFieldError) Error() string {
	return fmt.Sprintf("file '%s' field '%s' is present more than once", e.Filename, e.Field)
}

////////////////////////////

// FileRequiredFieldError reports when a file does not have a required column
type FileRequiredFieldError struct {
	bc
}

// NewFileRequiredFieldError returns a new FileRequiredFieldError
func NewFileRequiredFieldError(filename string, field string) *FileRequiredFieldError {
	return &FileRequiredFieldError{bc: bc{Filename: filename, Field: field}}
}

func (e *FileRequiredFieldError) Error() string {
	return fmt.Sprintf("file '%s' required field '%s' not in header", e.Filename, e.Field)
}

////////////////////////////

// DuplicateIDError reports when a unique ID is used more than once in a file.
type DuplicateIDError struct {
	bc
}

// NewDuplicateIDError returns a new DuplicateIDErrror
func NewDuplicateIDError(eid string) *DuplicateIDError {
	return &DuplicateIDError{bc: bc{EntityID: eid}}
}

func (e *DuplicateIDError) Error() string {
	return fmt.Sprintf("unique identifier '%s' is present more than once", e.EntityID)
}

////////////////////////////

// UnusedEntityError reports when an entity is present but not referenced.
type UnusedEntityError struct {
	bc
}

// NewUnusedEntityError returns a new UnusedEntityError
func NewUnusedEntityError(eid string) *UnusedEntityError {
	return &UnusedEntityError{bc: bc{EntityID: eid}}
}

func (e *UnusedEntityError) Error() string {
	return fmt.Sprintf("entity '%s' exists but is not referenced", e.EntityID)
}

////////////////////////////

////////////////////////////
// Entity level errors
////////////////////////////

// Loading Errors

// FieldParseError reports a value that cannot be parsed
type FieldParseError struct {
	bc
}

// NewFieldParseError returns a new FieldParseError
func NewFieldParseError(field string, value string) *FieldParseError {
	return &FieldParseError{bc: bc{Field: field, Value: value}}
}

func (e *FieldParseError) Error() string {
	return fmt.Sprintf("cannot parse value for field %s: '%s'", e.Field, e.Value)
}

////////////////////////////

// Value Errors

// RequiredFieldError reports a required field does not have a value
type RequiredFieldError struct {
	bc
}

// NewRequiredFieldError returns a new RequiredFieldError
func NewRequiredFieldError(field string) *RequiredFieldError {
	return &RequiredFieldError{bc: bc{Field: field}}
}

func (e *RequiredFieldError) Error() string {
	return fmt.Sprintf("no value for required field %s", e.Field)
}

// Value Errors

// ConditionallyRequiredFieldError reports an empty, conditionally required field.
type ConditionallyRequiredFieldError struct {
	bc
}

// NewConditionallyRequiredFieldError returns a new ConditionallyRequiredFieldError
func NewConditionallyRequiredFieldError(field string) *ConditionallyRequiredFieldError {
	return &ConditionallyRequiredFieldError{bc: bc{Field: field}}
}

func (e *ConditionallyRequiredFieldError) Error() string {
	return fmt.Sprintf("no value for required field %s", e.Field)
}

////////////////////////////

// InvalidFieldError reports an invalid value for a field
type InvalidFieldError struct {
	bc
}

// NewInvalidFieldError returns a new InvalidFieldError
func NewInvalidFieldError(field string, value string, err error) *InvalidFieldError {
	return &InvalidFieldError{bc: bc{Field: field, Value: value, cause: err}}
}

func (e *InvalidFieldError) Error() string {
	return fmt.Sprintf("invalid value for field %s: '%s'", e.Field, e.Value)
}

////////////////////////////
// Reference level errors
////////////////////////////

// InvalidReferenceError reports when an entity makes an invalid reference
type InvalidReferenceError struct {
	bc
}

// NewInvalidReferenceError returns a new InvalidReferenceError
func NewInvalidReferenceError(field string, eid string) *InvalidReferenceError {
	return &InvalidReferenceError{bc: bc{Field: field, Value: eid}}
}

func (e *InvalidReferenceError) Error() string {
	return fmt.Sprintf("reference to unknown entity: %s '%s'", e.Field, e.Value)
}

////////////////////////////

// SequenceError reports an invalid shapes.txt or stop_times.txt sequence
type SequenceError struct {
	bc
}

// NewSequenceError returns a new SequenceError
func NewSequenceError(field string, value string) *SequenceError {
	return &SequenceError{bc: bc{Value: value, Field: field}}
}

func (e *SequenceError) Error() string {
	return fmt.Sprintf("invalid sequence in field %s: %s", e.Field, e.Value)
}

////////////////////////////

// InvalidParentStationError reports when a parent station is not location_type = 1.
type InvalidParentStationError struct {
	bc
}

// NewInvalidParentStationError returns a new InvalidParentStationError
func NewInvalidParentStationError(value string) *InvalidParentStationError {
	return &InvalidParentStationError{bc: bc{Value: value}}
}

func (e *InvalidParentStationError) Error() string {
	return fmt.Sprintf("parent_station '%s' is missing or has invalid location_type", e.Value)
}

// InvalidFarezoneError reports when a farezone does not exist.
type InvalidFarezoneError struct {
	bc
}

// NewInvalidFarezoneError returns a new InvalidFarezoneError
func NewInvalidFarezoneError(field string, value string) *InvalidFarezoneError {
	return &InvalidFarezoneError{bc: bc{Field: field, Value: value}}
}

func (e *InvalidFarezoneError) Error() string {
	return fmt.Sprintf("%s farezone '%s' is not present on any stops", e.Field, e.Value)
}

//////////////////////////////

// EmptyTripError reports when a trip has one or zero stop times.
type EmptyTripError struct {
	bc
}

// NewEmptyTripError returns a new EmptyTripError
func NewEmptyTripError(length int) *EmptyTripError {
	return &EmptyTripError{bc: bc{Value: strconv.Itoa(length)}}
}

func (e *EmptyTripError) Error() string {
	return fmt.Sprintf("trip does not have at least 2 stop_times, has: %s", e.Value)
}

//////////////////////////////

////////////////////////////
// Validation warnings
////////////////////////////

// ValidationWarning reports warning messages or informational messages.
type ValidationWarning struct {
	bc
}

// NewValidationWarning returns a new ValidationWarning
func NewValidationWarning(field string, message string) *ValidationWarning {
	return &ValidationWarning{bc: bc{Message: message, Field: field}}
}

// ErrorLevel .
func (e *ValidationWarning) ErrorLevel() int {
	return 1
}

func (e *ValidationWarning) Error() string {
	return fmt.Sprintf("validation warning: %s", e.Message)
}

///////////////

// InconsistentTimezoneError reports when agency.txt has more than 1 timezone present.
type InconsistentTimezoneError struct {
	bc
}

// NewInconsistentTimezoneError returns a new InconsistentTimezoneError.
func NewInconsistentTimezoneError(value string) *InconsistentTimezoneError {
	return &InconsistentTimezoneError{bc: bc{Value: value}}
}

func (e *InconsistentTimezoneError) Error() string {
	return "file contains inconsistent timezones"
}

////////////////////////////
// Best Practices
////////////////////////////

// NoScheduledServiceError reports when a service entry contains no active days.
type NoScheduledServiceError struct{ bc }

func (e *NoScheduledServiceError) Error() string {
	return "service contains no active days"
}

///////////////

// StopTooFarError .
type StopTooFarError struct{ bc }

// NewStopTooFarError .
func NewStopTooFarError() *StopTooFarError {
	return &StopTooFarError{}
}

func (e *StopTooFarError) Error() string {
	return "stop too far from parent"
}

///////////////

// StopTooCloseError .
type StopTooCloseError struct {
	Target   string
	Distance float64
	bc
}

// NewStopTooCloseError .
func NewStopTooCloseError(target string, distance float64) *StopTooCloseError {
	return &StopTooCloseError{Target: target, Distance: distance}
}

func (e *StopTooCloseError) Error() string {
	return fmt.Sprintf("stop is too close to another stop '%s' at %0.2f m", e.Target, e.Distance)
}

///////////////

// StopTooFarFromShapeError reports when a stop is too far from a shape.
type StopTooFarFromShapeError struct {
	StopID   string
	ShapeID  string
	Distance float64
	bc
}

// NewStopTooFarFromShapeError .
func NewStopTooFarFromShapeError(stopid string, shapeid string, distance float64) *StopTooFarFromShapeError {
	return &StopTooFarFromShapeError{
		StopID:   stopid,
		ShapeID:  shapeid,
		Distance: distance,
	}
}

func (e *StopTooFarFromShapeError) Error() string {
	return fmt.Sprintf("stop '%s' is too far from shape '%s' at %0.2fm", e.StopID, e.ShapeID, e.Distance)
}

///////////////

// FastTravelError reports when reasonable maximum speeds have been exceeded.
type FastTravelError struct {
	FromStopID string
	ToStopID   string
	Distance   float64
	Time       int
	Speed      float64
	SpeedLimit float64
	bc
}

// NewFastTravelError .
func NewFastTravelError(from string, to string, t int, distance float64, speed float64, limit float64) *FastTravelError {
	return &FastTravelError{
		FromStopID: from,
		ToStopID:   to,
		Time:       t,
		Distance:   distance,
		Speed:      speed,
		SpeedLimit: limit,
	}
}

func (e *FastTravelError) Error() string {
	return fmt.Sprintf("traveled from stop '%s' to stop '%s' in %d seconds, a distance of %0.2f m and speed of %0.2f km/h where %0.2f km/h is the assumed maximum for this route type", e.FromStopID, e.ToStopID, e.Time, e.Distance, e.Speed, e.SpeedLimit)
}
