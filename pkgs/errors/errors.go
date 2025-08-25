package errors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Error Code
// 000 - 099: General errors
const (
	ECUnknown         = 000
	ECMarshalFailed   = 001
	ECUnmarshalFailed = 002
	ECIOError         = 003
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
	ECNATSJsPublishFailed
)

const (
	ECDatabaseError = iota + 550
	ECNoRows
	ECIntegrityConstrainViolation
	ECTransactionRollback
	ECDatabaseTypeConversionError
)

type Error struct {
	InternalStatusCode int      `json:"-"`
	HttpStatusCode     int      `json:"code"`
	Message            string   `json:"message"`
	Details            []string `json:"details,omitempty"`
	internal           error
}

var (
	ErrBadRequest                    = NewWithHTTPStatus(http.StatusBadRequest, ECBadRequest, "bad request")
	ErrNoContent                     = NewWithHTTPStatus(http.StatusNoContent, ECNoContent, "no content available")
	ErrValidationFailed              = NewWithHTTPStatus(http.StatusBadRequest, ECValidationError, "validation failed")
	ErrDBError                       = NewWithHTTPStatus(http.StatusInternalServerError, ECDatabaseError, "database error")
	ErrNotFound                      = NewWithHTTPStatus(http.StatusNotFound, ECNoRows, "no record found")
	ErrDBIntegrityConstrainViolation = NewWithHTTPStatus(http.StatusConflict, ECIntegrityConstrainViolation, "integrity constraint violation")
	ErrDBTransactionRollback         = NewWithHTTPStatus(http.StatusInternalServerError, ECTransactionRollback, "transaction rollback error")
	ErrDBTypeConversionError         = NewWithHTTPStatus(http.StatusInternalServerError, ECDatabaseTypeConversionError, "database type conversion error")
)

func NewWithHTTPStatus(internalSC, httpSC int, msg string, details ...string) *Error {
	return &Error{
		InternalStatusCode: internalSC,
		HttpStatusCode:     httpSC,
		Message:            msg,
		Details:            details,
		internal:           nil,
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
		return fmt.Sprintf("[%d] %s (original error: %s)", e.HttpStatusCode, e.Message, e.internal.Error())
	}
	return fmt.Sprintf("[%d] %s", e.HttpStatusCode, e.Message)
}

func (e *Error) ErrorWithDetails() string {
	sb := strings.Builder{}
	sb.WriteString("Error: ")
	sb.WriteString(fmt.Sprintf("  - [%d] %s\n", e.HttpStatusCode, e.Message))
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
		InternalStatusCode: e.InternalStatusCode,
		HttpStatusCode:     e.HttpStatusCode,
		Message:            e.Message,
		Details:            append([]string{}, e.Details...),
		internal:           e.internal,
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
		StatusCode: e.InternalStatusCode,
		Message:    e.Message,
		Details:    e.Details,
	}
}

func (e Error) MarshalAndWriteTo(w io.Writer) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
