// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type Backup struct {
	ID               string            `json:"id"`
	Name             *string           `json:"name"`
	Description      *string           `json:"description"`
	VolumeID         string            `json:"volume_id"`
	Size             *int64            `json:"size"`
	Container        *string           `json:"container"`
	Status           *string           `json:"status"`
	CreatedAt        *string           `json:"created_at"`
	AvailabilityZone *string           `json:"availability_zone"`
	Metadata         map[string]string `json:"metadata"`
}

func (c *Client) GetBackup(ctx context.Context, p, z, id string) (*Backup, error) {
	var out Backup
	err := c.storageObject(ctx, http.MethodGet, backupsPath, storageQuery(p, z, "backup_id", id), nil, &out)
	return &out, err
}
func (c *Client) ListBackups(ctx context.Context, p, z string) ([]Backup, error) {
	var out struct {
		Backups []Backup `json:"items"`
	}
	err := c.storageObject(ctx, http.MethodGet, backupListPath, storageQuery(p, z, "", ""), nil, &out)
	return out.Backups, err
}
