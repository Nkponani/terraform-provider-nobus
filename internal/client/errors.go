// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError represents an error response returned by the Nobus API.
//
// Nobus documents {"message": string} as its error payload. Some deployments
// add additional fields; those are preserved in Body so no diagnostic
// information is lost.
type APIError struct {
	// Method and Path identify the failed request.
	Method string
	Path   string

	// StatusCode is the HTTP status code.
	StatusCode int

	// Message is the "message" field returned by Nobus, if present.
	Message string

	// Body is the raw response body, truncated for readability.
	Body string
}

const maxErrorBody = 2048

func newAPIError(method, path string, statusCode int, raw []byte) *APIError {
	apiErr := &APIError{
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
	}

	var envelope struct {
		Message string `json:"message"`
		Detail  any    `json:"detail"`
		Error   any    `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil {
		apiErr.Message = strings.TrimSpace(envelope.Message)
	}

	body := strings.TrimSpace(string(raw))
	if len(body) > maxErrorBody {
		body = body[:maxErrorBody] + "... (truncated)"
	}
	apiErr.Body = body

	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(statusCode)
	}

	return apiErr
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "nobus API returned %d", e.StatusCode)
	if e.Method != "" || e.Path != "" {
		fmt.Fprintf(&b, " for %s %s", e.Method, e.Path)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	return b.String()
}

// IsNotFound reports whether err represents a missing Nobus object.
//
// The public Nobus API does not document a 404 response for these resources.
// A missing object therefore surfaces either as a 404, as a 400/500 carrying a
// "not found" style message, or as an HTTP 200 envelope with "status": false
// and such a message. All three forms are recognised here so that Read can
// reliably drop a resource from Terraform state.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr.IsNotFound()
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	if apiErr.StatusCode == http.StatusNotFound {
		return true
	}

	return messageIndicatesMissing(apiErr.Message)
}

func messageIndicatesMissing(message string) bool {
	m := strings.ToLower(message)
	if m == "" {
		return false
	}
	for _, needle := range []string{
		"not found",
		"does not exist",
		"doesn't exist",
		"no such",
		"not exist",
		"no networks",
		"no volumes",
		"no instances",
		"unknown network",
		"unknown volume",
		"unknown instance",
		"unknown server",
	} {
		if strings.Contains(m, needle) {
			return true
		}
	}
	return false
}

// StatusError indicates that the API responded with HTTP 200 but the Nobus
// envelope reported "status": false.
type StatusError struct {
	// Method and Path identify the failed request.
	Method string
	Path   string

	// Message is the human readable reason extracted from the response.
	Message string
}

func (e *StatusError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "the operation was not successful"
	}
	if e.Method != "" || e.Path != "" {
		return fmt.Sprintf("nobus API reported failure for %s %s: %s", e.Method, e.Path, msg)
	}
	return "nobus API reported failure: " + msg
}

// IsNotFound also treats a "status": false envelope as a missing object when
// the message indicates the object does not exist.
func (e *StatusError) IsNotFound() bool { return messageIndicatesMissing(e.Message) }

// TransportError wraps a failure to complete the HTTP exchange.
type TransportError struct {
	Method string
	Path   string
	Err    error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("nobus: %s %s: %s", e.Method, e.Path, e.Err)
}

func (e *TransportError) Unwrap() error { return e.Err }

// DecodeError indicates a successful HTTP response whose body could not be
// decoded into the expected schema.
type DecodeError struct {
	Method     string
	Path       string
	StatusCode int
	Err        error
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("nobus: decoding response from %s %s (HTTP %d): %s", e.Method, e.Path, e.StatusCode, e.Err)
}

func (e *DecodeError) Unwrap() error { return e.Err }
