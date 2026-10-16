// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// These acceptance tests talk to the real Nobus Cloud API and are only run when
// TF_ACC is set. They are the intended home for end-to-end coverage once a
// project is available; the unit tests in this package cover request mapping and
// state refresh without requiring infrastructure.
//
// Required environment:
//
//	NOBUS_API_KEY   API key used as the bearer token
//	NOBUS_PROJECT_ID project that owns the resources
//
// Optional environment:
//
//	NOBUS_ENDPOINT         defaults to https://cloud-api.nobus.io
//	NOBUS_AVAILABILITY_ZONE defaults to nobus-wa-az1
//	NOBUS_TEST_IMAGE       image used by the instance tests
//	NOBUS_TEST_FLAVOR      flavor used by the instance tests
//	NOBUS_TEST_KEY_NAME    keypair used by the instance tests

func testAccProviderConfig() string {
	return fmt.Sprintf(`
provider "nobus" {
  api_key     = %q
  endpoint    = %q
  project_id  = %q
}
`, os.Getenv("NOBUS_API_KEY"), endpointOrDefault(), os.Getenv("NOBUS_PROJECT_ID"))
}

func endpointOrDefault() string {
	if endpoint := os.Getenv("NOBUS_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return "https://cloud-api.nobus.io"
}

func testAccPreCheckInstance(t *testing.T) {
	t.Helper()

	testAccPreCheck(t)

	if os.Getenv("NOBUS_TEST_IMAGE") == "" {
		t.Skip("NOBUS_TEST_IMAGE must be set for the instance acceptance tests")
	}
	if os.Getenv("NOBUS_TEST_FLAVOR") == "" {
		t.Skip("NOBUS_TEST_FLAVOR must be set for the instance acceptance tests")
	}
	if os.Getenv("NOBUS_TEST_KEY_NAME") == "" {
		t.Skip("NOBUS_TEST_KEY_NAME must be set for the instance acceptance tests")
	}
}

// TestAccNetwork_basic exercises the full network lifecycle plus import.
func TestAccNetwork_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "nobus_network" "test" {
  name        = "tf-acc-network"
  subnet_cidr = "10.99.0.0/24"
  router_name = "tf-acc-router"

  description = "acceptance test"
  tags        = ["created-by=terraform-acc"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nobus_network.test", "id"),
					resource.TestCheckResourceAttrSet("nobus_network.test", "subnet_id"),
					resource.TestCheckResourceAttrSet("nobus_network.test", "router_id"),
					resource.TestCheckResourceAttr("nobus_network.test", "name", "tf-acc-network"),
					resource.TestCheckResourceAttr("nobus_network.test", "description", "acceptance test"),
				),
			},
			{
				ResourceName:      "nobus_network.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The create-only attributes are not returned by the API, so they
				// cannot be verified after an import.
				ImportStateVerifyIgnore: []string{"subnet_cidr", "router_name"},
			},
		},
	})
}

// TestAccVolume_basic exercises the volume lifecycle, growth and import.
func TestAccVolume_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "nobus_volume" "test" {
  name = "tf-acc-volume"
  size = 10

  metadata = {
    created_by = "terraform-acc"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nobus_volume.test", "id"),
					resource.TestCheckResourceAttr("nobus_volume.test", "size", "10"),
					resource.TestCheckResourceAttr("nobus_volume.test", "metadata.created_by", "terraform-acc"),
				),
			},
			{
				// Growing a volume is applied in place through the Nobus extend
				// operation.
				Config: testAccProviderConfig() + `
resource "nobus_volume" "test" {
  name = "tf-acc-volume"
  size = 20

  metadata = {
    created_by = "terraform-acc"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("nobus_volume.test", "size", "20"),
				),
			},
			{
				ResourceName:      "nobus_volume.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccInstance_basic exercises the instance lifecycle with a network
// dependency, plus import.
func TestAccInstance_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckInstance(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
resource "nobus_network" "test" {
  name        = "tf-acc-instance-net"
  subnet_cidr = "10.98.0.0/24"
  router_name = "tf-acc-instance-router"
}

resource "nobus_instance" "test" {
  name      = "tf-acc-instance"
  image     = %q
  flavor    = %q
  key_name  = %q
  network_id = nobus_network.test.id

  volume_size = 20
}
`, os.Getenv("NOBUS_TEST_IMAGE"), os.Getenv("NOBUS_TEST_FLAVOR"), os.Getenv("NOBUS_TEST_KEY_NAME")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("nobus_instance.test", "id"),
					resource.TestCheckResourceAttr("nobus_instance.test", "name", "tf-acc-instance"),
					resource.TestCheckResourceAttr("nobus_instance.test", "volume_size", "20"),
					resource.TestCheckResourceAttrSet("nobus_instance.test", "status"),
				),
			},
			{
				ResourceName:      "nobus_instance.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Write-only and create-only attributes cannot be read back.
				ImportStateVerifyIgnore: []string{"image", "flavor", "key_name", "network_id", "volume_size", "security_groups"},
			},
		},
	})
}
