// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package storage

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

// volume snapshot.
type volumeSnapshotResource struct{ config *shared.Config }
type volumeSnapshotModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	VolumeID         types.String `tfsdk:"volume_id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Force            types.Bool   `tfsdk:"force"`
	Metadata         types.Map    `tfsdk:"metadata"`
	Status           types.String `tfsdk:"status"`
	Size             types.Int64  `tfsdk:"size"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

func NewVolumeSnapshotResource() resource.Resource { return &volumeSnapshotResource{} }
func (r *volumeSnapshotResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_snapshot"
}
func (r *volumeSnapshotResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *volumeSnapshotResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A point-in-time snapshot of a Nobus volume.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "volume_id": schema.StringAttribute{Required: true, PlanModifiers: immutableString()}, "name": schema.StringAttribute{Optional: true, PlanModifiers: immutableString()},
		"description": schema.StringAttribute{Optional: true, PlanModifiers: immutableString()}, "force": schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
		"metadata": schema.MapAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()}}, "status": schema.StringAttribute{Computed: true},
		"size": schema.Int64Attribute{Computed: true}, "created_at": schema.StringAttribute{Computed: true}})}
}
func (r *volumeSnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p volumeSnapshotModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
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
	md, d := shared.MapToStringMap(ctx, p.Metadata)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, e := c.CreateSnapshot(ctx, client.CreateSnapshotRequest{VolumeID: p.VolumeID.ValueString(), Name: shared.OptionalString(p.Name), Description: shared.OptionalString(p.Description), Force: shared.OptionalBool(p.Force), Metadata: md, ProjectID: s.ProjectID, AvailabilityZone: s.AvailabilityZone})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus volume snapshot", shared.DiagnosticDetail(e))
		return
	}
	applySnapshot(ctx, v, &p, &resp.Diagnostics)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *volumeSnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s volumeSnapshotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.ResolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	v, e := c.GetSnapshot(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus volume snapshot", shared.DiagnosticDetail(e))
		return
	}
	applySnapshot(ctx, v, &s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *volumeSnapshotResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Volume snapshots cannot be updated", "Replace the snapshot to change its configuration.")
}
func (r *volumeSnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s volumeSnapshotModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.ResolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if e := c.DeleteSnapshot(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus volume snapshot", shared.DiagnosticDetail(e))
	}
}
func (r *volumeSnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shared.ImportID(ctx, req.ID, resp)
}
func applySnapshot(ctx context.Context, v *client.Snapshot, m *volumeSnapshotModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.VolumeID = types.StringValue(v.VolumeID)
	m.Name = shared.StringOrNull(v.Name)
	m.Description = shared.StringOrNull(v.Description)
	m.Status = shared.StringOrNull(v.Status)
	m.Size = shared.Int64OrNull(v.Size)
	m.CreatedAt = shared.StringOrNull(v.CreatedAt)
	if v.Metadata != nil {
		x, ds := types.MapValueFrom(ctx, types.StringType, v.Metadata)
		d.Append(ds...)
		m.Metadata = x
	}
}

var _ resource.ResourceWithConfigure = &volumeSnapshotResource{}
var _ resource.ResourceWithImportState = &volumeSnapshotResource{}
