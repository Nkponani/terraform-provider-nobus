// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type QosPolicy struct {
	ID          string   `json:"id"`
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	ProjectID   *string  `json:"project_id"`
	Tags        []string `json:"tags"`
	Shared      *bool    `json:"shared"`
}

type QosPolicyCreate struct {
	Name        string   `json:"name"`
	Shared      *bool    `json:"shared,omitempty"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type QosPolicyUpdate struct {
	Name        *string  `json:"name,omitempty"`
	Shared      *bool    `json:"shared,omitempty"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func (c *Client) CreateQosPolicy(ctx context.Context, p, z string, v QosPolicyCreate) (*QosPolicy, error) {
	body := map[string]any{"project_id": p, "availability_zone": z, "policy": v}
	var d struct {
		Policy QosPolicy `json:"policy"`
	}
	err := c.networkObject(ctx, http.MethodPost, qosPoliciesPath, nil, body, &d)
	return &d.Policy, err
}
func (c *Client) GetQosPolicy(ctx context.Context, p, z, id string) (*QosPolicy, error) {
	var d struct {
		Policy QosPolicy `json:"policy"`
	}
	err := c.networkObject(ctx, http.MethodGet, qosPoliciesPath, networkQuery(p, z, "qos_policy_id", id), nil, &d)
	return &d.Policy, err
}
func (c *Client) UpdateQosPolicy(ctx context.Context, p, z, id string, v QosPolicyUpdate) (*QosPolicy, error) {
	var d struct {
		Policy QosPolicy `json:"policy"`
	}
	err := c.networkObject(ctx, http.MethodPut, qosPoliciesPath, nil, updateEnvelope(p, z, "qos_policy_id", id, "policy", v), &d)
	return &d.Policy, err
}
func (c *Client) DeleteQosPolicy(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, qosPoliciesPath, networkQuery(p, z, "qos_policy_id", id), nil, nil)
}

func (c *Client) ListQosPolicies(ctx context.Context, p, z string) ([]QosPolicy, error) {
	var d struct {
		Items []QosPolicy `json:"policies"`
	}
	e := c.listNetworkObjects(ctx, qosPolicyListPath, p, z, &d)
	return d.Items, e
}
