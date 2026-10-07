// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

// TestApplyNetworkToModel verifies that every refreshable network property is
// written into state from the API response.
func TestApplyNetworkToModel(t *testing.T) {
	t.Parallel()

	name := "production-network"
	description := "primary"
	status := "ACTIVE"
	projectID := "p-1"
	mtu := int64(1500)

	network := &client.Network{
		ID:                  "n-1",
		Name:                &name,
		Description:         &description,
		ProjectID:           &projectID,
		Status:              &status,
		AdminStateUp:        boolPtr(true),
		Shared:              boolPtr(false),
		MTU:                 &mtu,
		RouterExternal:      boolPtr(false),
		PortSecurityEnabled: boolPtr(true),
		Tags:                []string{"env=prod"},
		Subnets:             []string{"s-1"},
	}

	var diags diag.Diagnostics
	model := networkResourceModel{
		// Create-only attributes are supplied by the configuration and must not
		// be overwritten by a read.
		SubnetCIDR: types.StringValue("10.0.0.0/24"),
		RouterName: types.StringValue("production-router"),
	}

	applyNetworkToModel(context.Background(), network, &model, &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags.Errors())
	}
	if model.ID.ValueString() != "n-1" {
		t.Errorf("id = %q, want n-1", model.ID.ValueString())
	}
	if model.Name.ValueString() != name {
		t.Errorf("name = %q, want %q", model.Name.ValueString(), name)
	}
	if model.Description.ValueString() != description {
		t.Errorf("description = %q, want %q", model.Description.ValueString(), description)
	}
	if model.Status.ValueString() != status {
		t.Errorf("status = %q, want %q", model.Status.ValueString(), status)
	}
	if model.ProjectID.ValueString() != projectID {
		t.Errorf("project_id = %q, want %q", model.ProjectID.ValueString(), projectID)
	}
	if model.MTU.ValueInt64() != 1500 {
		t.Errorf("mtu = %d, want 1500", model.MTU.ValueInt64())
	}
	if !model.AdminStateUp.ValueBool() {
		t.Error("admin_state_up should be true")
	}
	if model.Shared.ValueBool() {
		t.Error("shared should be false")
	}
	if model.Tags.IsNull() {
		t.Error("tags should not be null")
	}
	if model.SubnetIDs.IsNull() {
		t.Error("subnet_ids should not be null")
	}

	// The immutable configuration values must survive a refresh.
	if model.SubnetCIDR.ValueString() != "10.0.0.0/24" {
		t.Errorf("subnet_cidr was overwritten: %q", model.SubnetCIDR.ValueString())
	}
	if model.RouterName.ValueString() != "production-router" {
		t.Errorf("router_name was overwritten: %q", model.RouterName.ValueString())
	}
}

// TestApplyNetworkToModelHandlesNulls verifies that absent API values become
// null Terraform values rather than empty strings, avoiding perpetual diffs.
func TestApplyNetworkToModelHandlesNulls(t *testing.T) {
	t.Parallel()

	var diags diag.Diagnostics
	model := networkResourceModel{}
	applyNetworkToModel(context.Background(), &client.Network{ID: "n-1"}, &model, &diags)

	if !model.Name.IsNull() {
		t.Errorf("name should be null, got %q", model.Name.ValueString())
	}
	if !model.Description.IsNull() {
		t.Error("description should be null")
	}
	if !model.Status.IsNull() {
		t.Error("status should be null")
	}
	if !model.Tags.IsNull() {
		t.Error("tags should be null")
	}
	if !model.SubnetIDs.IsNull() {
		t.Error("subnet_ids should be null")
	}
	if !model.RouterExternal.IsNull() {
		t.Error("router_external should be null")
	}
}

