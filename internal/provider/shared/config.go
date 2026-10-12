// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

// Config is the resolved configuration shared with every resource. It
// carries the API client together with the defaults applied to resources that
// do not set project_id or availability_zone themselves.
type Config struct {
	Client           *client.Client
	ProjectID        string
	AvailabilityZone string
}

// FromProviderData extracts the shared configuration, reporting a
// clear diagnostic when the provider data is not what the provider produced.
func FromProviderData(providerData any) (*Config, diag.Diagnostics) {
	var diags diag.Diagnostics

	if providerData == nil {
		// Terraform calls Configure for resources before the provider itself is
		// configured during validation; this is not an error.
		return nil, diags
	}

	config, ok := providerData.(*Config)
	if !ok {
		diags.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected *Config, got %T. This is a bug in the provider.", providerData),
		)
		return nil, diags
	}

	return config, diags
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
