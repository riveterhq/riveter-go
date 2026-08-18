package riveter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// APIError is a non-2xx HTTP response, normalized across both error body
// dialects — the clean { "error": {...} } envelope and the legacy
// { "request_status": "error" } shape (401s).
type APIError struct {
	StatusCode int
	Type       string
	Message    string
	Details    any
}

func (e *APIError) Error() string {
	return fmt.Sprintf("riveter: API error %d (%s): %s", e.StatusCode, e.Type, e.Message)
}

// ConnectionError is a network-level failure: the request never produced an
// HTTP response.
type ConnectionError struct {
	Err error
}

func (e *ConnectionError) Error() string {
	return "riveter: connection error: " + e.Err.Error()
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// TimeoutError is returned by WaitForResult when the run does not finish
// within the configured budget.
type TimeoutError struct {
	Message string
}

func (e *TimeoutError) Error() string {
	return "riveter: " + e.Message
}

var fallbackErrorTypes = map[int]string{
	400: "bad_request",
	401: "unauthorized",
	403: "forbidden",
	404: "not_found",
	409: "conflict",
	422: "validation",
	429: "rate_limited",
}

func newAPIError(status int, body []byte) *APIError {
	var envelope struct {
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Details any    `json:"details"`
		} `json:"error"`
		RequestStatus string `json:"request_status"`
		Message       string `json:"message"`
		ErrorType     string `json:"error_type"`
		Errors        any    `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		if envelope.Error != nil && envelope.Error.Type != "" {
			return &APIError{
				StatusCode: status,
				Type:       envelope.Error.Type,
				Message:    envelope.Error.Message,
				Details:    envelope.Error.Details,
			}
		}
		if envelope.RequestStatus == "error" {
			errorType := envelope.ErrorType
			if errorType == "" {
				errorType = "error"
			}
			return &APIError{
				StatusCode: status,
				Type:       errorType,
				Message:    envelope.Message,
				Details:    envelope.Errors,
			}
		}
	}

	errorType, ok := fallbackErrorTypes[status]
	if !ok {
		errorType = "http_" + strconv.Itoa(status)
	}
	message := strings.TrimSpace(string(body))
	if len(message) > 200 {
		message = message[:200]
	}
	if message == "" {
		message = "HTTP " + strconv.Itoa(status)
	}
	return &APIError{StatusCode: status, Type: errorType, Message: message}
}
