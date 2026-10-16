// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type SecurityGroup struct {
	ID          string              `json:"id"`
	Name        *string             `json:"name"`
	Description *string             `json:"description"`
	ProjectID   *string             `json:"project_id"`
	Tags        []string            `json:"tags"`
	Rules       []SecurityGroupRule `json:"security_group_rules"`
}
type SecurityGroupCreate struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}
type SecurityGroupUpdate struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func (c *Client) CreateSecurityGroup(ctx context.Context, p, z string, v SecurityGroupCreate) (*SecurityGroup, error) {
	body := struct {
		ProjectID        string              `json:"project_id"`
		AvailabilityZone string              `json:"availability_zone"`
		SecurityGroup    SecurityGroupCreate `json:"security_group"`
	}{p, z, v}
	var d struct {
		SecurityGroup SecurityGroup `json:"security_group"`
	}
	e := c.networkObject(ctx, http.MethodPost, securityGroupsPath, nil, body, &d)
	return &d.SecurityGroup, e
}
func (c *Client) GetSecurityGroup(ctx context.Context, p, z, id string) (*SecurityGroup, error) {
	var d struct {
		SecurityGroup SecurityGroup `json:"security_group"`
	}
	e := c.networkObject(ctx, http.MethodGet, securityGroupsPath, networkQuery(p, z, "security_group_id", id), nil, &d)
	return &d.SecurityGroup, e
}
func (c *Client) DeleteSecurityGroup(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, securityGroupsPath, networkQuery(p, z, "security_group_id", id), nil, nil)
}
func (c *Client) UpdateSecurityGroup(ctx context.Context, p, z, id string, v SecurityGroupUpdate) (*SecurityGroup, error) {
	var d struct {
		SecurityGroup SecurityGroup `json:"security_group"`
	}
	err := c.networkObject(ctx, http.MethodPut, securityGroupsPath, nil, updateEnvelope(p, z, "securitygroup_id", id, "security_group", v), &d)
	return &d.SecurityGroup, err
}
func (c *Client) ListSecurityGroups(ctx context.Context, p, z string) ([]SecurityGroup, error) {
	var d struct {
		Items []SecurityGroup `json:"security_groups"`
	}
	e := c.listNetworkObjects(ctx, securityGroupListPath, p, z, &d)
	return d.Items, e
}
