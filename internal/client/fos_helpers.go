// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

type fosMessageResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

func fosQuery(projectID, availabilityZone string) map[string]string {
	query := map[string]string{"project_id": projectID}
	if availabilityZone != "" {
		query["availability_zone"] = availabilityZone
	}
	return query
}

func checkFOSMessage(method, path string, response fosMessageResponse, raw []byte) error {
	if response.Status {
		return nil
	}
	if response.Message != "" {
		return &StatusError{Method: method, Path: path, Message: response.Message}
	}
	return CheckStatus(method, path, false, raw)
}

type flexibleInt64 int64

func (v *flexibleInt64) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*v = 0
		return nil
	}
	var number json.Number
	if data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		parsed, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer %q: %w", text, err)
		}
		*v = flexibleInt64(parsed)
		return nil
	}
	if err := json.Unmarshal(data, &number); err != nil {
		return err
	}
	parsed, err := number.Int64()
	if err != nil {
		floatValue, floatErr := strconv.ParseFloat(number.String(), 64)
		if floatErr != nil {
			return err
		}
		parsed = int64(floatValue)
	}
	*v = flexibleInt64(parsed)
	return nil
}
