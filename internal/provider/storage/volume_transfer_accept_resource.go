// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package storage

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

func actionString() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace()}
}

type volumeTransferAcceptResource struct{ config *shared.Config }
type volumeTransferAcceptModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	TransferID       types.String `tfsdk:"transfer_id"`
	AuthKey          types.String `tfsdk:"auth_key"`
	VolumeID         types.String `tfsdk:"volume_id"`
}

func NewVolumeTransferAcceptResource() resource.Resource { return &volumeTransferAcceptResource{} }
func (r *volumeTransferAcceptResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_transfer_accept"
}
func (r *volumeTransferAcceptResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *volumeTransferAcceptResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Accepts a Nobus volume transfer. This is an action resource.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id":          schema.StringAttribute{Computed: true},
		"transfer_id": schema.StringAttribute{Required: true, PlanModifiers: actionString()},
		"auth_key":    schema.StringAttribute{Required: true, Sensitive: true, PlanModifiers: actionString()},
		"volume_id":   schema.StringAttribute{Computed: true},
	})}
}
func (r *volumeTransferAcceptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p volumeTransferAcceptModel
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
	v, err := c.AcceptVolumeTransfer(ctx, client.AcceptVolumeTransferRequest{TransferID: p.TransferID.ValueString(), AuthKey: p.AuthKey.ValueString(), ProjectID: s.ProjectID, AvailabilityZone: s.AvailabilityZone})
	if err != nil {
		resp.Diagnostics.AddError("Unable to accept Nobus volume transfer", shared.DiagnosticDetail(err))
		return
	}
	p.ID = types.StringValue(v.ID)
	p.VolumeID = types.StringValue(v.VolumeID)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *volumeTransferAcceptResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {
}
func (r *volumeTransferAcceptResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (r *volumeTransferAcceptResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

var _ resource.ResourceWithConfigure = &volumeTransferAcceptResource{}
