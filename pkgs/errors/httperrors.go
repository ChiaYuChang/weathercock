package errors

import (
	"fmt"
)

type HTTPError struct {
	StatusCode int      `json:"status_code"`
	Message    string   `json:"message"`
	Details    []string `json:"details,omitempty"`
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("[%d] %s", e.StatusCode, e.Message)
}
