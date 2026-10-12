// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
)

const (
	kubernetesPath           = "/api/v3/k8s"
	kubernetesDetailPath     = "/api/v3/k8s/detail"
	kubernetesStatusPath     = "/api/v3/k8s/status"
	kubernetesBastionPath    = "/api/v3/k8s/bastion"
	kubernetesLBPath         = "/api/v3/k8s/lb"
	kubernetesMonitoringPath = "/api/v3/k8s/monitoring"
	kubernetesNodesPath      = "/api/v3/k8s/nodes"
	kubernetesMigratePath    = "/api/v3/k8s/migrate"
	kubernetesScalePath      = "/api/v3/k8s/scale"
	kubernetesVerticalPath   = "/api/v3/k8s/scale/vertical"
	kubernetesEtcdVolumePath = "/api/v3/k8s/scale/etcd-volume"
)

type KubernetesClusterCreateRequest struct {
	ClusterName       string `json:"cluster_name"`
	AvailabilityZone  string `json:"availability_zone"`
	ProjectID         string `json:"project_id"`
	KubernetesVersion string `json:"k8s_version"`
	RouterID          string `json:"router_id,omitempty"`
	MasterCount       int64  `json:"number_of_masters"`
	WorkerCount       int64  `json:"number_of_workers"`
	EtcdVolumeSizeGB  int64  `json:"etcd_volume_size_gb"`
	MasterFlavorID    string `json:"master_flavor_id,omitempty"`
	WorkerFlavorID    string `json:"worker_flavor_id,omitempty"`
	ImageID           string `json:"image_id,omitempty"`
}

type KubernetesCluster struct {
	ClusterID           string          `json:"cluster_id"`
	ClusterName         string          `json:"cluster_name"`
	KubernetesVersion   string          `json:"k8s_version"`
	PodNetworkCIDR      string          `json:"pod_network_cidr"`
	MasterCount         int64           `json:"number_of_masters"`
	WorkerCount         int64           `json:"number_of_workers"`
	EtcdVolumeSizeGB    int64           `json:"etcd_volume_size_gb"`
	MasterFlavorID      string          `json:"master_flavor_id"`
	WorkerFlavorID      string          `json:"worker_flavor_id"`
	ImageID             string          `json:"image_id"`
	CNIPlugin           string          `json:"cni_plugin"`
	Status              string          `json:"status"`
	CreatedAt           string          `json:"created_at"`
	UpdatedAt           string          `json:"updated_at"`
	HAProxyLBPublicIP   string          `json:"haproxy_lb_public_ip"`
	HAProxyLBPrivateIP  string          `json:"haproxy_lb_private_ip"`
	HAProxyLBInstanceID string          `json:"haproxy_lb_instance_id"`
	BastionPublicIP     string          `json:"bastion_public_ip"`
	BastionInstanceID   string          `json:"bastion_instance_id"`
	HAProxyCPPrivateIP  string          `json:"haproxy_cp_private_ip"`
	HAProxyCPInstanceID string          `json:"haproxy_cp_instance_id"`
	KubeAPIVIP          string          `json:"kube_api_vip"`
	NFSPrivateIP        string          `json:"nfs_private_ip"`
	NFSInstanceID       string          `json:"nfs_instance_id"`
	NetworkID           string          `json:"network_id"`
	SubnetID            string          `json:"subnet_id"`
	Nodes               json.RawMessage `json:"nodes"`
	Masters             json.RawMessage `json:"masters"`
	Workers             json.RawMessage `json:"workers"`
	GrafanaURL          string          `json:"grafana_url"`
	GrafanaPassword     string          `json:"grafana_password"`
}

type KubernetesCreateResult struct {
	TaskID  string
	Cluster KubernetesCluster
}

type KubernetesTaskStatus struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

type KubernetesBastion struct {
	ClusterID string `json:"cluster_id"`
	IP        string `json:"bastion_ip"`
	Username  string `json:"bastion_username"`
	Password  string `json:"bastion_password"`
}

type KubernetesLoadBalancerLogin struct {
	ClusterID string `json:"cluster_id"`
	IP        string `json:"lb_ip"`
	Username  string `json:"lb_username"`
	Password  string `json:"lb_password"`
}

type KubernetesMonitoring struct {
	ClusterID       string `json:"cluster_id"`
	GrafanaURL      string `json:"grafana_url"`
	GrafanaPassword string `json:"grafana_password"`
}

type KubernetesNodes struct {
	ClusterID   string          `json:"cluster_id"`
	ClusterName string          `json:"cluster_name"`
	Masters     json.RawMessage `json:"masters"`
	Workers     json.RawMessage `json:"workers"`
}

func (c *Client) CreateKubernetesCluster(ctx context.Context, request KubernetesClusterCreateRequest) (*KubernetesCreateResult, error) {
	var raw []byte
	var response struct {
		Status bool              `json:"status"`
		TaskID string            `json:"task_id"`
		Data   KubernetesCluster `json:"data"`
	}
	if err := c.Do(ctx, Options{Method: http.MethodPost, Path: kubernetesPath, Body: request, Result: &response, RawBody: &raw}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodPost, kubernetesPath, response.Status, raw); err != nil {
		return nil, err
	}
	return &KubernetesCreateResult{TaskID: response.TaskID, Cluster: response.Data}, nil
}

