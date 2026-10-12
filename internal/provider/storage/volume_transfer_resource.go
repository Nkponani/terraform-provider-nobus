// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package storage

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

// volume transfer
type volumeTransferResource struct{ config *shared.Config }
type volumeTransferModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	VolumeID         types.String `tfsdk:"volume_id"`
	Name             types.String `tfsdk:"name"`
	CreatedAt        types.String `tfsdk:"created_at"`
	AuthKey          types.String `tfsdk:"auth_key"`
}

func NewVolumeTransferResource() resource.Resource { return &volumeTransferResource{} }
func (r *volumeTransferResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_transfer"
}
func (r *volumeTransferResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *volumeTransferResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A one-time Nobus volume ownership transfer.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "volume_id": schema.StringAttribute{Required: true, PlanModifiers: immutableString()}, "name": schema.StringAttribute{Optional: true, PlanModifiers: immutableString()}, "created_at": schema.StringAttribute{Computed: true}, "auth_key": schema.StringAttribute{Computed: true, Sensitive: true}})}
}
func (r *volumeTransferResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p volumeTransferModel
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
	v, e := c.CreateVolumeTransfer(ctx, client.CreateVolumeTransferRequest{VolumeID: p.VolumeID.ValueString(), Name: shared.OptionalString(p.Name), ProjectID: s.ProjectID, AvailabilityZone: s.AvailabilityZone})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus volume transfer", shared.DiagnosticDetail(e))
		return
	}
	applyVolumeTransfer(v, &p)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *volumeTransferResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s volumeTransferModel
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
	v, e := c.GetVolumeTransfer(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus volume transfer", shared.DiagnosticDetail(e))
		return
	}
	key := s.AuthKey
	applyVolumeTransfer(v, &s)
	if v.AuthKey == nil {
		s.AuthKey = key
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *volumeTransferResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Volume transfers cannot be updated", "Replace the transfer to change it.")
}
func (r *volumeTransferResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s volumeTransferModel
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
	if e := c.DeleteVolumeTransfer(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus volume transfer", shared.DiagnosticDetail(e))
	}
}
func (r *volumeTransferResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shared.ImportID(ctx, req.ID, resp)
}
func applyVolumeTransfer(v *client.VolumeTransfer, m *volumeTransferModel) {
	m.ID = types.StringValue(v.ID)
	m.VolumeID = types.StringValue(v.VolumeID)
	m.Name = shared.StringOrNull(v.Name)
	m.CreatedAt = shared.StringOrNull(v.CreatedAt)
	m.AuthKey = shared.StringOrNull(v.AuthKey)
}

var _ resource.ResourceWithConfigure = &volumeTransferResource{}
var _ resource.ResourceWithImportState = &volumeTransferResource{}
