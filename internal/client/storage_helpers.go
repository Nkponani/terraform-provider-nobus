// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
)

func (c *Client) storageObject(ctx context.Context, method, endpoint string, query map[string]string, body, result any) error {
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

func storageQuery(projectID, availabilityZone, key, id string) map[string]string {
	query := map[string]string{"project_id": projectID, "availability_zone": availabilityZone}
	if key != "" {
		query[key] = id
	}
	return query
}

func (c *Client) storageList(ctx context.Context, method, endpoint string, query map[string]string, result any) error {
	var raw []byte
	var envelope struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if err := c.Do(ctx, Options{Method: method, Path: endpoint, Query: query, Result: &envelope, RawBody: &raw}); err != nil {
		return err
	}
	if err := CheckStatus(method, endpoint, envelope.Status, raw); err != nil {
		return err
	}
	if result != nil {
		return json.Unmarshal(envelope.Data, &result)
	}
	return nil
}

func (c *Client) storageNoContent(ctx context.Context, method, endpoint string, query map[string]string) error {
	var raw []byte
	var envelope struct {
		Status bool   `json:"status"`
		Data   string `json:"data"`
	}
	if err := c.Do(ctx, Options{Method: method, Path: endpoint, Query: query, Result: &envelope, RawBody: &raw}); err != nil {
		return err
	}
	return CheckStatus(method, endpoint, envelope.Status, raw)
}
