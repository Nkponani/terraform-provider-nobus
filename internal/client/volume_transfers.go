// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type VolumeTransfer struct {
	ID        string  `json:"id"`
	Name      *string `json:"name"`
	VolumeID  string  `json:"volume_id"`
	CreatedAt *string `json:"created_at"`
	AuthKey   *string `json:"auth_key"`
}

type CreateVolumeTransferRequest struct {
	VolumeID         string  `json:"volume_id"`
	Name             *string `json:"name,omitempty"`
	ProjectID        string  `json:"project_id"`
	AvailabilityZone string  `json:"availability_zone"`
}

type AcceptVolumeTransferRequest struct {
	TransferID       string `json:"transfer_id"`
	AuthKey          string `json:"auth_key"`
	ProjectID        string `json:"project_id"`
	AvailabilityZone string `json:"availability_zone"`
}

func (c *Client) CreateVolumeTransfer(ctx context.Context, req CreateVolumeTransferRequest) (*VolumeTransfer, error) {
	var out VolumeTransfer
	err := c.storageObject(ctx, http.MethodPost, volumeTransfersPath, nil, req, &out)
	return &out, err
}
func (c *Client) GetVolumeTransfer(ctx context.Context, p, z, id string) (*VolumeTransfer, error) {
	var out VolumeTransfer
	err := c.storageObject(ctx, http.MethodGet, volumeTransfersPath, storageQuery(p, z, "volume_transfer_id", id), nil, &out)
	return &out, err
}
func (c *Client) DeleteVolumeTransfer(ctx context.Context, p, z, id string) error {
	return c.storageObject(ctx, http.MethodDelete, volumeTransfersPath, storageQuery(p, z, "volume_transfer_id", id), nil, nil)
}
func (c *Client) ListVolumeTransfers(ctx context.Context, p, z string) ([]VolumeTransfer, error) {
	var out struct {
		Transfers []VolumeTransfer `json:"items"`
	}
	err := c.storageObject(ctx, http.MethodGet, volumeTransferListPath, storageQuery(p, z, "", ""), nil, &out)
	return out.Transfers, err
}
func (c *Client) AcceptVolumeTransfer(ctx context.Context, req AcceptVolumeTransferRequest) (*VolumeTransfer, error) {
	var out VolumeTransfer
	err := c.storageObject(ctx, http.MethodPost, volumeTransferAcceptPath, nil, req, &out)
	return &out, err
}
