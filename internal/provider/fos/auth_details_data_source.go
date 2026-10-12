// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fos

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type authDetailsDataSource struct{ config *shared.Config }

type authDetailsDataSourceModel struct {
	ID                      types.String `tfsdk:"id"`
	ProjectID               types.String `tfsdk:"project_id"`
	UserName                types.String `tfsdk:"user_name"`
	ProjectName             types.String `tfsdk:"project_name"`
	DomainID                types.String `tfsdk:"domain_id"`
	Region                  types.String `tfsdk:"region"`
	IdentityServiceEndpoint types.String `tfsdk:"identity_service_endpoint"`
	IdentityServiceVersion  types.Int64  `tfsdk:"identity_service_version"`
	ManagementEndpoint      types.String `tfsdk:"management_endpoint"`
}

func NewAuthDetailsDataSource() datasource.DataSource { return &authDetailsDataSource{} }

func (d *authDetailsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fos_auth_details"
}

func (d *authDetailsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.config = config
}

func (d *authDetailsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Nobus FOS Swift-compatible endpoint and identity details.", Attributes: map[string]schema.Attribute{
		"id":                        schema.StringAttribute{Computed: true},
		"project_id":                schema.StringAttribute{Optional: true, Computed: true},
		"user_name":                 schema.StringAttribute{Computed: true},
		"project_name":              schema.StringAttribute{Computed: true},
		"domain_id":                 schema.StringAttribute{Computed: true},
		"region":                    schema.StringAttribute{Computed: true},
		"identity_service_endpoint": schema.StringAttribute{Computed: true},
		"identity_service_version":  schema.Int64Attribute{Computed: true},
		"management_endpoint":       schema.StringAttribute{Computed: true},
	}}
}

func (d *authDetailsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config authDetailsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(config.ProjectID, types.StringNull(), &resp.Diagnostics)
	if !ok {
		return
	}
	details, err := api.GetFOSAuthDetails(ctx, scope.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Nobus FOS authentication details", shared.DiagnosticDetail(err))
		return
	}
	config.ID = types.StringValue(scope.ProjectID)
	config.ProjectID = types.StringValue(details.ProjectID)
	config.UserName = types.StringValue(details.UserName)
	config.ProjectName = types.StringValue(details.ProjectName)
	config.DomainID = types.StringValue(details.DomainID)
	config.Region = types.StringValue(details.Region)
	config.IdentityServiceEndpoint = types.StringValue(details.IdentityServiceEndpoint)
	config.IdentityServiceVersion = types.Int64Value(details.IdentityServiceVersion)
	config.ManagementEndpoint = types.StringValue(details.ManagementEndpoint)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

var _ datasource.DataSourceWithConfigure = &authDetailsDataSource{}
