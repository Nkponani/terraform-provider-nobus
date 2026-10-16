// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type SecurityGroupRule struct {
	ID              string  `json:"id"`
	Description     *string `json:"description"`
	ProjectID       *string `json:"project_id"`
	SecurityGroupID string  `json:"security_group_id"`
	Direction       string  `json:"direction"`
	EtherType       string  `json:"ethertype"`
	Protocol        *string `json:"protocol"`
	PortRangeMin    *int64  `json:"port_range_min"`
	PortRangeMax    *int64  `json:"port_range_max"`
	RemoteIPPrefix  *string `json:"remote_ip_prefix"`
	RemoteGroupID   *string `json:"remote_group_id"`
}
type SecurityGroupRuleCreate struct {
	SecurityGroupID string  `json:"security_group_id"`
	Direction       string  `json:"direction"`
	EtherType       string  `json:"ethertype"`
	Protocol        *string `json:"protocol,omitempty"`
	PortRangeMin    *int64  `json:"port_range_min,omitempty"`
	PortRangeMax    *int64  `json:"port_range_max,omitempty"`
	RemoteIPPrefix  *string `json:"remote_ip_prefix,omitempty"`
	RemoteGroupID   *string `json:"remote_group_id,omitempty"`
	Description     *string `json:"description,omitempty"`
}

func (c *Client) CreateSecurityGroupRule(ctx context.Context, p, z string, v SecurityGroupRuleCreate) (*SecurityGroupRule, error) {
	body := struct {
		ProjectID        string                  `json:"project_id"`
		AvailabilityZone string                  `json:"availability_zone"`
		Rule             SecurityGroupRuleCreate `json:"security_group_rule"`
	}{p, z, v}
	var d struct {
		Rule SecurityGroupRule `json:"security_group_rule"`
	}
	e := c.networkObject(ctx, http.MethodPost, securityGroupRulesPath, nil, body, &d)
	return &d.Rule, e
}
func (c *Client) GetSecurityGroupRule(ctx context.Context, p, z, id string) (*SecurityGroupRule, error) {
	var d struct {
		Rule SecurityGroupRule `json:"security_group_rule"`
	}
	e := c.networkObject(ctx, http.MethodGet, securityGroupRulesPath, networkQuery(p, z, "security_group_rule_id", id), nil, &d)
	return &d.Rule, e
}
func (c *Client) DeleteSecurityGroupRule(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, securityGroupRulesPath, networkQuery(p, z, "security_group_rule_id", id), nil, nil)
}
func (c *Client) ListSecurityGroupRules(ctx context.Context, p, z string) ([]SecurityGroupRule, error) {
	var d struct {
		Items []SecurityGroupRule `json:"security_group_rules"`
	}
	e := c.listNetworkObjects(ctx, securityGroupRuleListPath, p, z, &d)
	return d.Items, e
}
