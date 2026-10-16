// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
)

func networkQuery(projectID, availabilityZone, idKey, id string) map[string]string {
	query := map[string]string{"project_id": projectID, "availability_zone": availabilityZone}
	if idKey != "" {
		query[idKey] = id
	}
	return query
}

func (c *Client) networkObject(ctx context.Context, method, endpoint string, query map[string]string, body any, result any) error {
	var raw []byte
	var envelope struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if err := c.Do(ctx, Options{Method: method, Path: endpoint, Query: query, Body: body, Result: &envelope, RawBody: &raw}); err != nil {
		return err
	}
	if err := CheckStatus(method, endpoint, envelope.Status, raw); err != nil {
		return err
	}
	if result != nil {
		return json.Unmarshal(envelope.Data, result)
	}
	return nil
}
func updateEnvelope[T any](p, z, idKey, id, objectKey string, value T) map[string]any {
	return map[string]any{"project_id": p, "availability_zone": z, idKey: id, objectKey: value, "exclude_none": true}
}
func (c *Client) listNetworkObjects(ctx context.Context, endpoint string, p, z string, result any) error {
	return c.networkObject(ctx, http.MethodGet, endpoint, networkQuery(p, z, "", ""), nil, result)
}
