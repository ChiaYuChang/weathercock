package errors

import (
	"fmt"
	"net/http"
	"strings"
)

// Error Code
// 000 - 099: General errors
const (
	ECUnknown         = 000
	ECMarshalFailed   = 001
	ECUnmarshalFailed = 002
)

// HTTP 400 - 499: Client errors
const (
	ECBadRequest      = http.StatusBadRequest
	ECUnauthorized    = http.StatusUnauthorized
	ECNoContent       = http.StatusNoContent
	ECTooManyRequests = http.StatusTooManyRequests
)

// HTTP 500 - 599: Server errors
const (
	ECInternalServerError = http.StatusInternalServerError
	ECNotImplemented      = http.StatusNotImplemented
	ECServiceUnavailable  = http.StatusServiceUnavailable
	ECGatewayTimeout      = http.StatusGatewayTimeout
)
const (
	ECWebpageParsingError = iota + 520
	ECPressReleaseCollectorError
	ECValidationError
)

const (
	ECDatabaseError = iota + 550
	ECNoRows
	ECIntegrityConstrainViolation
	ECTransactionRollback
)

type Error struct {
	StatusCode int      `json:"-"`
	Code       int      `json:"code"`
	Message    string   `json:"message"`
	Details    []string `json:"details,omitempty"`
	internal   error
}

var (
	ErrBadRequest                    = NewWithHTTPStatus(http.StatusBadRequest, ECBadRequest, "bad request")
	ErrNoContent                     = NewWithHTTPStatus(http.StatusNoContent, ECNoContent, "no content available")
	ErrValidationFailed              = NewWithHTTPStatus(http.StatusBadRequest, ECValidationError, "validation failed")
	ErrDBError                       = NewWithHTTPStatus(http.StatusInternalServerError, ECDatabaseError, "database error")
	ErrNotFound                      = NewWithHTTPStatus(http.StatusNotFound, ECNoRows, "no record found")
	ErrDBIntegrityConstrainViolation = NewWithHTTPStatus(http.StatusConflict, ECIntegrityConstrainViolation, "integrity constraint violation")
	ErrDBTransactionRollback         = NewWithHTTPStatus(http.StatusInternalServerError, ECTransactionRollback, "transaction rollback error")
)

func NewWithHTTPStatus(status, code int, message string, details ...string) *Error {
	return &Error{
		StatusCode: status,
		Code:       code,
		Message:    message,
		Details:    details,
		internal:   nil,
	}
}

func New(code int, message string, details ...string) *Error {
	return NewWithHTTPStatus(
		http.StatusInternalServerError,
		code,
		message,
		details...,
	)
}

func FromPgError(e *PGErr) *Error {
	if e == nil {
		return nil
	}
	return NewWithHTTPStatus(
		http.StatusInternalServerError,
		ECDatabaseError,
		fmt.Sprintf("[%s][%s] %s", e.Code, e.Severity, e.Message),
		e.Details,
	)

}

func (e *Error) Error() string {
	if e.internal != nil {
		return fmt.Sprintf("[%d] %s (original error: %s)", e.Code, e.Message, e.internal.Error())
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *Error) ErrorWithDetails() string {
	sb := strings.Builder{}
	sb.WriteString("Error: ")
	sb.WriteString(fmt.Sprintf("  - [%d] %s\n", e.Code, e.Message))
	if len(e.Details) > 0 {
		sb.WriteString("  - Details:\n")
		for _, detail := range e.Details {
			sb.WriteString(fmt.Sprintf("    - %s\n", detail))
		}
	}
	if e.internal != nil {
		sb.WriteString("  - Internal Error: ")
		sb.WriteString(e.internal.Error())
	}
	return sb.String()
}

func (e *Error) Clone() *Error {
	return &Error{
		StatusCode: e.StatusCode,
		Code:       e.Code,
		Message:    e.Message,
		Details:    append([]string{}, e.Details...),
		internal:   e.internal,
	}
}

func (e *Error) WithMessage(message string) *Error {
	if e == nil {
		return nil
	}
	e.Message = message
	return e
}

func (e *Error) WithDetails(details ...string) *Error {
	if e == nil {
		return nil
	}
	e.Details = append(e.Details, details...)
	return e
}

func (e *Error) Warp(err error) *Error {
	if e == nil {
		return nil
	}
	if err == nil {
		return e
	}
	e.internal = err
	return e
}

func Wrap(err error, status, code int, message string, details ...string) *Error {
	return NewWithHTTPStatus(status, code, message, details...)
}

func (e *Error) Unwrap() error {
	return e.internal
}

func (e Error) ToHTTPError() *HTTPError {
	return &HTTPError{
		StatusCode: e.StatusCode,
		Message:    e.Message,
		Details:    e.Details,
	}
}
