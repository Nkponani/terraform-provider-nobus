// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type FloatingIP struct {
	ID                string   `json:"id"`
	Description       *string  `json:"description"`
	ProjectID         *string  `json:"project_id"`
	FloatingNetworkID string   `json:"floating_network_id"`
	FloatingIPAddress *string  `json:"floating_ip_address"`
	PortID            *string  `json:"port_id"`
	FixedIPAddress    *string  `json:"fixed_ip_address"`
	Status            *string  `json:"status"`
	Tags              []string `json:"tags"`
}

type FloatingIPCreate struct {
	FloatingNetworkID string   `json:"floating_network_id"`
	PortID            *string  `json:"port_id,omitempty"`
	FixedIPAddress    *string  `json:"fixed_ip_address,omitempty"`
	Description       *string  `json:"description,omitempty"`
	Tags              []string `json:"tags,omitempty"`
}
type FloatingIPUpdate struct {
	PortID         *string  `json:"port_id,omitempty"`
	FixedIPAddress *string  `json:"fixed_ip_address,omitempty"`
	Description    *string  `json:"description,omitempty"`
	Tags           []string `json:"tags,omitempty"`
}

func (c *Client) CreateFloatingIP(ctx context.Context, p, z string, v FloatingIPCreate) (*FloatingIP, error) {
	body := struct {
		ProjectID        string           `json:"project_id"`
		AvailabilityZone string           `json:"availability_zone"`
		FloatingIP       FloatingIPCreate `json:"floatingip"`
	}{p, z, v}
	var d struct {
		FloatingIP FloatingIP `json:"floatingip"`
	}
	e := c.networkObject(ctx, http.MethodPost, floatingIPsPath, nil, body, &d)
	return &d.FloatingIP, e
}
func (c *Client) GetFloatingIP(ctx context.Context, p, z, id string) (*FloatingIP, error) {
	var d struct {
		FloatingIP FloatingIP `json:"floatingip"`
	}
	e := c.networkObject(ctx, http.MethodGet, floatingIPsPath, networkQuery(p, z, "floating_ip_id", id), nil, &d)
	return &d.FloatingIP, e
}
func (c *Client) DeleteFloatingIP(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, floatingIPsPath, networkQuery(p, z, "floating_ip_id", id), nil, nil)
}
func (c *Client) UpdateFloatingIP(ctx context.Context, p, z, id string, v FloatingIPUpdate) (*FloatingIP, error) {
	var d struct {
		FloatingIP FloatingIP `json:"floatingip"`
	}
	err := c.networkObject(ctx, http.MethodPut, floatingIPsPath, nil, updateEnvelope(p, z, "floatingip_id", id, "floatingip", v), &d)
	return &d.FloatingIP, err
}
func (c *Client) DisassociateFloatingIP(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodGet, floatingIPDisassociatePath, networkQuery(p, z, "floating_ip_id", id), nil, nil)
}
func (c *Client) ListFloatingIPs(ctx context.Context, p, z string) ([]FloatingIP, error) {
	var d struct {
		Items []FloatingIP `json:"floatingips"`
	}
	e := c.listNetworkObjects(ctx, floatingIPListPath, "floatingips", p, z, &d)
	return d.Items, e
}
