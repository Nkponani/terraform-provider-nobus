// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories instantiates the provider for acceptance tests.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"nobus": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck verifies that the credentials required for acceptance testing
// are present before a test runs. Acceptance tests are skipped unless TF_ACC is
// set, so this never fails a normal `go test` run.
func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("NOBUS_API_KEY") == "" {
		t.Fatal("NOBUS_API_KEY must be set for acceptance tests")
	}
	if os.Getenv("NOBUS_PROJECT_ID") == "" {
		t.Fatal("NOBUS_PROJECT_ID must be set for acceptance tests")
	}
}

// TestProviderSchema exercises the provider schema through the plugin protocol,
// which catches invalid attribute definitions at test time.
func TestProviderSchema(t *testing.T) {
	t.Parallel()

	server, err := testAccProtoV6ProviderFactories["nobus"]()
	if err != nil {
		t.Fatalf("failed to create provider server: %s", err)
	}

	response, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("failed to read provider schema: %s", err)
	}

	provider := requireProviderSchema(t, response)

	providerAttributes := attributeNames(provider)
	for _, attribute := range []string{"api_key", "endpoint", "project_id", "availability_zone"} {
		if _, ok := providerAttributes[attribute]; !ok {
			t.Errorf("provider schema is missing the %q attribute", attribute)
		}
	}
	if _, ok := providerAttributes["request_timeout"]; ok {
		t.Error("provider must not expose request_timeout")
	}
	if len(provider.Block.BlockTypes) > 0 {
		t.Error("the provider schema must not expose nested blocks")
	}
}

