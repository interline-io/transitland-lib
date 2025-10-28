package util

import (
	"encoding/json"
	"net/http"
)

func WriteJsonError(w http.ResponseWriter, msg string, statusCode int) {
	a := map[string]string{
		"error": msg,
	}
	jj, _ := json.Marshal(&a)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(jj)
}

// HTTPStatusError interface for errors that have an associated HTTP status code
type HTTPStatusError interface {
	error
	GetStatusCode() int
}

// HTTPError represents an error with an associated HTTP status code
type HTTPError struct {
	Message    string
	StatusCode int
	Cause      error
}

func (e *HTTPError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *HTTPError) Unwrap() error {
	return e.Cause
}

func (e *HTTPError) GetStatusCode() int {
	return e.StatusCode
}

// NewHTTPError creates a new HTTPError
func NewHTTPError(statusCode int, message string, cause error) *HTTPError {
	return &HTTPError{
		Message:    message,
		StatusCode: statusCode,
		Cause:      cause,
	}
}

// Common HTTP error constructors
func NewBadRequestError(message string, cause error) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, message, cause)
}

func NewNotFoundError(message string, cause error) *HTTPError {
	return NewHTTPError(http.StatusNotFound, message, cause)
}

func NewForbiddenError(message string, cause error) *HTTPError {
	return NewHTTPError(http.StatusForbidden, message, cause)
}

func NewInternalServerError(message string, cause error) *HTTPError {
	return NewHTTPError(http.StatusInternalServerError, message, cause)
}

// WriteError writes any error to the response, checking for HTTPStatusError interface
// If the error implements HTTPStatusError, uses its status code, otherwise uses defaultStatusCode
func WriteError(w http.ResponseWriter, err error, defaultStatusCode int) {
	if statusErr, ok := err.(HTTPStatusError); ok {
		WriteJsonError(w, err.Error(), statusErr.GetStatusCode())
	} else {
		WriteJsonError(w, err.Error(), defaultStatusCode)
	}
}
