// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package storage

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type volumeBackupDataSource struct{ config *shared.Config }
type volumeBackupDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	VolumeID         types.String `tfsdk:"volume_id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Size             types.Int64  `tfsdk:"size"`
	Container        types.String `tfsdk:"container"`
	Status           types.String `tfsdk:"status"`
	CreatedAt        types.String `tfsdk:"created_at"`
	Metadata         types.Map    `tfsdk:"metadata"`
}

func NewVolumeBackupDataSource() datasource.DataSource { return &volumeBackupDataSource{} }
func (r *volumeBackupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_backup"
}
func (r *volumeBackupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *volumeBackupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Reads an existing Nobus volume backup.", Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Required: true}, "project_id": schema.StringAttribute{Optional: true, Computed: true}, "availability_zone": schema.StringAttribute{Optional: true, Computed: true}, "volume_id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Computed: true}, "description": schema.StringAttribute{Computed: true}, "size": schema.Int64Attribute{Computed: true}, "container": schema.StringAttribute{Computed: true}, "status": schema.StringAttribute{Computed: true}, "created_at": schema.StringAttribute{Computed: true}, "metadata": schema.MapAttribute{Computed: true, ElementType: types.StringType}}}
}
func (r *volumeBackupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var p volumeBackupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	s, ok := r.config.ResolveScope(p.ProjectID, p.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	v, e := c.GetBackup(ctx, s.ProjectID, s.AvailabilityZone, p.ID.ValueString())
	if e != nil {
		resp.Diagnostics.AddError("Unable to read Nobus volume backup", shared.DiagnosticDetail(e))
		return
	}
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	p.VolumeID = types.StringValue(v.VolumeID)
	p.Name = shared.StringOrNull(v.Name)
	p.Description = shared.StringOrNull(v.Description)
	p.Size = shared.Int64OrNull(v.Size)
	p.Container = shared.StringOrNull(v.Container)
	p.Status = shared.StringOrNull(v.Status)
	p.CreatedAt = shared.StringOrNull(v.CreatedAt)
	if v.Metadata != nil {
		x, d := types.MapValueFrom(ctx, types.StringType, v.Metadata)
		resp.Diagnostics.Append(d...)
		p.Metadata = x
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}

var _ datasource.DataSourceWithConfigure = &volumeBackupDataSource{}