// requireProviderSchema returns the provider schema, failing the test when the
// plugin reported any diagnostic.
func requireProviderSchema(t *testing.T, response *tfprotov6.GetProviderSchemaResponse) *tfprotov6.Schema {
	t.Helper()

	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("provider schema diagnostic: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
	if response.Provider == nil {
		t.Fatal("provider schema is nil")
	}

	return response.Provider
}

// TestResourceSchemas verifies that each managed resource is registered and
// that its schema is valid.
func TestResourceSchemas(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"nobus_network": {
			"id", "project_id", "availability_zone", "name", "subnet_cidr", "router_name",
			"external_network_id", "dns_nameservers", "enable_snat", "description",
			"admin_state_up", "shared", "mtu", "router_external", "port_security_enabled",
			"tags", "status", "subnet_id", "router_id", "subnet_ids",
		},
		"nobus_volume": {
			"id", "project_id", "availability_zone", "name", "description", "size",
			"volume_type", "snapshot_id", "source_volume_id", "image_ref",
			"metadata", "status", "bootable", "encrypted", "multiattach", "created_at", "updated_at",
			"attachment_ids", "server_ids",
		},
		"nobus_volume_attachment": {
			"id", "project_id", "availability_zone", "instance_id", "volume_id",
			"volume_status", "attached",
		},
		"nobus_instance": {
			"id", "project_id", "availability_zone", "name", "image", "flavor", "key_name",
			"network_id", "volume_size", "security_groups", "description", "hostname",
			"admin_pass", "user_data", "delete_volume_on_destroy", "instance_ready_states",
			"status", "addresses", "volume_ids", "flavor_id", "flavor_name", "image_id",
			"image_name", "created_at", "updated_at",
		},
		"nobus_keypair":                   {"id", "project_id", "availability_zone", "name", "public_key", "fingerprint", "user_id", "type"},
		"nobus_subnet":                    {"id", "project_id", "availability_zone", "network_id", "cidr", "ip_version", "name", "description", "gateway_ip", "enable_dhcp", "dns_nameservers"},
		"nobus_router":                    {"id", "project_id", "availability_zone", "name", "description", "admin_state_up", "status"},
		"nobus_router_interface":          {"id", "project_id", "availability_zone", "router_id", "subnet_id", "port_id"},
		"nobus_port":                      {"id", "project_id", "availability_zone", "network_id", "name", "description", "admin_state_up", "device_id", "device_owner", "security_groups", "port_security_enabled", "dns_name", "status", "mac_address"},
		"nobus_floating_ip":               {"id", "project_id", "availability_zone", "floating_network_id", "port_id", "fixed_ip_address", "description", "floating_ip_address", "status"},
		"nobus_security_group":            {"id", "project_id", "availability_zone", "name", "description"},
		"nobus_security_group_rule":       {"id", "project_id", "availability_zone", "security_group_id", "direction", "ethertype", "protocol", "port_range_min", "port_range_max", "remote_ip_prefix", "remote_group_id", "description"},
		"nobus_instance_action":           {"id", "project_id", "availability_zone", "instance_id", "action", "reboot_type", "flavor_id", "port_id", "network_id", "fixed_ip", "security_group_name", "image_name", "new_password", "tags", "tag", "result"},
		"nobus_volume_snapshot":           {"id", "project_id", "availability_zone", "volume_id", "name", "description", "force", "metadata", "status", "size", "created_at"},
		"nobus_volume_type":               {"id", "project_id", "availability_zone", "name", "description", "is_public", "extra_specs"},
		"nobus_volume_transfer":           {"id", "project_id", "availability_zone", "volume_id", "name", "created_at", "auth_key"},
		"nobus_volume_transfer_accept":    {"id", "project_id", "availability_zone", "transfer_id", "auth_key", "volume_id"},
		"nobus_trunk":                     {"id", "project_id", "availability_zone", "name", "port_id", "sub_ports", "description", "tags", "status"},
		"nobus_qos_policy":                {"id", "project_id", "availability_zone", "name", "description", "shared", "tags"},
		"nobus_firewall_group":            {"id", "project_id", "availability_zone", "name", "description", "admin_state_up", "ingress_firewall_policy_id", "egress_firewall_policy_id", "ports", "tags"},
		"nobus_standalone_network":        {"id", "project_id", "availability_zone", "name", "description", "admin_state_up", "shared", "mtu", "router_external", "port_security_enabled", "tags", "status", "subnet_ids"},
		"nobus_port_address_pairs":        {"id", "project_id", "availability_zone", "port_id", "address_pairs"},
		"nobus_fos_container":             {"id", "project_id", "availability_zone", "name", "is_public", "object_count", "bytes", "last_modified"},
		"nobus_fos_object":                {"id", "project_id", "availability_zone", "container_name", "object_name", "source", "source_hash", "bytes", "content_type", "last_modified", "etag"},
		"nobus_kubernetes_cluster":        {"id", "project_id", "availability_zone", "name", "kubernetes_version", "master_count", "worker_count", "etcd_volume_size_gb", "master_flavor_id", "worker_flavor_id", "task_id", "status"},
		"nobus_kubernetes_migration":      {"id", "payload_json"},
		"nobus_database":                  {"id", "project_id", "availability_zone", "user_email", "flavor_name", "flavor_id", "name", "database_type", "database_version", "size", "install_type", "customer_username", "customer_password", "zones", "status"},
		"nobus_database_action":           {"id", "project_id", "availability_zone", "database_id", "database_type", "action"},
		"nobus_existing_databases":        {"id", "payload_json"},
		"nobus_lbaas_load_balancer":       {"id", "project_id", "availability_zone", "name", "flavor", "load_balancer_provider", "volume_size", "security_group_names", "task_id", "vip_address"},
		"nobus_lbaas_listener":            {"id", "project_id", "availability_zone", "name", "load_balancer_id", "protocol", "protocol_port", "allow_from", "default_pool_id"},
		"nobus_lbaas_pool":                {"id", "project_id", "availability_zone", "name", "load_balancer_id", "listener_id", "protocol", "lb_algorithm", "session_persistence_type"},
		"nobus_lbaas_member":              {"id", "project_id", "availability_zone", "pool_id", "address", "protocol_port", "weight", "backup"},
		"nobus_lbaas_health_monitor":      {"id", "project_id", "availability_zone", "pool_id", "type", "delay", "timeout", "max_retries"},
		"nobus_lbaas_l7_policy":           {"id", "project_id", "availability_zone", "listener_id", "action", "redirect_pool_id", "redirect_url"},
		"nobus_lbaas_l7_rule":             {"id", "project_id", "availability_zone", "l7_policy_id", "type", "compare_type", "value", "invert"},
		"nobus_lbaas_security_group_rule": {"id", "project_id", "availability_zone", "load_balancer_id", "port", "source"},
		"nobus_lbaas_failover":            {"id", "project_id", "availability_zone", "load_balancer_id", "trigger"},
		"nobus_lbaas_provision":           {"id", "project_id", "availability_zone", "payload_json", "task_id", "state"},
	}

	server, err := testAccProtoV6ProviderFactories["nobus"]()
	if err != nil {
		t.Fatalf("failed to create provider server: %s", err)
	}

	response, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("failed to read provider schema: %s", err)
	}
	requireProviderSchema(t, response)

	for resourceName, expected := range cases {
		t.Run(resourceName, func(t *testing.T) {
			resourceSchema, ok := response.ResourceSchemas[resourceName]
			if !ok {
				t.Fatalf("resource %s is not registered", resourceName)
			}
			attributes := attributeNames(resourceSchema)
			for _, attribute := range expected {
				if _, ok := attributes[attribute]; !ok {
					t.Errorf("resource %s is missing the %q attribute", resourceName, attribute)
				}
			}
			if hasBlockType(resourceSchema, "timeouts") {
				t.Errorf("resource %s must not expose a timeouts block", resourceName)
			}
		})
	}
}

