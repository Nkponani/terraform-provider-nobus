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
	if provider.Block.BlockTypes != nil && len(provider.Block.BlockTypes) > 0 {
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
			"metadata", "status", "bootable", "encrypted", "created_at", "updated_at",
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
		"nobus_keypair":             {"id", "project_id", "availability_zone", "name", "public_key", "fingerprint", "user_id", "type"},
		"nobus_subnet":              {"id", "project_id", "availability_zone", "network_id", "cidr", "ip_version", "name", "description", "gateway_ip", "enable_dhcp", "dns_nameservers"},
		"nobus_router":              {"id", "project_id", "availability_zone", "name", "description", "admin_state_up", "status"},
		"nobus_router_interface":    {"id", "project_id", "availability_zone", "router_id", "subnet_id", "port_id"},
		"nobus_port":                {"id", "project_id", "availability_zone", "network_id", "name", "description", "admin_state_up", "device_id", "device_owner", "security_groups", "port_security_enabled", "dns_name", "status", "mac_address"},
		"nobus_floating_ip":         {"id", "project_id", "availability_zone", "floating_network_id", "port_id", "fixed_ip_address", "description", "floating_ip_address", "status"},
		"nobus_security_group":      {"id", "project_id", "availability_zone", "name", "description"},
		"nobus_security_group_rule": {"id", "project_id", "availability_zone", "security_group_id", "direction", "ethertype", "protocol", "port_range_min", "port_range_max", "remote_ip_prefix", "remote_group_id", "description"},
		"nobus_instance_action":     {"id", "project_id", "availability_zone", "instance_id", "action", "reboot_type", "flavor_id", "port_id", "network_id", "fixed_ip", "security_group_name", "image_name", "new_password", "tags", "tag", "result"},
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
			if resourceName == "nobus_volume" {
				if _, ok := attributes["multiattach"]; ok {
					t.Error("nobus_volume must not expose the unsupported multiattach attribute")
				}
			}

			if hasBlockType(resourceSchema, "timeouts") {
				t.Errorf("resource %s must not expose a timeouts block", resourceName)
			}
		})
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
