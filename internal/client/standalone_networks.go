// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type StandaloneNetworkCreate struct {
	Name                string   `json:"name"`
	AdminStateUp        *bool    `json:"admin_state_up,omitempty"`
	Shared              *bool    `json:"shared,omitempty"`
	MTU                 *int64   `json:"mtu,omitempty"`
	RouterExternal      *bool    `json:"router:external,omitempty"`
	PortSecurityEnabled *bool    `json:"port_security_enabled,omitempty"`
	Description         *string  `json:"description,omitempty"`
	Tags                []string `json:"tags,omitempty"`
}

func (c *Client) CreateStandaloneNetwork(ctx context.Context, p, z string, v StandaloneNetworkCreate) (*Network, error) {
	body := map[string]any{"project_id": p, "availability_zone": z, "network": v}
	var d struct {
		Network Network `json:"network"`
	}
	err := c.networkObject(ctx, http.MethodPost, standaloneNetworksPath, nil, body, &d)
	return &d.Network, err
}