func (c *Client) GetKubernetesCluster(ctx context.Context, projectID, availabilityZone, clusterID string) (*KubernetesCluster, error) {
	query := fosQuery(projectID, availabilityZone)
	query["cluster_id"] = clusterID
	var raw []byte
	var response struct {
		Status bool               `json:"status"`
		Data   *KubernetesCluster `json:"data"`
	}
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: kubernetesDetailPath, Query: query, Result: &response, RawBody: &raw}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, kubernetesDetailPath, response.Status, raw); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (c *Client) ListKubernetesClusters(ctx context.Context, projectID, availabilityZone string) ([]KubernetesCluster, error) {
	var clusters []KubernetesCluster
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: kubernetesPath, Query: fosQuery(projectID, availabilityZone), Result: &clusters}); err != nil {
		return nil, err
	}
	return clusters, nil
}

func (c *Client) DeleteKubernetesCluster(ctx context.Context, projectID, availabilityZone, clusterID string) error {
	body := map[string]string{"project_id": projectID, "availability_zone": availabilityZone, "cluster_id": clusterID}
	var raw []byte
	var response fosMessageResponse
	if err := c.Do(ctx, Options{Method: http.MethodDelete, Path: kubernetesPath, Body: body, Result: &response, RawBody: &raw}); err != nil {
		return err
	}
	return checkFOSMessage(http.MethodDelete, kubernetesPath, response, raw)
}

func (c *Client) ScaleKubernetesCluster(ctx context.Context, projectID, availabilityZone, clusterID, nodeType string, targetCount int64) error {
	body := map[string]any{"project_id": projectID, "availability_zone": availabilityZone, "cluster_id": clusterID, "node_type": nodeType, "target_count": targetCount}
	return c.kubernetesMutation(ctx, kubernetesScalePath, body)
}

func (c *Client) VerticallyScaleKubernetesCluster(ctx context.Context, projectID, availabilityZone, clusterID, nodeType, flavorID string, instanceIDs []string) error {
	body := map[string]any{"project_id": projectID, "availability_zone": availabilityZone, "cluster_id": clusterID, "node_type": nodeType, "flavor_id": flavorID}
	if len(instanceIDs) > 0 {
		body["instance_ids"] = instanceIDs
	}
	return c.kubernetesMutation(ctx, kubernetesVerticalPath, body)
}

func (c *Client) ResizeKubernetesEtcdVolume(ctx context.Context, projectID, availabilityZone, clusterID string, size int64) error {
	body := map[string]any{"project_id": projectID, "availability_zone": availabilityZone, "cluster_id": clusterID, "new_size_gb": size}
	return c.kubernetesMutation(ctx, kubernetesEtcdVolumePath, body)
}

func (c *Client) kubernetesMutation(ctx context.Context, path string, body any) error {
	var raw []byte
	var response fosMessageResponse
	if err := c.Do(ctx, Options{Method: http.MethodPost, Path: path, Body: body, Result: &response, RawBody: &raw}); err != nil {
		return err
	}
	return checkFOSMessage(http.MethodPost, path, response, raw)
}

func (c *Client) GetKubernetesTaskStatus(ctx context.Context, taskID string) (*KubernetesTaskStatus, error) {
	var response KubernetesTaskStatus
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: kubernetesStatusPath, Query: map[string]string{"task_id": taskID}, Result: &response}); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetKubernetesBastion(ctx context.Context, projectID, availabilityZone, clusterID string) (*KubernetesBastion, error) {
	var response KubernetesBastion
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: kubernetesBastionPath, Query: kubernetesDetailQuery(projectID, availabilityZone, clusterID), Result: &response}); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetKubernetesLoadBalancerLogin(ctx context.Context, projectID, availabilityZone, clusterID string) (*KubernetesLoadBalancerLogin, error) {
	var response KubernetesLoadBalancerLogin
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: kubernetesLBPath, Query: kubernetesDetailQuery(projectID, availabilityZone, clusterID), Result: &response}); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetKubernetesMonitoring(ctx context.Context, projectID, availabilityZone, clusterID string) (*KubernetesMonitoring, error) {
	var response KubernetesMonitoring
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: kubernetesMonitoringPath, Query: kubernetesDetailQuery(projectID, availabilityZone, clusterID), Result: &response}); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetKubernetesNodes(ctx context.Context, projectID, availabilityZone, clusterID string) (*KubernetesNodes, error) {
	var response KubernetesNodes
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: kubernetesNodesPath, Query: kubernetesDetailQuery(projectID, availabilityZone, clusterID), Result: &response}); err != nil {
		return nil, err
	}
	return &response, nil
}

func kubernetesDetailQuery(projectID, availabilityZone, clusterID string) map[string]string {
	query := fosQuery(projectID, availabilityZone)
	query["cluster_id"] = clusterID
	return query
}

func (c *Client) MigrateKubernetesCluster(ctx context.Context, payload json.RawMessage) error {
	var body any
	if err := json.Unmarshal(payload, &body); err != nil {
		return err
	}
	return c.kubernetesMutation(ctx, kubernetesMigratePath, body)
}
