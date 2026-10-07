// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

// Network paths documented by the Nobus OpenAPI specification.
const (
	networksPath = "/api/v3/network/"
	networkList  = "/api/v3/network/list"
)

// CreateNetworkRequest mirrors CreateNetworkRequestSchema.
//
// Required by the API: ProjectID, AvailabilityZone, NetworkName, SubnetCIDR
// and RouterName. A single call creates the network together with its subnet
// and router, which is why the response reports all three identifiers.
type CreateNetworkRequest struct {
	ProjectID         string   `json:"project_id"`
	AvailabilityZone  string   `json:"availability_zone"`
	NetworkName       string   `json:"network_name"`
	SubnetCIDR        string   `json:"subnet_cidr"`
	RouterName        string   `json:"router_name"`
	ExternalNetworkID *string  `json:"external_network_id,omitempty"`
	DNSNameservers    []string `json:"dns_nameservers,omitempty"`
	EnableSNAT        *bool    `json:"enable_snat,omitempty"`
}

// NetworkSetupOut mirrors NetworkSetupOut, the payload returned on create.
type NetworkSetupOut struct {
	NetworkID string `json:"network_id"`
	SubnetID  string `json:"subnet_id"`
	RouterID  string `json:"router_id"`
}

type createNetworkResponse struct {
	Status bool            `json:"status"`
	Data   NetworkSetupOut `json:"data"`
}

// Network mirrors NetworkOut.
type Network struct {
	ID                  string   `json:"id"`
	Name                *string  `json:"name"`
	Description         *string  `json:"description"`
	TenantID            *string  `json:"tenant_id"`
	ProjectID           *string  `json:"project_id"`
	Tags                []string `json:"tags"`
	AdminStateUp        *bool    `json:"admin_state_up"`
	Status              *string  `json:"status"`
	Shared              *bool    `json:"shared"`
	MTU                 *int64   `json:"mtu"`
	RouterExternal      *bool    `json:"router_external"`
	PortSecurityEnabled *bool    `json:"port_security_enabled"`
	Subnets             []string `json:"subnets"`
}

// NetworkUpdate mirrors NetworkUpdate, the mutable subset of a network.
//
// Every field is optional and only the fields that are set are applied,
// because NetworkUpdateRequestBody defaults exclude_none to true. Note that
// the API spells the external router flag "router:external" on update but
// "router_external" on read.
type NetworkUpdate struct {
	Name                *string   `json:"name,omitempty"`
	AdminStateUp        *bool     `json:"admin_state_up,omitempty"`
	Shared              *bool     `json:"shared,omitempty"`
	MTU                 *int64    `json:"mtu,omitempty"`
	RouterExternal      *bool     `json:"router:external,omitempty"`
	PortSecurityEnabled *bool     `json:"port_security_enabled,omitempty"`
	Description         *string   `json:"description,omitempty"`
	Tags                *[]string `json:"tags,omitempty"`
}

// NetworkUpdateRequest mirrors NetworkUpdateRequestBody.
type NetworkUpdateRequest struct {
	ProjectID        string        `json:"project_id"`
	AvailabilityZone string        `json:"availability_zone"`
	NetworkID        string        `json:"network_id"`
	Network          NetworkUpdate `json:"network"`
	ExcludeNone      *bool         `json:"exclude_none,omitempty"`
}

type networkResponse struct {
	Status bool `json:"status"`
	Data   struct {
		Network Network `json:"network"`
	} `json:"data"`
}

type networksResponse struct {
	Status bool `json:"status"`
	Data   struct {
		Networks []Network `json:"networks"`
	} `json:"data"`
}

// CreateNetwork creates a network along with its subnet and router.
func (c *Client) CreateNetwork(ctx context.Context, req CreateNetworkRequest) (*NetworkSetupOut, error) {
	var out createNetworkResponse
	var raw []byte
	if err := c.Do(ctx, Options{
		Method:  http.MethodPost,
		Path:    networksPath,
		Body:    req,
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return nil, err
	}

	if err := CheckStatus(http.MethodPost, networksPath, out.Status, raw); err != nil {
		return nil, err
	}
	if out.Data.NetworkID == "" {
		return nil, &StatusError{
			Method:  http.MethodPost,
			Path:    networksPath,
			Message: "the API did not return a network_id",
		}
	}

	return &out.Data, nil
}

// GetNetwork reads a single network.
//
// The API takes project_id, network_id and availability_zone as required query
// parameters.
func (c *Client) GetNetwork(ctx context.Context, projectID, availabilityZone, networkID string) (*Network, error) {
	var out networkResponse
	var raw []byte
	err := c.Do(ctx, Options{
		Method: http.MethodGet,
		Path:   networksPath,
		Query: map[string]string{
			"project_id":        projectID,
			"availability_zone": availabilityZone,
			"network_id":        networkID,
		},
		Result:  &out,
		RawBody: &raw,
	})
	if err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, networksPath, out.Status, raw); err != nil {
		return nil, err
	}

	return &out.Data.Network, nil
}

// UpdateNetwork applies a partial update to an existing network.
func (c *Client) UpdateNetwork(ctx context.Context, req NetworkUpdateRequest) (*Network, error) {
	var out networkResponse
	var raw []byte
	if err := c.Do(ctx, Options{
		Method:  http.MethodPut,
		Path:    networksPath,
		Body:    req,
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodPut, networksPath, out.Status, raw); err != nil {
		return nil, err
	}

	return &out.Data.Network, nil
}

// DeleteNetwork deletes a network. Callers should treat a not-found error as a
// successful deletion.
func (c *Client) DeleteNetwork(ctx context.Context, projectID, availabilityZone, networkID string) error {
	var out struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	var raw []byte
	if err := c.Do(ctx, Options{
		Method: http.MethodDelete,
		Path:   networksPath,
		Query: map[string]string{
			"project_id":        projectID,
			"availability_zone": availabilityZone,
			"network_id":        networkID,
		},
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return err
	}

	return CheckStatus(http.MethodDelete, networksPath, out.Status, raw)
}

// ListNetworks returns one page of networks.
//
// Pagination uses the documented limit/marker parameters; an empty marker
// returns the first page.
func (c *Client) ListNetworks(ctx context.Context, projectID, availabilityZone string, limit int, marker string) ([]Network, error) {
	query := map[string]string{
		"project_id":        projectID,
		"availability_zone": availabilityZone,
		"marker":            marker,
	}
	if limit > 0 {
		query["limit"] = strconv.Itoa(limit)
	}

	var out networksResponse
	var raw []byte
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: networkList, Query: query, Result: &out, RawBody: &raw}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, networkList, out.Status, raw); err != nil {
		return nil, err
	}

	return out.Data.Networks, nil
}
