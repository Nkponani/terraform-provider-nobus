// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type Snapshot struct {
	ID          string            `json:"id"`
	Name        *string           `json:"name"`
	Description *string           `json:"description"`
	VolumeID    string            `json:"volume_id"`
	Status      *string           `json:"status"`
	Size        *int64            `json:"size"`
	CreatedAt   *string           `json:"created_at"`
	Metadata    map[string]string `json:"metadata"`
}

type CreateSnapshotRequest struct {
	VolumeID         string            `json:"volume_id"`
	Name             *string           `json:"name,omitempty"`
	Description      *string           `json:"description,omitempty"`
	Force            *bool             `json:"force,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	ProjectID        string            `json:"project_id"`
	AvailabilityZone string            `json:"availability_zone"`
}

func (c *Client) CreateSnapshot(ctx context.Context, req CreateSnapshotRequest) (*Snapshot, error) {
	var out Snapshot
	err := c.storageObject(ctx, http.MethodPost, snapshotsPath, nil, req, &out)
	return &out, err
}
func (c *Client) GetSnapshot(ctx context.Context, p, z, id string) (*Snapshot, error) {
	var out Snapshot
	err := c.storageObject(ctx, http.MethodGet, snapshotsPath, storageQuery(p, z, "snapshot_id", id), nil, &out)
	return &out, err
}
func (c *Client) DeleteSnapshot(ctx context.Context, p, z, id string) error {
	return c.storageObject(ctx, http.MethodDelete, snapshotsPath, storageQuery(p, z, "snapshot_id", id), nil, nil)
}
func (c *Client) ListSnapshots(ctx context.Context, p, z string, filters map[string]string) ([]Snapshot, error) {
	q := storageQuery(p, z, "", "")
	for k, v := range filters {
		q[k] = v
	}
	var out struct {
		Snapshots []Snapshot `json:"items"`
	}
	err := c.storageObject(ctx, http.MethodGet, snapshotListPath, q, nil, &out)
	return out.Snapshots, err
}