// TestApplyVolumeToModel verifies the volume refresh mapping.
func TestApplyVolumeToModel(t *testing.T) {
	t.Parallel()

	name := "production-data"
	status := client.VolumeStateAvailable
	volumeType := "ssd"
	projectID := "p-1"

	volume := &client.Volume{
		ID:          "v-1",
		Name:        &name,
		Size:        100,
		Status:      &status,
		VolumeType:  &volumeType,
		ProjectID:   &projectID,
		Bootable:    boolPtr(false),
		Encrypted:   boolPtr(true),
		Metadata:    map[string]string{"env": "prod"},
		Attachments: []client.VolumeAttachment{{AttachmentID: stringPtr("a-1"), ServerID: stringPtr("i-1")}},
	}

	var diags diag.Diagnostics
	model := volumeResourceModel{}
	applyVolumeToModel(context.Background(), volume, &model, &diags)

	if model.ID.ValueString() != "v-1" {
		t.Errorf("id = %q, want v-1", model.ID.ValueString())
	}
	if model.Size.ValueInt64() != 100 {
		t.Errorf("size = %d, want 100", model.Size.ValueInt64())
	}
	if model.Status.ValueString() != "available" {
		t.Errorf("status = %q, want available", model.Status.ValueString())
	}
	if model.VolumeType.ValueString() != volumeType {
		t.Errorf("volume_type = %q, want %q", model.VolumeType.ValueString(), volumeType)
	}
	if !model.Encrypted.ValueBool() {
		t.Error("encrypted should be true")
	}
	if model.AttachmentIDs.IsNull() {
		t.Error("attachment_ids should not be null")
	}
	if model.ServerIDs.IsNull() {
		t.Error("server_ids should not be null")
	}
}

// TestApplyInstanceToModel verifies the instance refresh mapping, including the
// JSON encoding of the untyped addresses object.
func TestApplyInstanceToModel(t *testing.T) {
	t.Parallel()

	status := "ACTIVE"
	instance := &client.Instance{
		ID:      "i-1",
		Name:    "web-01",
		Status:  &status,
		KeyName: stringPtr("ops"),
		Flavor:  map[string]any{"id": "f-1", "name": "small"},
		Image:   map[string]any{"id": "img-1", "name": "ubuntu-22-04"},
		Addresses: map[string]any{
			"private": []any{map[string]any{"addr": "10.0.0.5"}},
		},
		Volumes: []client.InstanceVolume{{ID: "v-1"}},
	}

	var diags diag.Diagnostics
	model := instanceResourceModel{
		// Write-only attributes are never returned by the API.
		AdminPass: types.StringValue("secret"),
		UserData:  types.StringValue("#!/bin/sh"),
	}
	applyInstanceToModel(context.Background(), instance, &model, &diags)

	if model.ID.ValueString() != "i-1" {
		t.Errorf("id = %q, want i-1", model.ID.ValueString())
	}
	if model.Status.ValueString() != "ACTIVE" {
		t.Errorf("status = %q, want ACTIVE", model.Status.ValueString())
	}
	if model.FlavorID.ValueString() != "f-1" {
		t.Errorf("flavor_id = %q, want f-1", model.FlavorID.ValueString())
	}
	if model.FlavorName.ValueString() != "small" {
		t.Errorf("flavor_name = %q, want small", model.FlavorName.ValueString())
	}
	if model.ImageID.ValueString() != "img-1" {
		t.Errorf("image_id = %q, want img-1", model.ImageID.ValueString())
	}
	if model.VolumeIDs.IsNull() {
		t.Error("volume_ids should not be null")
	}
	if !strings.Contains(model.Addresses.ValueString(), "10.0.0.5") {
		t.Errorf("addresses should contain the reported address, got %q", model.Addresses.ValueString())
	}

	// Write-only values must be preserved across a refresh.
	if model.AdminPass.ValueString() != "secret" {
		t.Error("admin_pass must not be cleared by a read")
	}
	if model.UserData.ValueString() != "#!/bin/sh" {
		t.Error("user_data must not be cleared by a read")
	}
}

