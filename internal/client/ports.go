// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type Port struct {
	ID                  string            `json:"id"`
	Name                *string           `json:"name"`
	Description         *string           `json:"description"`
	ProjectID           *string           `json:"project_id"`
	NetworkID           string            `json:"network_id"`
	AdminStateUp        *bool             `json:"admin_state_up"`
	Status              *string           `json:"status"`
	MACAddress          *string           `json:"mac_address"`
	DeviceID            *string           `json:"device_id"`
	DeviceOwner         *string           `json:"device_owner"`
	SecurityGroups      []string          `json:"security_groups"`
	PortSecurity        *bool             `json:"port_security_enabled"`
	DNSName             *string           `json:"dns_name"`
	FixedIPs            []FixedIP         `json:"fixed_ips"`
	AllowedAddressPairs []PortAddressPair `json:"allowed_address_pairs"`
	BindingHostID       *string           `json:"binding_host_id"`
	BindingVNICType     *string           `json:"binding_vnic_type"`
	BindingProfile      map[string]any    `json:"binding_profile"`
	ExtraDHCPOptions    []ExtraDHCPOption `json:"extra_dhcp_opts"`
	Tags                []string          `json:"tags"`
}

type PortCreate struct {
	NetworkID           string            `json:"network_id"`
	Name                *string           `json:"name,omitempty"`
	Description         *string           `json:"description,omitempty"`
	AdminStateUp        *bool             `json:"admin_state_up,omitempty"`
	DeviceID            *string           `json:"device_id,omitempty"`
	DeviceOwner         *string           `json:"device_owner,omitempty"`
	SecurityGroups      []string          `json:"security_groups,omitempty"`
	PortSecurity        *bool             `json:"port_security_enabled,omitempty"`
	DNSName             *string           `json:"dns_name,omitempty"`
	FixedIPs            []FixedIP         `json:"fixed_ips,omitempty"`
	AllowedAddressPairs []PortAddressPair `json:"allowed_address_pairs,omitempty"`
	BindingHostID       *string           `json:"binding:host_id,omitempty"`
	BindingVNICType     *string           `json:"binding:vnic_type,omitempty"`
	BindingProfile      map[string]any    `json:"binding:profile,omitempty"`
	ExtraDHCPOptions    []ExtraDHCPOption `json:"extra_dhcp_opts,omitempty"`
	Tags                []string          `json:"tags,omitempty"`
}
type FixedIP struct {
	SubnetID  string  `json:"subnet_id"`
	IPAddress *string `json:"ip_address,omitempty"`
}

type PortAddressPair struct {
	IPAddress  string  `json:"ip_address"`
	MACAddress *string `json:"mac_address,omitempty"`
}

type ExtraDHCPOption struct {
	Name  string `json:"opt_name"`
	Value string `json:"opt_value"`
}
type PortUpdate struct {
	Name                *string           `json:"name,omitempty"`
	Description         *string           `json:"description,omitempty"`
	AdminStateUp        *bool             `json:"admin_state_up,omitempty"`
	DeviceID            *string           `json:"device_id,omitempty"`
	DeviceOwner         *string           `json:"device_owner,omitempty"`
	SecurityGroups      []string          `json:"security_groups,omitempty"`
	FixedIPs            []FixedIP         `json:"fixed_ips,omitempty"`
	AllowedAddressPairs []PortAddressPair `json:"allowed_address_pairs,omitempty"`
	PortSecurity        *bool             `json:"port_security_enabled,omitempty"`
	BindingHostID       *string           `json:"binding:host_id,omitempty"`
	BindingVNICType     *string           `json:"binding:vnic_type,omitempty"`
	BindingProfile      map[string]any    `json:"binding:profile,omitempty"`
	DNSName             *string           `json:"dns_name,omitempty"`
	ExtraDHCPOptions    []ExtraDHCPOption `json:"extra_dhcp_opts,omitempty"`
	Tags                []string          `json:"tags,omitempty"`
}

func (c *Client) CreatePort(ctx context.Context, p, z string, v PortCreate) (*Port, error) {
	body := struct {
		ProjectID        string     `json:"project_id"`
		AvailabilityZone string     `json:"availability_zone"`
		Port             PortCreate `json:"port"`
	}{p, z, v}
	var d struct {
		Port Port `json:"port"`
	}
	e := c.networkObject(ctx, http.MethodPost, portsPath, nil, body, &d)
	return &d.Port, e
}
func (c *Client) GetPort(ctx context.Context, p, z, id string) (*Port, error) {
	var d struct {
		Port Port `json:"port"`
	}
	e := c.networkObject(ctx, http.MethodGet, portsPath, networkQuery(p, z, "port_id", id), nil, &d)
	return &d.Port, e
}
func (c *Client) DeletePort(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, portsPath, networkQuery(p, z, "port_id", id), nil, nil)
}
func (c *Client) UpdatePort(ctx context.Context, p, z, id string, v PortUpdate) (*Port, error) {
	var d struct {
		Port Port `json:"port"`
	}
	err := c.networkObject(ctx, http.MethodPut, portsPath, nil, updateEnvelope(p, z, "port_id", id, "port", v), &d)
	return &d.Port, err
}
func (c *Client) SetPortAddressPairs(ctx context.Context, p, z, id string, pairs []PortAddressPair) error {
	body := map[string]any{"project_id": p, "availability_zone": z, "port_id": id, "address_pairs": pairs}
	return c.networkObject(ctx, http.MethodPost, portAddressPairsPath, nil, body, nil)
}
func (c *Client) ClearPortAddressPairs(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodPost, portClearAddressPairsPath, networkQuery(p, z, "port_id", id), nil, nil)
}
func (c *Client) ListPorts(ctx context.Context, p, z string) ([]Port, error) {
	var d struct {
		Items []Port `json:"ports"`
	}
	e := c.listNetworkObjects(ctx, portListPath, p, z, &d)
	return d.Items, e
}
