// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
)

const keypairsPath = "/api/v3/keypair/"

// CreateKeypairRequest mirrors CreateKeypairRequestSchema.
type CreateKeypairRequest struct {
	Name             string `json:"name"`
	ProjectID        string `json:"project_id"`
	AvailabilityZone string `json:"availability_zone"`
}

// Keypair mirrors KeypairSchema.
type Keypair struct {
	ID          *string `json:"id"`
	Name        *string `json:"name"`
	PublicKey   *string `json:"public_key"`
	Fingerprint *string `json:"fingerprint"`
	UserID      *string `json:"user_id"`
	Type        *string `json:"type"`
}

type keypairResponse struct {
	Status bool    `json:"status"`
	Data   Keypair `json:"data"`
}

type keypairsResponse struct {
	Status bool      `json:"status"`
	Data   []Keypair `json:"data"`
}

func (c *Client) CreateKeypair(ctx context.Context, req CreateKeypairRequest) (*Keypair, error) {
	var out keypairResponse
	var raw []byte
	if err := c.Do(ctx, Options{Method: http.MethodPost, Path: keypairsPath, Body: req, Result: &out, RawBody: &raw}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodPost, keypairsPath, out.Status, raw); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

func (c *Client) ListKeypairs(ctx context.Context, projectID, availabilityZone string) ([]Keypair, error) {
	var out keypairsResponse
	var raw []byte
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: keypairsPath, Query: map[string]string{"project_id": projectID, "availability_zone": availabilityZone}, Result: &out, RawBody: &raw}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, keypairsPath, out.Status, raw); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) DeleteKeypair(ctx context.Context, projectID, availabilityZone, keypairID string) error {
	var out struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	var raw []byte
	if err := c.Do(ctx, Options{Method: http.MethodDelete, Path: keypairsPath, Query: map[string]string{"project_id": projectID, "availability_zone": availabilityZone, "keypair_id": keypairID}, Result: &out, RawBody: &raw}); err != nil {
		return err
	}
	return CheckStatus(http.MethodDelete, keypairsPath, out.Status, raw)
}
