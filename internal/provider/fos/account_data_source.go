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

type accountDataSource struct{ config *shared.Config }

type accountDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectID      types.String `tfsdk:"project_id"`
	ContainerCount types.Int64  `tfsdk:"container_count"`
	ObjectCount    types.Int64  `tfsdk:"object_count"`
	BytesUsed      types.Int64  `tfsdk:"bytes_used"`
}

func NewAccountDataSource() datasource.DataSource { return &accountDataSource{} }

func (d *accountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fos_account"
}

func (d *accountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.config = config
}

func (d *accountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Nobus FOS account usage totals.", Attributes: map[string]schema.Attribute{
		"id":              schema.StringAttribute{Computed: true},
		"project_id":      schema.StringAttribute{Optional: true, Computed: true},
		"container_count": schema.Int64Attribute{Computed: true},
		"object_count":    schema.Int64Attribute{Computed: true},
		"bytes_used":      schema.Int64Attribute{Computed: true},
	}}
}

func (d *accountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config accountDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(config.ProjectID, types.StringNull(), &resp.Diagnostics)
	if !ok {
		return
	}
	account, err := api.GetFOSAccount(ctx, scope.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Nobus FOS account", shared.DiagnosticDetail(err))
		return
	}
	config.ID = types.StringValue(scope.ProjectID)
	config.ProjectID = types.StringValue(scope.ProjectID)
	config.ContainerCount = types.Int64Value(account.ContainerCount)
	config.ObjectCount = types.Int64Value(account.ObjectCount)
	config.BytesUsed = types.Int64Value(account.BytesUsed)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

var _ datasource.DataSourceWithConfigure = &accountDataSource{}
