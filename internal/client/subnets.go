// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type Subnet struct {
	ID              string           `json:"id"`
	Name            *string          `json:"name"`
	Description     *string          `json:"description"`
	ProjectID       *string          `json:"project_id"`
	NetworkID       string           `json:"network_id"`
	CIDR            string           `json:"cidr"`
	IPVersion       int64            `json:"ip_version"`
	GatewayIP       *string          `json:"gateway_ip"`
	EnableDHCP      *bool            `json:"enable_dhcp"`
	DNSNameservers  []string         `json:"dns_nameservers"`
	HostRoutes      []HostRoute      `json:"host_routes"`
	AllocationPools []AllocationPool `json:"allocation_pools"`
	Tags            []string         `json:"tags"`
}

type SubnetCreate struct {
	NetworkID       string           `json:"network_id"`
	CIDR            string           `json:"cidr"`
	IPVersion       int64            `json:"ip_version"`
	Name            *string          `json:"name,omitempty"`
	Description     *string          `json:"description,omitempty"`
	GatewayIP       *string          `json:"gateway_ip,omitempty"`
	EnableDHCP      *bool            `json:"enable_dhcp,omitempty"`
	DNSNameservers  []string         `json:"dns_nameservers,omitempty"`
	HostRoutes      []HostRoute      `json:"host_routes,omitempty"`
	AllocationPools []AllocationPool `json:"allocation_pools,omitempty"`
}
type HostRoute struct {
	Destination string `json:"destination"`
	NextHop     string `json:"nexthop"`
}

type AllocationPool struct {
	Start string `json:"start"`
	End   string `json:"end"`
}
type SubnetUpdate struct {
	Name            *string          `json:"name,omitempty"`
	EnableDHCP      *bool            `json:"enable_dhcp,omitempty"`
	GatewayIP       *string          `json:"gateway_ip,omitempty"`
	DNSNameservers  []string         `json:"dns_nameservers,omitempty"`
	HostRoutes      []HostRoute      `json:"host_routes,omitempty"`
	AllocationPools []AllocationPool `json:"allocation_pools,omitempty"`
	Description     *string          `json:"description,omitempty"`
}

func (c *Client) CreateSubnet(ctx context.Context, p, z string, v SubnetCreate) (*Subnet, error) {
	body := struct {
		ProjectID        string       `json:"project_id"`
		AvailabilityZone string       `json:"availability_zone"`
		Subnet           SubnetCreate `json:"subnet"`
	}{p, z, v}
	var d struct {
		Subnet Subnet `json:"subnet"`
	}
	e := c.networkObject(ctx, http.MethodPost, subnetsPath, nil, body, &d)
	return &d.Subnet, e
}
func (c *Client) GetSubnet(ctx context.Context, p, z, id string) (*Subnet, error) {
	var d struct {
		Subnet Subnet `json:"subnet"`
	}
	e := c.networkObject(ctx, http.MethodGet, subnetsPath, networkQuery(p, z, "subnet_id", id), nil, &d)
	return &d.Subnet, e
}
func (c *Client) DeleteSubnet(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, subnetsPath, networkQuery(p, z, "subnet_id", id), nil, nil)
}
func (c *Client) UpdateSubnet(ctx context.Context, p, z, id string, v SubnetUpdate) (*Subnet, error) {
	var d struct {
		Subnet Subnet `json:"subnet"`
	}
	err := c.networkObject(ctx, http.MethodPut, subnetsPath, nil, updateEnvelope(p, z, "subnet_id", id, "subnet", v), &d)
	return &d.Subnet, err
}
func (c *Client) ListSubnets(ctx context.Context, p, z string) ([]Subnet, error) {
	var d struct {
		Items []Subnet `json:"subnets"`
	}
	e := c.listNetworkObjects(ctx, subnetListPath, p, z, &d)
	return d.Items, e
}
