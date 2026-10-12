// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type LBaaSCollection string

const (
	LBaaSLoadBalancers  LBaaSCollection = "loadbalancers"
	LBaaSListeners      LBaaSCollection = "listeners"
	LBaaSPools          LBaaSCollection = "pools"
	LBaaSMembers        LBaaSCollection = "members"
	LBaaSHealthMonitors LBaaSCollection = "healthmonitors"
	LBaaSL7Policies     LBaaSCollection = "l7policies"
	LBaaSL7Rules        LBaaSCollection = "l7rules"
)

type lbaasDescriptor struct {
	collectionPath string
	createPath     string
	detailPath     string
	idParameter    string
	resultKey      string
}

var lbaasDescriptors = map[LBaaSCollection]lbaasDescriptor{
	LBaaSLoadBalancers:  {"/api/v3/lbaas/loadbalancers", "/api/v3/lbaas/loadbalancers", "/api/v3/lbaas/loadbalancers/detail", "loadbalancer_id", "loadbalancer"},
	LBaaSListeners:      {"/api/v3/lbaas/listeners", "/api/v3/lbaas/listeners", "/api/v3/lbaas/listeners/detail", "listener_id", "listener"},
	LBaaSPools:          {"/api/v3/lbaas/pools", "/api/v3/lbaas/pools", "/api/v3/lbaas/pools/detail", "pool_id", "pool"},
	LBaaSMembers:        {"/api/v3/lbaas/pools/members", "/api/v3/lbaas/members", "/api/v3/lbaas/pools/members/detail", "member_id", "member"},
	LBaaSHealthMonitors: {"/api/v3/lbaas/healthmonitors", "/api/v3/lbaas/healthmonitors", "/api/v3/lbaas/healthmonitors/detail", "healthmonitor_id", "healthmonitor"},
	LBaaSL7Policies:     {"/api/v3/lbaas/l7policies", "/api/v3/lbaas/l7policies", "/api/v3/lbaas/l7policies/detail", "l7policy_id", "l7policy"},
	LBaaSL7Rules:        {"/api/v3/lbaas/l7policies/rules", "/api/v3/lbaas/l7rules", "/api/v3/lbaas/l7policies/rules/detail", "l7rule_id", "l7rule"},
}

type LBaaSCreateResult struct {
	Resource json.RawMessage
	TaskID   string
	State    string
}

func (c *Client) CreateLBaaSResource(ctx context.Context, collection LBaaSCollection, projectID, availabilityZone string, body any) (*LBaaSCreateResult, error) {
	descriptor, err := lbaasDescriptorFor(collection)
	if err != nil {
		return nil, err
	}
	query := fosQuery(projectID, availabilityZone)
	if collection == LBaaSLoadBalancers {
		query = nil
	}
	data, err := c.lbaasEnvelope(ctx, Options{Method: http.MethodPost, Path: descriptor.createPath, Query: query, Body: body})
	if err != nil {
		return nil, err
	}
	resource, taskID, state := unwrapLBaaSResource(data, descriptor.resultKey)
	return &LBaaSCreateResult{Resource: resource, TaskID: taskID, State: state}, nil
}

func (c *Client) GetLBaaSResource(ctx context.Context, collection LBaaSCollection, projectID, availabilityZone, resourceID string, parents map[string]string) (json.RawMessage, error) {
	descriptor, err := lbaasDescriptorFor(collection)
	if err != nil {
		return nil, err
	}
	query := lbaasQuery(projectID, availabilityZone, parents)
	query[descriptor.idParameter] = resourceID
	data, err := c.lbaasEnvelope(ctx, Options{Method: http.MethodGet, Path: descriptor.detailPath, Query: query})
	if err != nil {
		return nil, err
	}
	resource, _, _ := unwrapLBaaSResource(data, descriptor.resultKey)
	return resource, nil
}

func (c *Client) ListLBaaSResources(ctx context.Context, collection LBaaSCollection, projectID, availabilityZone string, filters map[string]string) (json.RawMessage, error) {
	descriptor, err := lbaasDescriptorFor(collection)
	if err != nil {
		return nil, err
	}
	return c.lbaasEnvelope(ctx, Options{Method: http.MethodGet, Path: descriptor.collectionPath, Query: lbaasQuery(projectID, availabilityZone, filters)})
}

func (c *Client) UpdateLBaaSResource(ctx context.Context, collection LBaaSCollection, projectID, availabilityZone, resourceID string, parents map[string]string, body any) (json.RawMessage, error) {
	descriptor, err := lbaasDescriptorFor(collection)
	if err != nil {
		return nil, err
	}
	query := lbaasQuery(projectID, availabilityZone, parents)
	query[descriptor.idParameter] = resourceID
	path := descriptor.collectionPath
	if collection == LBaaSMembers {
		path += "/" + url.PathEscape(resourceID)
		delete(query, descriptor.idParameter)
	}
	data, err := c.lbaasEnvelope(ctx, Options{Method: http.MethodPatch, Path: path, Query: query, Body: body})
	if err != nil {
		return nil, err
	}
	resource, _, _ := unwrapLBaaSResource(data, descriptor.resultKey)
	return resource, nil
}

func (c *Client) DeleteLBaaSResource(ctx context.Context, collection LBaaSCollection, projectID, availabilityZone, resourceID string, parents map[string]string) error {
	descriptor, err := lbaasDescriptorFor(collection)
	if err != nil {
		return err
	}
	query := lbaasQuery(projectID, availabilityZone, parents)
	query[descriptor.idParameter] = resourceID
	_, err = c.lbaasEnvelope(ctx, Options{Method: http.MethodDelete, Path: descriptor.collectionPath, Query: query})
	return err
}

