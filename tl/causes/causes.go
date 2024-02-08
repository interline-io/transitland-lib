package causes

import (
	"fmt"
	"strconv"
	"time"
)

// Context adds structured context.
type Context struct {
	Filename   string
	Line       int
	ErrorCode  string
	EntityID   string
	Field      string
	Value      string
	Message    string
	EntityJson map[string]any
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
	if v.ErrorCode != "" {
		e.ErrorCode = v.ErrorCode
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
	if v.EntityJson != nil {
		e.EntityJson = v.EntityJson
	}
}

func (e *Context) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("field: %s value: '%s'", e.Field, e.Value)
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
	return &DuplicateIDError{bc: bc{EntityID: eid, Value: eid}}
}

func (e *DuplicateIDError) Error() string {
	return fmt.Sprintf("unique identifier '%s' is present more than once", e.EntityID)
}

////////////////////////////

// DuplicateServiceExceptionError reports when a (service_id,date) value is present more than once.
type DuplicateServiceExceptionError struct {
	ServiceID string
	Date      string
	bc
}

func (e *DuplicateServiceExceptionError) Error() string {
	return fmt.Sprintf("service_id '%s' has more than one exception for date %s", e.ServiceID, e.Date)
}

// NewDuplicateServiceExceptionError returns a new DuplicateServiceExceptionError.
func NewDuplicateServiceExceptionError(service string, date time.Time) *DuplicateServiceExceptionError {
	return &DuplicateServiceExceptionError{
		ServiceID: service,
		Date:      date.Format("20060102"),
	}
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
	return fmt.Sprintf("no value for required field %s %s", e.Field, e.Message)
}

// ConditionallyForbiddenFieldError reports a value that is present but disallowed.
type ConditionallyForbiddenFieldError struct {
	bc
}

func NewConditionallyForbiddenFieldError(field string, value string, msg string) *ConditionallyForbiddenFieldError {
	return &ConditionallyForbiddenFieldError{bc: bc{Field: field, Value: value, Message: msg}}
}

func (e *ConditionallyForbiddenFieldError) Error() string {
	return fmt.Sprintf("field %s is conditionally forbidden: %s", e.Field, e.Message)
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
	if e.cause != nil {
		return fmt.Sprintf("invalid value for field %s '%s': %s", e.Field, e.Value, e.cause.Error())
	}
	return fmt.Sprintf("invalid value for field %s '%s'", e.Field, e.Value)
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

// InvalidTimezoneError reports when a timezone is not valid.
type InvalidTimezoneError struct{ bc }

func (e *InvalidTimezoneError) Error() string {
	return fmt.Sprintf(
		"entity %s field %s: invalid timezone value '%s'",
		e.EntityID,
		e.Field,
		e.Value,
	)
}

// NewInvalidTimezoneError returns a new InvalidTimezoneError.
func NewInvalidTimezoneError(field string, value string) *InvalidTimezoneError {
	return &InvalidTimezoneError{
		bc: bc{
			Field: field,
			Value: value,
		},
	}
}
