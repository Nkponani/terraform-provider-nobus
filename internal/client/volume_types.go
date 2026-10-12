// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type VolumeType struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description *string           `json:"description"`
	IsPublic    *bool             `json:"is_public"`
	ExtraSpecs  map[string]string `json:"extra_specs"`
}

type CreateVolumeTypeRequest struct {
	Name             string            `json:"name"`
	Description      *string           `json:"description,omitempty"`
	IsPublic         *bool             `json:"is_public,omitempty"`
	ExtraSpecs       map[string]string `json:"extra_specs,omitempty"`
	ProjectID        string            `json:"project_id"`
	AvailabilityZone string            `json:"availability_zone"`
}

func (c *Client) CreateVolumeType(ctx context.Context, req CreateVolumeTypeRequest) (*VolumeType, error) {
	var out VolumeType
	err := c.storageObject(ctx, http.MethodPost, volumeTypesPath, nil, req, &out)
	return &out, err
}

type volumeTypeListResponse struct {
	Status bool         `json:"status"`
	Data   []VolumeType `json:"data"`
}

func (c *Client) ListVolumeTypes(ctx context.Context, projectID, availabilityZone string) ([]VolumeType, error) {
	var out volumeTypeListResponse
	err := c.storageList(ctx, http.MethodGet, volumeTypesPath, storageQuery(projectID, availabilityZone, "", ""), &out)
	return out.Data, err
}

func (c *Client) GetVolumeType(ctx context.Context, projectID, availabilityZone, volumeTypeID string) (*VolumeType, error) {
	var out VolumeType
	err := c.storageObject(ctx, http.MethodGet, volumeTypesPath, storageQuery(projectID, availabilityZone, "id", volumeTypeID), nil, &out)
	return &out, err
}

func (c *Client) DeleteVolumeType(ctx context.Context, projectID, availabilityZone, volumeTypeID string) error {
	return c.storageNoContent(ctx, http.MethodDelete, volumeTypesPath, storageQuery(projectID, availabilityZone, "id", volumeTypeID))
}
