// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"strings"
)

// CheckStatus converts an unsuccessful Nobus response into a *StatusError so
// that callers never treat a "status": false payload as success.
//
// The Nobus API answers with HTTP 200 and reports failures through the
// "status" boolean inside the response envelope, so this check is required on
// every operation. raw is the response body and is used to recover the message
// Nobus returned, whether it sits at the top level or inside "data".
func CheckStatus(method, path string, status bool, raw []byte) error {
	if status {
		return nil
	}

	return &StatusError{Method: method, Path: path, Message: extractMessage(raw)}
}

// extractMessage pulls a human readable reason out of an opaque payload.
//
// Nobus returns errors either at the top level of the envelope or nested inside
// "data", so both shapes are inspected.
func extractMessage(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var payload struct {
		Message string          `json:"message"`
		Error   string          `json:"error"`
		Detail  string          `json:"detail"`
		Reason  string          `json:"reason"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}

	for _, candidate := range []string{payload.Message, payload.Error, payload.Detail, payload.Reason} {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}

	// Fall back to the nested payload, but only when it is an object; a plain
	// string "data" is the operation result rather than an error reason.
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(payload.Data, &nested); err == nil {
		return extractMessage(payload.Data)
	}

	return ""
}
