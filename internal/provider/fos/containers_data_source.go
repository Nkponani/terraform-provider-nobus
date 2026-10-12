// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fos

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

var containerListItemType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"name":          types.StringType,
	"object_count":  types.Int64Type,
	"bytes":         types.Int64Type,
	"last_modified": types.StringType,
}}

type containersDataSource struct{ config *shared.Config }

type containerListItemModel struct {
	Name         types.String `tfsdk:"name"`
	ObjectCount  types.Int64  `tfsdk:"object_count"`
	Bytes        types.Int64  `tfsdk:"bytes"`
	LastModified types.String `tfsdk:"last_modified"`
}

type containersDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Limit            types.Int64  `tfsdk:"limit"`
	Marker           types.String `tfsdk:"marker"`
	EndMarker        types.String `tfsdk:"end_marker"`
	Prefix           types.String `tfsdk:"prefix"`
	Delimiter        types.String `tfsdk:"delimiter"`
	FullListing      types.Bool   `tfsdk:"full_listing"`
	ResultCount      types.Int64  `tfsdk:"result_count"`
	Containers       types.List   `tfsdk:"containers"`
}

func NewContainersDataSource() datasource.DataSource { return &containersDataSource{} }

func (d *containersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fos_containers"
}

func (d *containersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.config = config
}

func (d *containersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Lists Nobus FOS containers.", Attributes: map[string]schema.Attribute{
		"id":                schema.StringAttribute{Computed: true},
		"project_id":        schema.StringAttribute{Optional: true, Computed: true},
		"availability_zone": schema.StringAttribute{Optional: true, Computed: true},
		"limit":             schema.Int64Attribute{Optional: true},
		"marker":            schema.StringAttribute{Optional: true},
		"end_marker":        schema.StringAttribute{Optional: true},
		"prefix":            schema.StringAttribute{Optional: true},
		"delimiter":         schema.StringAttribute{Optional: true},
		"full_listing":      schema.BoolAttribute{Optional: true},
		"result_count":      schema.Int64Attribute{Computed: true},
		"containers": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{Computed: true}, "object_count": schema.Int64Attribute{Computed: true}, "bytes": schema.Int64Attribute{Computed: true}, "last_modified": schema.StringAttribute{Computed: true},
		}}},
	}}
}

func (d *containersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config containersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(config.ProjectID, config.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	options := client.FOSListOptions{Limit: config.Limit.ValueInt64(), Marker: config.Marker.ValueString(), EndMarker: config.EndMarker.ValueString(), Prefix: config.Prefix.ValueString(), Delimiter: config.Delimiter.ValueString(), FullListing: config.FullListing.ValueBool()}
	containers, err := api.ListFOSContainers(ctx, scope.ProjectID, scope.AvailabilityZone, options)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Nobus FOS containers", shared.DiagnosticDetail(err))
		return
	}
	items := make([]containerListItemModel, 0, len(containers))
	for _, container := range containers {
		items = append(items, containerListItemModel{Name: types.StringValue(container.Name), ObjectCount: types.Int64Value(container.Count), Bytes: types.Int64Value(container.Bytes), LastModified: shared.StringOrNull(container.LastModified)})
	}
	value, diags := types.ListValueFrom(ctx, containerListItemType, items)
	resp.Diagnostics.Append(diags...)
	config.ID = types.StringValue(scope.ProjectID + "/" + scope.AvailabilityZone)
	config.ProjectID = types.StringValue(scope.ProjectID)
	config.AvailabilityZone = types.StringValue(scope.AvailabilityZone)
	config.ResultCount = types.Int64Value(int64(len(containers)))
	config.Containers = value
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

var _ datasource.DataSourceWithConfigure = &containersDataSource{}
