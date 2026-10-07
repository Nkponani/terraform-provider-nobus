// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0
package provider

import (
	"context"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure NobusProvider satisfies the provider interfaces.
var _ provider.Provider = &NobusProvider{}

// NobusProvider defines the Nobus Cloud provider implementation.
type NobusProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// NobusProviderModel describes the provider data model.
type NobusProviderModel struct {
	APIKey           types.String `tfsdk:"api_key"`
	Endpoint         types.String `tfsdk:"endpoint"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
}

// Metadata returns the provider type name and version.
func (p *NobusProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nobus"
	resp.Version = p.version
}

// Schema defines the provider level configuration.
func (p *NobusProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage infrastructure on [Nobus Cloud](https://cloud.nobus.io) through the public Nobus API.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Nobus API key, sent as a bearer token. Can also be set with the `NOBUS_API_KEY` " +
					"environment variable. Keep the key in a sensitive variable rather than in version control.",
				Optional:  true,
				Sensitive: true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Base URL of the Nobus API. Defaults to `" + client.DefaultBaseURL + "`. " +
					"Can also be set with the `NOBUS_ENDPOINT` environment variable.",
				Optional: true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Default Nobus project ID applied to resources that do not set their own " +
					"`project_id`. Can also be set with the `NOBUS_PROJECT_ID` environment variable.",
				Optional: true,
			},
			"availability_zone": schema.StringAttribute{
				MarkdownDescription: "Default availability zone applied to resources that do not set their own " +
					"`availability_zone`. Defaults to `" + client.DefaultAvailabilityZone + "`, the value the Nobus API " +
					"itself defaults to. Can also be set with the `NOBUS_AVAILABILITY_ZONE` environment variable.",
				Optional: true,
			},
		},
	}
}

// Resources returns the managed resources offered by this provider.
func (p *NobusProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNetworkResource,
		NewVolumeResource,
		NewInstanceResource,
		NewVolumeAttachmentResource,
		NewKeypairResource,
		NewSubnetResource,
		NewRouterResource,
		NewRouterInterfaceResource,
		NewPortResource,
		NewFloatingIPResource,
		NewSecurityGroupResource,
		NewSecurityGroupRuleResource,
		NewInstanceActionResource,
	}
}

// DataSources returns the data sources offered by this provider.
func (p *NobusProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// New returns a function that returns a NobusProvider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NobusProvider{
			version: version,
		}
	}
}
