// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"strconv"
)

type FOSContainer struct {
	Name         string  `json:"name"`
	Count        int64   `json:"count"`
	Bytes        int64   `json:"bytes"`
	LastModified *string `json:"last_modified"`
}

type FOSListOptions struct {
	Limit       int64
	Marker      string
	EndMarker   string
	Prefix      string
	Delimiter   string
	Path        string
	FullListing bool
}

func addFOSListOptions(query map[string]string, options FOSListOptions, includePath bool) {
	if options.Limit > 0 {
		query["limit"] = strconv.FormatInt(options.Limit, 10)
	}
	for key, value := range map[string]string{"marker": options.Marker, "end_marker": options.EndMarker, "prefix": options.Prefix, "delimiter": options.Delimiter} {
		if value != "" {
			query[key] = value
		}
	}
	if includePath && options.Path != "" {
		query["path"] = options.Path
	}
	if options.FullListing {
		query["full_listing"] = "true"
	}
}

func (c *Client) CreateFOSContainer(ctx context.Context, projectID, availabilityZone, name string, isPublic bool) error {
	body := map[string]any{"project_id": projectID, "availability_zone": availabilityZone, "container_name": name, "is_public": isPublic}
	var raw []byte
	var response fosMessageResponse
	if err := c.Do(ctx, Options{Method: http.MethodPost, Path: fosContainerPath, Body: body, Result: &response, RawBody: &raw}); err != nil {
		return err
	}
	return checkFOSMessage(http.MethodPost, fosContainerPath, response, raw)
}

func (c *Client) DeleteFOSContainer(ctx context.Context, projectID, availabilityZone, name string) error {
	query := fosQuery(projectID, availabilityZone)
	query["container_name"] = name
	var raw []byte
	var response fosMessageResponse
	if err := c.Do(ctx, Options{Method: http.MethodDelete, Path: fosContainerPath, Query: query, Result: &response, RawBody: &raw}); err != nil {
		return err
	}
	return checkFOSMessage(http.MethodDelete, fosContainerPath, response, raw)
}

func (c *Client) ListFOSContainers(ctx context.Context, projectID, availabilityZone string, options FOSListOptions) ([]FOSContainer, error) {
	query := fosQuery(projectID, availabilityZone)
	addFOSListOptions(query, options, false)
	var raw []byte
	var response struct {
		Status     bool           `json:"status"`
		Containers []FOSContainer `json:"containers"`
		Count      int64          `json:"count"`
	}
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: fosContainerPath, Query: query, Result: &response, RawBody: &raw}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, fosContainerPath, response.Status, raw); err != nil {
		return nil, err
	}
	return response.Containers, nil
}
