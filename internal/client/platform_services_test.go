// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKubernetesRequests(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case kubernetesPath:
			if r.Method == http.MethodPost {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["cluster_name"] != "cluster-a" || body["project_id"] != "project-1" {
					t.Errorf("unexpected create body: %v", body)
				}
				_, _ = w.Write([]byte(`{"status":true,"task_id":"task-1","data":{"cluster_id":"cluster-1","cluster_name":"cluster-a","status":"creating"}}`))
				return
			}
		case kubernetesDetailPath:
			if r.URL.Query().Get("cluster_id") != "cluster-1" || r.URL.Query().Get("project_id") != "project-1" {
				t.Errorf("unexpected detail query: %v", r.URL.Query())
			}
			_, _ = w.Write([]byte(`{"status":true,"data":{"cluster_id":"cluster-1","cluster_name":"cluster-a","number_of_masters":2,"number_of_workers":3,"status":"running"}}`))
			return
		case kubernetesScalePath:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["node_type"] != "worker" || body["target_count"] != float64(4) {
				t.Errorf("unexpected scale body: %v", body)
			}
			_, _ = w.Write([]byte(`{"status":true,"message":"accepted"}`))
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()
	c := newTestClient(t, server)
	created, err := c.CreateKubernetesCluster(context.Background(), KubernetesClusterCreateRequest{ClusterName: "cluster-a", ProjectID: "project-1", AvailabilityZone: "zone-1", MasterCount: 2, WorkerCount: 3, EtcdVolumeSizeGB: 50})
	if err != nil {
		t.Fatal(err)
	}
	if created.TaskID != "task-1" || created.Cluster.ClusterID != "cluster-1" {
		t.Fatalf("unexpected create result: %+v", created)
	}
	cluster, err := c.GetKubernetesCluster(context.Background(), "project-1", "zone-1", "cluster-1")
	if err != nil {
		t.Fatal(err)
	}
	if cluster.WorkerCount != 3 || cluster.Status != "running" {
		t.Fatalf("unexpected cluster: %+v", cluster)
	}
	if err := c.ScaleKubernetesCluster(context.Background(), "project-1", "zone-1", "cluster-1", "worker", 4); err != nil {
		t.Fatal(err)
	}
}

func TestDatabaseRequests(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case databasePath:
			if r.Method == http.MethodPost {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["database_name"] != "orders" || body["customer_user_password"] != "secret" {
					t.Errorf("unexpected database body: %v", body)
				}
				_, _ = w.Write([]byte(`{"status":true,"data":{"database_id":"db-1","task_id":"task-db"}}`))
				return
			}
			if r.URL.Query().Get("database_id") != "db-1" {
				t.Errorf("unexpected database query: %v", r.URL.Query())
			}
			_, _ = w.Write([]byte(`{"database_id":"db-1","database_name":"orders","database_type":"postgres","database_version":"16","installation_type":"single","project_id":"project-1","created_at":"now","status":"active","instance_count":1,"instances":[]}`))
			return
		case databaseSizePath:
			if r.URL.Query().Get("new_size") != "100" {
				t.Errorf("unexpected resize query: %v", r.URL.Query())
			}
			_, _ = w.Write([]byte(`{"status":"accepted"}`))
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()
	c := newTestClient(t, server)
	created, err := c.CreateDatabase(context.Background(), DatabaseCreateRequest{ProjectID: "project-1", AvailabilityZone: "zone-1", DatabaseName: "orders", CustomerPassword: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if created.DatabaseID != "db-1" || created.TaskID != "task-db" {
		t.Fatalf("unexpected create result: %+v", created)
	}
	database, err := c.GetDatabase(context.Background(), "project-1", "zone-1", "db-1")
	if err != nil {
		t.Fatal(err)
	}
	if database.DatabaseName != "orders" || database.InstanceCount != 1 {
		t.Fatalf("unexpected database: %+v", database)
	}
	if err := c.ResizeDatabase(context.Background(), "project-1", "zone-1", "db-1", 100); err != nil {
		t.Fatal(err)
	}
}

func TestLBaaSRequests(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v3/lbaas/loadbalancers":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["name"] != "edge" || body["project_id"] != "project-1" {
				t.Errorf("unexpected LB body: %v", body)
			}
			_, _ = w.Write([]byte(`{"status":true,"message":"created","data":{"loadbalancer":{"id":"lb-1","name":"edge","admin_state_up":true,"provisioning_status":"PENDING","operating_status":"OFFLINE","tags":[],"provider":"haproxy","ssh_username":"ubuntu"},"task_id":"task-lb","state":"PENDING"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/lbaas/listeners":
			if r.URL.Query().Get("loadbalancer_id") != "lb-1" {
				t.Errorf("unexpected list query: %v", r.URL.Query())
			}
			_, _ = w.Write([]byte(`{"status":true,"message":"ok","data":[{"id":"listener-1"}]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v3/lbaas/pools/members/member-1":
			if r.URL.Query().Get("pool_id") != "pool-1" {
				t.Errorf("unexpected member query: %v", r.URL.Query())
			}
			_, _ = w.Write([]byte(`{"status":true,"message":"updated","data":{"member":{"id":"member-1","pool":"pool-1","address":"10.0.0.5","protocol_port":80,"weight":2,"backup":false,"admin_state_up":true,"provisioning_status":"ACTIVE","operating_status":"ONLINE","tags":[]},"haproxy_config":""}}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	c := newTestClient(t, server)
	created, err := c.CreateLBaaSResource(context.Background(), LBaaSLoadBalancers, "project-1", "zone-1", map[string]any{"name": "edge", "project_id": "project-1"})
	if err != nil {
		t.Fatal(err)
	}
	if created.TaskID != "task-lb" {
		t.Fatalf("unexpected LB result: %+v", created)
	}
	var lb map[string]any
	if err := json.Unmarshal(created.Resource, &lb); err != nil || lb["id"] != "lb-1" {
		t.Fatalf("unexpected LB resource: %s (%v)", created.Resource, err)
	}
	listed, err := c.ListLBaaSResources(context.Background(), LBaaSListeners, "project-1", "zone-1", map[string]string{"loadbalancer_id": "lb-1"})
	if err != nil {
		t.Fatal(err)
	}
	var listeners []map[string]any
	if err := json.Unmarshal(listed, &listeners); err != nil || len(listeners) != 1 {
		t.Fatalf("unexpected listeners: %s", listed)
	}
	updated, err := c.UpdateLBaaSResource(context.Background(), LBaaSMembers, "project-1", "zone-1", "member-1", map[string]string{"pool_id": "pool-1"}, map[string]any{"weight": 2})
	if err != nil {
		t.Fatal(err)
	}
	var member map[string]any
	_ = json.Unmarshal(updated, &member)
	if member["id"] != "member-1" {
		t.Fatalf("unexpected member: %s", updated)
	}
}
