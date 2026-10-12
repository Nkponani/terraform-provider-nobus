// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fos

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type objectContentDataSource struct{ config *shared.Config }

type objectContentDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	ContainerName    types.String `tfsdk:"container_name"`
	ObjectName       types.String `tfsdk:"object_name"`
	ContentBase64    types.String `tfsdk:"content_base64"`
	Bytes            types.Int64  `tfsdk:"bytes"`
	SHA256           types.String `tfsdk:"sha256"`
}

func NewObjectContentDataSource() datasource.DataSource { return &objectContentDataSource{} }

func (d *objectContentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fos_object_content"
}

func (d *objectContentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.config = config
}

func (d *objectContentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Downloads a Nobus FOS object. The content is stored in Terraform state as sensitive base64.", Attributes: map[string]schema.Attribute{
		"id":                schema.StringAttribute{Computed: true},
		"project_id":        schema.StringAttribute{Optional: true, Computed: true},
		"availability_zone": schema.StringAttribute{Optional: true, Computed: true},
		"container_name":    schema.StringAttribute{Required: true},
		"object_name":       schema.StringAttribute{Required: true},
		"content_base64":    schema.StringAttribute{Computed: true, Sensitive: true},
		"bytes":             schema.Int64Attribute{Computed: true},
		"sha256":            schema.StringAttribute{Computed: true},
	}}
}

func (d *objectContentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config objectContentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(config.ProjectID, config.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	content, err := api.DownloadFOSObject(ctx, scope.ProjectID, scope.AvailabilityZone, config.ContainerName.ValueString(), config.ObjectName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to download Nobus FOS object", shared.DiagnosticDetail(err))
		return
	}
	digest := sha256.Sum256(content)
	config.ID = types.StringValue(config.ContainerName.ValueString() + "/" + config.ObjectName.ValueString())
	config.ProjectID = types.StringValue(scope.ProjectID)
	config.AvailabilityZone = types.StringValue(scope.AvailabilityZone)
	config.ContentBase64 = types.StringValue(base64.StdEncoding.EncodeToString(content))
	config.Bytes = types.Int64Value(int64(len(content)))
	config.SHA256 = types.StringValue(hex.EncodeToString(digest[:]))
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

var _ datasource.DataSourceWithConfigure = &objectContentDataSource{}
