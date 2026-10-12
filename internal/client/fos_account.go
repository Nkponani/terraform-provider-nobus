// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type FOSAccount struct {
	ContainerCount int64
	ObjectCount    int64
	BytesUsed      int64
}

type FOSAuthDetails struct {
	ProjectID               string `json:"project_id"`
	UserName                string `json:"user_name"`
	ProjectName             string `json:"project_name"`
	DomainID                string `json:"domain_id"`
	Region                  string `json:"region"`
	IdentityServiceEndpoint string `json:"identity_service_endpoint"`
	IdentityServiceVersion  int64  `json:"identity_service_version"`
	ManagementEndpoint      string `json:"management_endpoint"`
}

func (c *Client) GetFOSAccount(ctx context.Context, projectID string) (*FOSAccount, error) {
	var raw []byte
	var response struct {
		Status bool `json:"status"`
		Data   struct {
			ContainerCount flexibleInt64 `json:"container_count"`
			ObjectCount    flexibleInt64 `json:"object_count"`
			BytesUsed      flexibleInt64 `json:"bytes_used"`
		} `json:"data"`
	}
	err := c.Do(ctx, Options{Method: http.MethodGet, Path: fosAccountPath, Query: fosQuery(projectID, ""), Result: &response, RawBody: &raw})
	if err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, fosAccountPath, response.Status, raw); err != nil {
		return nil, err
	}
	return &FOSAccount{ContainerCount: int64(response.Data.ContainerCount), ObjectCount: int64(response.Data.ObjectCount), BytesUsed: int64(response.Data.BytesUsed)}, nil
}

func (c *Client) GetFOSAuthDetails(ctx context.Context, projectID string) (*FOSAuthDetails, error) {
	var raw []byte
	var response struct {
		Status bool           `json:"status"`
		Data   FOSAuthDetails `json:"data"`
	}
	err := c.Do(ctx, Options{Method: http.MethodGet, Path: fosAuthDetailsPath, Query: fosQuery(projectID, ""), Result: &response, RawBody: &raw})
	if err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, fosAuthDetailsPath, response.Status, raw); err != nil {
		return nil, err
	}
	return &response.Data, nil
}