func TestDataSourceSchemas(t *testing.T) {
	t.Parallel()
	server, err := testAccProtoV6ProviderFactories["nobus"]()
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	collections := []string{"networks", "subnets", "ports", "routers", "floating_ips", "security_groups", "security_group_rules", "trunks", "qos_policies", "firewall_groups", "volumes", "volume_snapshots", "volume_backups", "volume_transfers", "volume_availability_zones"}
	for _, name := range collections {
		dsSchema, ok := response.DataSourceSchemas["nobus_"+name]
		if !ok {
			t.Errorf("data source nobus_%s is not registered", name)
			continue
		}
		attrs := attributeNames(dsSchema)
		for _, attribute := range []string{"project_id", "availability_zone", "ids", "names"} {
			if _, ok := attrs[attribute]; !ok {
				t.Errorf("nobus_%s is missing %s", name, attribute)
			}
		}
	}
	if _, ok := response.DataSourceSchemas["nobus_volume_backup"]; !ok {
		t.Error("data source nobus_volume_backup is not registered")
	}
	fosCases := map[string][]string{
		"nobus_fos_account":        {"id", "project_id", "container_count", "object_count", "bytes_used"},
		"nobus_fos_auth_details":   {"id", "project_id", "user_name", "project_name", "domain_id", "region", "identity_service_endpoint", "identity_service_version", "management_endpoint"},
		"nobus_fos_containers":     {"id", "project_id", "availability_zone", "limit", "marker", "end_marker", "prefix", "delimiter", "full_listing", "result_count", "containers"},
		"nobus_fos_objects":        {"id", "project_id", "availability_zone", "container_name", "limit", "marker", "end_marker", "prefix", "delimiter", "path", "full_listing", "result_count", "objects"},
		"nobus_fos_object_content": {"id", "project_id", "availability_zone", "container_name", "object_name", "content_base64", "bytes", "sha256"},
	}
	for name, expected := range fosCases {
		dsSchema, ok := response.DataSourceSchemas[name]
		if !ok {
			t.Errorf("data source %s is not registered", name)
			continue
		}
		attrs := attributeNames(dsSchema)
		for _, attribute := range expected {
			if _, ok := attrs[attribute]; !ok {
				t.Errorf("%s is missing %s", name, attribute)
			}
		}
	}
	additional := map[string][]string{
		"nobus_kubernetes_clusters":            {"id", "project_id", "availability_zone", "result_count", "ids", "names", "clusters_json"},
		"nobus_kubernetes_bastion":             {"id", "cluster_id", "ip", "username", "password"},
		"nobus_kubernetes_load_balancer_login": {"id", "cluster_id", "ip", "username", "password"},
		"nobus_kubernetes_monitoring":          {"id", "cluster_id", "url", "password"},
		"nobus_kubernetes_nodes":               {"id", "cluster_id", "masters_json", "workers_json"},
		"nobus_kubernetes_task_status":         {"id", "task_id", "status", "data_json"},
		"nobus_databases":                      {"id", "project_id", "availability_zone", "result_count", "ids", "names", "databases_json"},
		"nobus_database_credentials":           {"id", "database_id", "customer_name", "customer_password", "instances_json"},
		"nobus_database_creation_status":       {"id", "database_id", "result_json"},
		"nobus_database_status":                {"id", "database_id", "replica_check", "result_json"},
		"nobus_database_zabbix_credentials":    {"id", "database_id", "result_json"},
		"nobus_database_logs":                  {"id", "database_id", "database_type", "line_count", "result_json"},
		"nobus_lbaas_load_balancers":           {"id", "result_count", "result_json"},
		"nobus_lbaas_listeners":                {"id", "load_balancer_id", "result_count", "result_json"},
		"nobus_lbaas_pools":                    {"id", "listener_id", "result_count", "result_json"},
		"nobus_lbaas_members":                  {"id", "pool_id", "result_count", "result_json"},
		"nobus_lbaas_health_monitors":          {"id", "pool_id", "result_count", "result_json"},
		"nobus_lbaas_l7_policies":              {"id", "listener_id", "result_count", "result_json"},
		"nobus_lbaas_l7_rules":                 {"id", "l7_policy_id", "result_count", "result_json"},
		"nobus_lbaas_task":                     {"id", "task_id", "result_json"},
		"nobus_lbaas_deployment_status":        {"id", "load_balancer_id", "result_json"},
		"nobus_lbaas_load_balancer_status":     {"id", "load_balancer_id", "result_json"},
		"nobus_lbaas_load_balancer_stats":      {"id", "load_balancer_id", "result_json"},
		"nobus_lbaas_listener_stats":           {"id", "listener_id", "result_json"},
	}
	for name, expected := range additional {
		dsSchema, ok := response.DataSourceSchemas[name]
		if !ok {
			t.Errorf("data source %s is not registered", name)
			continue
		}
		attrs := attributeNames(dsSchema)
		for _, attribute := range expected {
			if _, ok := attrs[attribute]; !ok {
				t.Errorf("%s is missing %s", name, attribute)
			}
		}
	}
}

// attributeNames returns the attribute names declared by a schema block.
func attributeNames(schema *tfprotov6.Schema) map[string]struct{} {
	names := make(map[string]struct{})
	if schema == nil || schema.Block == nil {
		return names
	}
	for _, attribute := range schema.Block.Attributes {
		names[attribute.Name] = struct{}{}
	}
	return names
}

// hasBlockType reports whether a schema declares a nested block with the given
// type name.
func hasBlockType(schema *tfprotov6.Schema, name string) bool {
	if schema == nil || schema.Block == nil {
		return false
	}
	for _, blockType := range schema.Block.BlockTypes {
		if blockType.TypeName == name {
			return true
		}
	}
	return false
}
