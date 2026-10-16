// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type FirewallGroup struct {
	ID                      string   `json:"id"`
	Name                    *string  `json:"name"`
	Description             *string  `json:"description"`
	ProjectID               *string  `json:"project_id"`
	Tags                    []string `json:"tags"`
	AdminStateUp            *bool    `json:"admin_state_up"`
	IngressFirewallPolicyID *string  `json:"ingress_firewall_policy_id"`
	EgressFirewallPolicyID  *string  `json:"egress_firewall_policy_id"`
	Ports                   []string `json:"ports"`
}

type FirewallGroupCreate struct {
	Name                    string   `json:"name"`
	IngressFirewallPolicyID *string  `json:"ingress_firewall_policy_id,omitempty"`
	EgressFirewallPolicyID  *string  `json:"egress_firewall_policy_id,omitempty"`
	Ports                   []string `json:"ports,omitempty"`
	AdminStateUp            *bool    `json:"admin_state_up,omitempty"`
	Description             *string  `json:"description,omitempty"`
	Tags                    []string `json:"tags,omitempty"`
}

type FirewallGroupUpdate struct {
	Name                    *string  `json:"name,omitempty"`
	IngressFirewallPolicyID *string  `json:"ingress_firewall_policy_id,omitempty"`
	EgressFirewallPolicyID  *string  `json:"egress_firewall_policy_id,omitempty"`
	Ports                   []string `json:"ports,omitempty"`
	AdminStateUp            *bool    `json:"admin_state_up,omitempty"`
	Description             *string  `json:"description,omitempty"`
	Tags                    []string `json:"tags,omitempty"`
}

func (c *Client) CreateFirewallGroup(ctx context.Context, p, z string, v FirewallGroupCreate) (*FirewallGroup, error) {
	body := map[string]any{"project_id": p, "availability_zone": z, "firewall_group": v}
	var d struct {
		FirewallGroup FirewallGroup `json:"firewall_group"`
	}
	err := c.networkObject(ctx, http.MethodPost, firewallGroupsPath, nil, body, &d)
	return &d.FirewallGroup, err
}
func (c *Client) GetFirewallGroup(ctx context.Context, p, z, id string) (*FirewallGroup, error) {
	var d struct {
		FirewallGroup FirewallGroup `json:"firewall_group"`
	}
	err := c.networkObject(ctx, http.MethodGet, firewallGroupsPath, networkQuery(p, z, "firewall_group_id", id), nil, &d)
	return &d.FirewallGroup, err
}
func (c *Client) UpdateFirewallGroup(ctx context.Context, p, z, id string, v FirewallGroupUpdate) (*FirewallGroup, error) {
	var d struct {
		FirewallGroup FirewallGroup `json:"firewall_group"`
	}
	err := c.networkObject(ctx, http.MethodPut, firewallGroupsPath, nil, updateEnvelope(p, z, "firewall_group_id", id, "firewall_group", v), &d)
	return &d.FirewallGroup, err
}
func (c *Client) DeleteFirewallGroup(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, firewallGroupsPath, networkQuery(p, z, "firewall_group_id", id), nil, nil)
}

func (c *Client) ListFirewallGroups(ctx context.Context, p, z string) ([]FirewallGroup, error) {
	var d struct {
		Items []FirewallGroup `json:"firewall_groups"`
	}
	e := c.listNetworkObjects(ctx, firewallGroupListPath, p, z, &d)
	return d.Items, e
}