func (c *Client) UpdateLBaaSSecurityGroups(ctx context.Context, projectID, availabilityZone, loadBalancerID string, names []string) error {
	_, err := c.lbaasEnvelope(ctx, Options{Method: http.MethodPatch, Path: "/api/v3/lbaas/loadbalancers/security-groups", Query: lbaasQuery(projectID, availabilityZone, map[string]string{"loadbalancer_id": loadBalancerID}), Body: map[string]any{"security_group_names": names}})
	return err
}

func (c *Client) UpdateLBaaSSessionPersistence(ctx context.Context, projectID, availabilityZone, poolID string, persistence any) (json.RawMessage, error) {
	data, err := c.lbaasEnvelope(ctx, Options{Method: http.MethodPatch, Path: "/api/v3/lbaas/pools/session-persistence", Query: lbaasQuery(projectID, availabilityZone, map[string]string{"pool_id": poolID}), Body: map[string]any{"session_persistence": persistence}})
	if err != nil {
		return nil, err
	}
	resource, _, _ := unwrapLBaaSResource(data, "pool")
	return resource, nil
}

func (c *Client) FailoverLBaaSLoadBalancer(ctx context.Context, projectID, availabilityZone, loadBalancerID string) error {
	_, err := c.lbaasEnvelope(ctx, Options{Method: http.MethodPost, Path: "/api/v3/lbaas/loadbalancers/failover", Query: lbaasQuery(projectID, availabilityZone, map[string]string{"loadbalancer_id": loadBalancerID})})
	return err
}

func (c *Client) GetLBaaSOperation(ctx context.Context, operation, projectID, availabilityZone string, query map[string]string) (json.RawMessage, error) {
	paths := map[string]string{
		"task": "/api/v3/lbaas/tasks", "deployment_status": "/api/v3/lbaas/loadbalancers/deployment-status",
		"loadbalancer_status": "/api/v3/lbaas/loadbalancers/status", "loadbalancer_stats": "/api/v3/lbaas/loadbalancers/stats", "listener_stats": "/api/v3/lbaas/listeners/stats",
	}
	path, ok := paths[operation]
	if !ok {
		return nil, fmt.Errorf("unsupported LBaaS operation %q", operation)
	}
	return c.lbaasEnvelope(ctx, Options{Method: http.MethodGet, Path: path, Query: lbaasQuery(projectID, availabilityZone, query)})
}

func (c *Client) ProvisionLBaaS(ctx context.Context, body json.RawMessage) (*LBaaSCreateResult, error) {
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	data, err := c.lbaasEnvelope(ctx, Options{Method: http.MethodPost, Path: "/api/v3/lbaas/provision", Body: value})
	if err != nil {
		return nil, err
	}
	resource, taskID, state := unwrapLBaaSResource(data, "loadbalancer")
	return &LBaaSCreateResult{Resource: resource, TaskID: taskID, State: state}, nil
}

func (c *Client) CreateLBaaSSecurityGroupRule(ctx context.Context, projectID, availabilityZone, loadBalancerID string, port int64, source string) (json.RawMessage, error) {
	return c.lbaasEnvelope(ctx, Options{Method: http.MethodPost, Path: "/api/v3/lbaas/loadbalancers/security-group-rules", Query: lbaasQuery(projectID, availabilityZone, map[string]string{"loadbalancer_id": loadBalancerID}), Body: map[string]any{"port": port, "from": source}})
}

func (c *Client) DeleteLBaaSSecurityGroupRule(ctx context.Context, projectID, availabilityZone, loadBalancerID string, port int64, source string) error {
	query := lbaasQuery(projectID, availabilityZone, map[string]string{"loadbalancer_id": loadBalancerID, "port": fmt.Sprintf("%d", port), "from": source})
	_, err := c.lbaasEnvelope(ctx, Options{Method: http.MethodDelete, Path: "/api/v3/lbaas/loadbalancers/security-group-rules", Query: query})
	return err
}

func (c *Client) lbaasEnvelope(ctx context.Context, options Options) (json.RawMessage, error) {
	var raw []byte
	var response struct {
		Status  bool            `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	options.Result, options.RawBody = &response, &raw
	if err := c.Do(ctx, options); err != nil {
		return nil, err
	}
	if !response.Status {
		return nil, &StatusError{Method: options.Method, Path: options.Path, Message: strings.TrimSpace(response.Message)}
	}
	return response.Data, nil
}

func lbaasDescriptorFor(collection LBaaSCollection) (lbaasDescriptor, error) {
	descriptor, ok := lbaasDescriptors[collection]
	if !ok {
		return lbaasDescriptor{}, fmt.Errorf("unsupported LBaaS collection %q", collection)
	}
	return descriptor, nil
}

func lbaasQuery(projectID, availabilityZone string, extra map[string]string) map[string]string {
	query := fosQuery(projectID, availabilityZone)
	for key, value := range extra {
		if value != "" {
			query[key] = value
		}
	}
	return query
}

func unwrapLBaaSResource(data json.RawMessage, key string) (json.RawMessage, string, string) {
	var object map[string]json.RawMessage
	if json.Unmarshal(data, &object) != nil {
		return data, "", ""
	}
	resource := data
	if nested, ok := object[key]; ok {
		resource = nested
	}
	var taskID, state string
	_ = json.Unmarshal(object["task_id"], &taskID)
	_ = json.Unmarshal(object["state"], &state)
	return resource, taskID, state
}
