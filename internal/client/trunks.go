// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
)

type TrunkSubPort struct {
	PortID           string `json:"port_id"`
	SegmentationType string `json:"segmentation_type"`
	SegmentationID   int64  `json:"segmentation_id"`
}

type Trunk struct {
	ID          string         `json:"id"`
	Name        *string        `json:"name"`
	Description *string        `json:"description"`
	ProjectID   *string        `json:"project_id"`
	Tags        []string       `json:"tags"`
	PortID      string         `json:"port_id"`
	SubPorts    []TrunkSubPort `json:"sub_ports"`
	Status      *string        `json:"status"`
}

type TrunkCreate struct {
	Name        string         `json:"name"`
	PortID      string         `json:"port_id"`
	SubPorts    []TrunkSubPort `json:"sub_ports,omitempty"`
	Description *string        `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
}

type TrunkUpdate struct {
	Name        *string        `json:"name,omitempty"`
	SubPorts    []TrunkSubPort `json:"sub_ports,omitempty"`
	Description *string        `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
}

func (c *Client) CreateTrunk(ctx context.Context, p, z string, v TrunkCreate) (*Trunk, error) {
	body := map[string]any{"project_id": p, "availability_zone": z, "trunk": v}
	var d struct {
		Trunk Trunk `json:"trunk"`
	}
	err := c.networkObject(ctx, http.MethodPost, trunksPath, nil, body, &d)
	return &d.Trunk, err
}
func (c *Client) GetTrunk(ctx context.Context, p, z, id string) (*Trunk, error) {
	var d struct {
		Trunk Trunk `json:"trunk"`
	}
	err := c.networkObject(ctx, http.MethodGet, trunksPath, networkQuery(p, z, "trunk_id", id), nil, &d)
	return &d.Trunk, err
}
func (c *Client) UpdateTrunk(ctx context.Context, p, z, id string, v TrunkUpdate) (*Trunk, error) {
	var d struct {
		Trunk Trunk `json:"trunk"`
	}
	err := c.networkObject(ctx, http.MethodPut, trunksPath, nil, updateEnvelope(p, z, "trunk_id", id, "trunk", v), &d)
	return &d.Trunk, err
}
func (c *Client) DeleteTrunk(ctx context.Context, p, z, id string) error {
	return c.networkObject(ctx, http.MethodDelete, trunksPath, networkQuery(p, z, "trunk_id", id), nil, nil)
}

func (c *Client) ListTrunks(ctx context.Context, p, z string) ([]Trunk, error) {
	var d struct {
		Items []Trunk `json:"trunks"`
	}
	e := c.listNetworkObjects(ctx, trunkListPath, "trunks", p, z, &d)
	return d.Items, e
}
