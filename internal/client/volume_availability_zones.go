// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type AvailabilityZone struct {
	Name      string         `json:"name"`
	ZoneState map[string]any `json:"zoneState"`
	Hosts     map[string]any `json:"hosts"`
}

func (c *Client) ListVolumeAvailabilityZones(ctx context.Context, p, z string) ([]AvailabilityZone, error) {
	var out struct {
		AvailabilityZones []AvailabilityZone `json:"items"`
	}
	err := c.storageObject(ctx, http.MethodGet, volumeAvailabilityZonesPath, storageQuery(p, z, "", ""), nil, &out)
	return out.AvailabilityZones, err
}