// TestNetworkUpdateRequired verifies which attributes trigger an update call.
func TestNetworkUpdateRequired(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		model networkResourceModel
		want  bool
	}{
		"no updatable attribute": {
			model: networkResourceModel{},
			want:  false,
		},
		"description set": {
			model: networkResourceModel{Description: types.StringValue("primary")},
			want:  true,
		},
		"mtu set": {
			model: networkResourceModel{MTU: types.Int64Value(9000)},
			want:  true,
		},
		"only immutable attributes": {
			model: networkResourceModel{
				SubnetCIDR: types.StringValue("10.0.0.0/24"),
				RouterName: types.StringValue("router"),
			},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := networkUpdateRequired(tc.model); got != tc.want {
				t.Errorf("networkUpdateRequired = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestVolumeAttributesChanged verifies the update trigger for volumes.
func TestVolumeAttributesChanged(t *testing.T) {
	t.Parallel()

	nullMetadata := types.MapNull(types.StringType)

	state := volumeResourceModel{
		Name:        types.StringValue("data"),
		Description: types.StringNull(),
		Metadata:    nullMetadata,
	}
	unchanged := volumeResourceModel{
		Name:        types.StringValue("data"),
		Description: types.StringNull(),
		Metadata:    nullMetadata,
	}
	renamed := volumeResourceModel{
		Name:        types.StringValue("data-2"),
		Description: types.StringNull(),
		Metadata:    nullMetadata,
	}

	if volumeAttributesChanged(state, unchanged) {
		t.Error("identical models must not trigger an update")
	}
	if !volumeAttributesChanged(state, renamed) {
		t.Error("a renamed volume must trigger an update")
	}
}

// TestOptionalConverters verifies the null handling of the request converters.
func TestOptionalConverters(t *testing.T) {
	t.Parallel()

	if got := optionalString(types.StringNull()); got != nil {
		t.Errorf("optionalString(null) = %v, want nil", *got)
	}
	if got := optionalString(types.StringUnknown()); got != nil {
		t.Error("optionalString(unknown) should be nil")
	}
	if got := optionalString(types.StringValue("x")); got == nil || *got != "x" {
		t.Error("optionalString(value) should return a pointer to the value")
	}
	if got := optionalBool(types.BoolNull()); got != nil {
		t.Error("optionalBool(null) should be nil")
	}
	if got := optionalInt64(types.Int64Null()); got != nil {
		t.Error("optionalInt64(null) should be nil")
	}
	if !stringOrNull(nil).IsNull() {
		t.Error("stringOrNull(nil) should be null")
	}
	if !boolOrNull(nil).IsNull() {
		t.Error("boolOrNull(nil) should be null")
	}
	if !int64OrNull(nil).IsNull() {
		t.Error("int64OrNull(nil) should be null")
	}
	if !stringValueOrNull("").IsNull() {
		t.Error("stringValueOrNull(\"\") should be null")
	}
}

// TestResolveScope verifies the provider defaults are applied.
func TestResolveScope(t *testing.T) {
	t.Parallel()

	config := &providerConfig{
		ProjectID:        "provider-project",
		AvailabilityZone: "provider-az",
	}

	t.Run("falls back to the provider defaults", func(t *testing.T) {
		var diags diag.Diagnostics
		resolved, ok := config.resolveScope(types.StringNull(), types.StringNull(), &diags)
		if !ok {
			t.Fatal("expected the scope to resolve")
		}
		if resolved.ProjectID != "provider-project" {
			t.Errorf("project = %q, want provider-project", resolved.ProjectID)
		}
		if resolved.AvailabilityZone != "provider-az" {
			t.Errorf("zone = %q, want provider-az", resolved.AvailabilityZone)
		}
	})

	t.Run("resource values win over the provider defaults", func(t *testing.T) {
		var diags diag.Diagnostics
		resolved, ok := config.resolveScope(types.StringValue("p-9"), types.StringValue("az-9"), &diags)
		if !ok {
			t.Fatal("expected the scope to resolve")
		}
		if resolved.ProjectID != "p-9" || resolved.AvailabilityZone != "az-9" {
			t.Errorf("unexpected scope: %+v", resolved)
		}
	})

	t.Run("reports a missing project", func(t *testing.T) {
		empty := &providerConfig{}
		var diags diag.Diagnostics
		if _, ok := empty.resolveScope(types.StringNull(), types.StringNull(), &diags); ok {
			t.Error("expected resolution to fail without a project ID")
		}
		if !diags.HasError() {
			t.Error("expected a diagnostic explaining the missing project ID")
		}
	})
}

// TestDiagnosticDetailPreservesAPIMessage verifies that error diagnostics carry
// the information Nobus returned.
func TestDiagnosticDetailPreservesAPIMessage(t *testing.T) {
	t.Parallel()

	err := &client.APIError{
		StatusCode: 400,
		Message:    "requested image is unavailable in the selected availability zone",
	}
	detail := diagnosticDetail(err)
	if detail == "" || !strings.Contains(detail, "requested image is unavailable") {
		t.Errorf("diagnostic should preserve the API message, got: %s", detail)
	}

	statusErr := &client.StatusError{Message: "cidr overlaps an existing subnet"}
	if detail := diagnosticDetail(statusErr); !strings.Contains(detail, "cidr overlaps") {
		t.Errorf("diagnostic should preserve the API message, got: %s", detail)
	}
}

func boolPtr(v bool) *bool { return &v }

func stringPtr(v string) *string { return &v }
