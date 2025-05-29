package errors

import (
	"fmt"
	"net/http"
	"strings"
)

// Error Code
// 000 - 099: General errors
const (
	ErrorCodeUnknown         = 000
	ErrorCodeMarshalFailed   = 001
	ErrorCodeUnmarshalFailed = 002
)

// HTTP 400 - 499: Client errors
const (
	ErrorCodeBadRequest      = http.StatusBadRequest
	ErrorCodeUnauthorized    = http.StatusUnauthorized
	ErrorCodeNoContent       = http.StatusNoContent
	ErrorCodeTooManyRequests = http.StatusTooManyRequests
)

// HTTP 500 - 599: Server errors
const (
	ErrorCodeInternalServerError = http.StatusInternalServerError
	ErrorCodeNotImplemented      = http.StatusNotImplemented
	ErrorCodeServiceUnavailable  = http.StatusServiceUnavailable
	ErrorCodeGatewayTimeout      = http.StatusGatewayTimeout
	ErrorCodeWebpageParsingError = 520
)

type Error struct {
	StatusCode int      `json:"-"`
	Code       int      `json:"code"`
	Message    string   `json:"message"`
	Details    []string `json:"details,omitempty"`
	internal   error
}

var (
	ErrBadRequest = NewWithHTTPStatus(http.StatusBadRequest, ErrorCodeBadRequest, "Bad request")
	ErrNoContent  = NewWithHTTPStatus(http.StatusNoContent, ErrorCodeNoContent, "No content available")
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
