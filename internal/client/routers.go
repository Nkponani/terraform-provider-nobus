// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type Router struct {
	ID           string               `json:"id"`
	Name         *string              `json:"name"`
	Description  *string              `json:"description"`
	ProjectID    *string              `json:"project_id"`
	AdminStateUp *bool                `json:"admin_state_up"`
	Status       *string              `json:"status"`
	Tags         []string             `json:"tags"`
	Gateway      *ExternalGatewayInfo `json:"external_gateway_info"`
}

type RouterCreate struct {
	Name         string               `json:"name"`
	Description  *string              `json:"description,omitempty"`
	AdminStateUp *bool                `json:"admin_state_up,omitempty"`
	Tags         []string             `json:"tags,omitempty"`
	Gateway      *ExternalGatewayInfo `json:"external_gateway_info,omitempty"`
}
type ExternalFixedIP struct {
	IPAddress string `json:"ip_address"`
}

type ExternalGatewayInfo struct {
	NetworkID        string            `json:"network_id"`
	EnableSNAT       *bool             `json:"enable_snat,omitempty"`
	ExternalFixedIPs []ExternalFixedIP `json:"external_fixed_ips,omitempty"`
}
type RouterUpdate struct {
	Name                *string              `json:"name,omitempty"`
	AdminStateUp        *bool                `json:"admin_state_up,omitempty"`
	Description         *string              `json:"description,omitempty"`
	ExternalGatewayInfo *ExternalGatewayInfo `json:"external_gateway_info,omitempty"`
	Tags                []string             `json:"tags,omitempty"`
}

func (c *Client) CreateRouter(ctx context.Context, p, z string, v RouterCreate) (*Router, error) {
	body := struct {
		ProjectID        string       `json:"project_id"`
		AvailabilityZone string       `json:"availability_zone"`
		Router           RouterCreate `json:"router"`
	}{p, z, v}
	var d struct {
		Router Router `json:"router"`
	}
	e := c.networkObject(ctx, http.MethodPost, routersPath, nil, body, &d)
	return &d.Router, e
}
func (c *Client) GetRouter(ctx context.Context, p, z, id string) (*Router, error) {
	var d struct {
		Router Router `json:"router"`
	}
	e := c.networkObject(ctx, http.MethodGet, routersPath, networkQuery(p, z, "router_id", id), nil, &d)
	return &d.Router, e
}
func (c *Client) DeleteRouter(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, routersPath, networkQuery(p, z, "router_id", id), nil, nil)
}
func (c *Client) SetRouterInterface(ctx context.Context, p, z, routerID, subnetID, portID string, add bool) error {
	q := networkQuery(p, z, "router_id", routerID)
	q["subnet_id"] = subnetID
	q["port_id"] = portID
	if add {
		return c.networkObject(ctx, http.MethodPost, routerAddInterfacePath, q, nil, nil)
	}
	return c.networkObject(ctx, http.MethodDelete, routerDropInterfacePath, q, nil, nil)
}
func (c *Client) UpdateRouter(ctx context.Context, p, z, id string, v RouterUpdate) (*Router, error) {
	var d struct {
		Router Router `json:"router"`
	}
	err := c.networkObject(ctx, http.MethodPut, routersPath, nil, updateEnvelope(p, z, "router_id", id, "router", v), &d)
	return &d.Router, err
}
func (c *Client) ClearRouterGateway(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, routerClearGatewayPath, networkQuery(p, z, "router_id", id), nil, nil)
}
func (c *Client) ListRouters(ctx context.Context, p, z string) ([]Router, error) {
	var d struct {
		Items []Router `json:"routers"`
	}
	e := c.listNetworkObjects(ctx, routerListPath, "routers", p, z, &d)
	return d.Items, e
}
