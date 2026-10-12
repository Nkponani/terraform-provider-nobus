// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

// providerConfig is the resolved configuration shared with every resource. It
// carries the API client together with the defaults applied to resources that
// do not set project_id or availability_zone themselves.
type providerConfig = shared.Config

// Configure builds the Nobus API client from the provider configuration and the
// environment.
func (p *NobusProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config NobusProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Unknown values during the initial plan are expected; treat them as unset
	// so that a partially known configuration can still be used.
	if config.APIKey.IsUnknown() {
		config.APIKey = types.StringNull()
	}
	if config.Endpoint.IsUnknown() {
		config.Endpoint = types.StringNull()
	}
	if config.ProjectID.IsUnknown() {
		config.ProjectID = types.StringNull()
	}
	if config.AvailabilityZone.IsUnknown() {
		config.AvailabilityZone = types.StringNull()
	}

	apiKey := firstNonEmpty(config.APIKey.ValueString(), os.Getenv("NOBUS_API_KEY"))
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Nobus API key",
			"The Nobus provider requires an API key. Set the `api_key` attribute in the provider block "+
				"or set the NOBUS_API_KEY environment variable.",
		)
	}

	endpoint := firstNonEmpty(config.Endpoint.ValueString(), os.Getenv("NOBUS_ENDPOINT"), client.DefaultBaseURL)

	if resp.Diagnostics.HasError() {
		return
	}

	apiClient, err := client.New(client.Config{
		APIKey:  apiKey,
		BaseURL: endpoint,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Nobus API client",
			err.Error(),
		)
		return
	}

	data := &providerConfig{
		Client:           apiClient,
		ProjectID:        firstNonEmpty(config.ProjectID.ValueString(), os.Getenv("NOBUS_PROJECT_ID")),
		AvailabilityZone: firstNonEmpty(config.AvailabilityZone.ValueString(), os.Getenv("NOBUS_AVAILABILITY_ZONE"), client.DefaultAvailabilityZone),
	}

	resp.DataSourceData = data
	resp.ResourceData = data
}

var providerConfigFromResourceData = shared.FromProviderData

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
