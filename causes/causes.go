package causes

import (
	"fmt"
	"strconv"
)

// TODO
// Errors:
//   FileEmptyError
//   DuplicateColumnError
//   MissingColumnError
//   NoHeaderError
//   InvalidZoneError
//   TripWithoutStopTimes
//
// Warnings:
//   NonuniformAgencyTimezoneError
//   UnusedEntityError
//   StationVisitError
//   StopTooCloseError
//   StopTooFarError
//   NoServiceError
//   FastTravelError
//   UnequalColumnError
//   ShapeReversedError
//   StopTooFarFromShapeError
//
// Maybe validations:
//   route color starts with '#' ?

// Context adds structured context.
type Context struct {
	Filename string
	Line     int
	EntityID string
	Field    string
	Value    string
	Message  string
}

// bc avoids the problem of having a method
// with the same name as the embedded struct.
type bc = Context

// Context returns the base context
func (e *Context) Context() *Context {
	return e
}

func (e *Context) Error() string {
	return ""
}

////////////////////////////
// Feed level errors
////////////////////////////

// SourceUnreadableError reports when the archive itself cannot be read
type SourceUnreadableError struct {
	cause error
	bc
}

// NewSourceUnreadableError returns a new SourceUnreadableError
func NewSourceUnreadableError(message string, err error) *SourceUnreadableError {
	return &SourceUnreadableError{bc: bc{Message: message}, cause: err}
}

func (e *SourceUnreadableError) Error() string {
	return fmt.Sprintf("could not read file '%s': %s", e.Filename, e.cause)
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
	return fmt.Sprintf("required file not present or could not be read: %s", e.Filename)
}

////////////////////////////

// FileNotPresentError is returned when a file is not present
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

// FileParseError reports an error parsing a row
type FileParseError struct {
	cause error
	bc
}

// NewFileParseError returns a new FileParseError
func NewFileParseError(line int, err error) *FileParseError {
	return &FileParseError{bc: bc{Line: line}, cause: err}
}

func (e *FileParseError) Error() string {
	return fmt.Sprintf("could not parse row %d: %s", e.Line, e.cause)
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
	return fmt.Sprintf("file '%s' field '%s' is present more than once", e.Filename, e.Field)
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
	cause error
	bc
}

// NewInvalidFieldError returns a new InvalidFieldError
func NewInvalidFieldError(field string, value string, err error) *InvalidFieldError {
	return &InvalidFieldError{bc: bc{Field: field, Value: value}, cause: err}
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
	return fmt.Sprintf("file contaims more than one timezone")
}

////////////////////////////
// Validation suggestions
////////////////////////////
