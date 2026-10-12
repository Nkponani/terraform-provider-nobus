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

var objectListItemType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"name":          types.StringType,
	"bytes":         types.Int64Type,
	"content_type":  types.StringType,
	"last_modified": types.StringType,
	"etag":          types.StringType,
	"subdirectory":  types.StringType,
}}

type objectsDataSource struct{ config *shared.Config }

type objectListItemModel struct {
	Name         types.String `tfsdk:"name"`
	Bytes        types.Int64  `tfsdk:"bytes"`
	ContentType  types.String `tfsdk:"content_type"`
	LastModified types.String `tfsdk:"last_modified"`
	ETag         types.String `tfsdk:"etag"`
	Subdirectory types.String `tfsdk:"subdirectory"`
}

type objectsDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	ContainerName    types.String `tfsdk:"container_name"`
	Limit            types.Int64  `tfsdk:"limit"`
	Marker           types.String `tfsdk:"marker"`
	EndMarker        types.String `tfsdk:"end_marker"`
	Prefix           types.String `tfsdk:"prefix"`
	Delimiter        types.String `tfsdk:"delimiter"`
	Path             types.String `tfsdk:"path"`
	FullListing      types.Bool   `tfsdk:"full_listing"`
	ResultCount      types.Int64  `tfsdk:"result_count"`
	Objects          types.List   `tfsdk:"objects"`
}

func NewObjectsDataSource() datasource.DataSource { return &objectsDataSource{} }

func (d *objectsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fos_objects"
}

func (d *objectsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.config = config
}

func (d *objectsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Lists objects in a Nobus FOS container.", Attributes: map[string]schema.Attribute{
		"id":                schema.StringAttribute{Computed: true},
		"project_id":        schema.StringAttribute{Optional: true, Computed: true},
		"availability_zone": schema.StringAttribute{Optional: true, Computed: true},
		"container_name":    schema.StringAttribute{Required: true},
		"limit":             schema.Int64Attribute{Optional: true},
		"marker":            schema.StringAttribute{Optional: true},
		"end_marker":        schema.StringAttribute{Optional: true},
		"prefix":            schema.StringAttribute{Optional: true},
		"delimiter":         schema.StringAttribute{Optional: true},
		"path":              schema.StringAttribute{Optional: true},
		"full_listing":      schema.BoolAttribute{Optional: true},
		"result_count":      schema.Int64Attribute{Computed: true},
		"objects": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{Computed: true}, "bytes": schema.Int64Attribute{Computed: true}, "content_type": schema.StringAttribute{Computed: true}, "last_modified": schema.StringAttribute{Computed: true}, "etag": schema.StringAttribute{Computed: true}, "subdirectory": schema.StringAttribute{Computed: true},
		}}},
	}}
}

func (d *objectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config objectsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(config.ProjectID, config.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	options := client.FOSListOptions{
		Limit: config.Limit.ValueInt64(), Marker: config.Marker.ValueString(), EndMarker: config.EndMarker.ValueString(),
		Prefix: config.Prefix.ValueString(), Delimiter: config.Delimiter.ValueString(), Path: config.Path.ValueString(),
		FullListing: config.FullListing.ValueBool(),
	}
	objects, err := api.ListFOSObjects(ctx, scope.ProjectID, scope.AvailabilityZone, config.ContainerName.ValueString(), options)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Nobus FOS objects", shared.DiagnosticDetail(err))
		return
	}
	items := make([]objectListItemModel, 0, len(objects))
	for _, object := range objects {
		items = append(items, objectListItemModel{
			Name: types.StringValue(object.Name), Bytes: types.Int64Value(object.Bytes), ContentType: shared.StringOrNull(object.ContentType),
			LastModified: shared.StringOrNull(object.LastModified), ETag: shared.StringOrNull(object.ETag), Subdirectory: shared.StringOrNull(object.Subdirectory),
		})
	}
	value, diags := types.ListValueFrom(ctx, objectListItemType, items)
	resp.Diagnostics.Append(diags...)
	config.ID = types.StringValue(scope.ProjectID + "/" + scope.AvailabilityZone + "/" + config.ContainerName.ValueString())
	config.ProjectID = types.StringValue(scope.ProjectID)
	config.AvailabilityZone = types.StringValue(scope.AvailabilityZone)
	config.ResultCount = types.Int64Value(int64(len(objects)))
	config.Objects = value
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

var _ datasource.DataSourceWithConfigure = &objectsDataSource{}
