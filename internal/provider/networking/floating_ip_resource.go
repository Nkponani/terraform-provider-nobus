// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networking

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type floatingIPResource struct{ config *shared.Config }
type floatingIPModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	AvailabilityZone  types.String `tfsdk:"availability_zone"`
	FloatingNetworkID types.String `tfsdk:"floating_network_id"`
	PortID            types.String `tfsdk:"port_id"`
	FixedIPAddress    types.String `tfsdk:"fixed_ip_address"`
	Description       types.String `tfsdk:"description"`
	FloatingIPAddress types.String `tfsdk:"floating_ip_address"`
	Status            types.String `tfsdk:"status"`
	Tags              types.Set    `tfsdk:"tags"`
}

func NewFloatingIPResource() resource.Resource { return &floatingIPResource{} }
func (r *floatingIPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_floating_ip"
}
func (r *floatingIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *floatingIPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus floating IP allocation and optional port association.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "floating_network_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "port_id": schema.StringAttribute{Optional: true}, "fixed_ip_address": schema.StringAttribute{Optional: true}, "description": schema.StringAttribute{Optional: true}, "tags": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType}, "floating_ip_address": schema.StringAttribute{Computed: true}, "status": schema.StringAttribute{Computed: true}})}
}
func (r *floatingIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p floatingIPModel
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
	tags, d := shared.SetToStringSlice(ctx, p.Tags)
	resp.Diagnostics.Append(d...)
	v, e := c.CreateFloatingIP(ctx, s.ProjectID, s.AvailabilityZone, client.FloatingIPCreate{FloatingNetworkID: p.FloatingNetworkID.ValueString(), PortID: shared.OptionalString(p.PortID), FixedIPAddress: shared.OptionalString(p.FixedIPAddress), Description: shared.OptionalString(p.Description), Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus floating IP", shared.DiagnosticDetail(e))
		return
	}
	applyFloatingIP(ctx, v, &p, &resp.Diagnostics)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *floatingIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s floatingIPModel
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
	v, e := c.GetFloatingIP(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus floating IP", shared.DiagnosticDetail(e))
		return
	}
	applyFloatingIP(ctx, v, &s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *floatingIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p, old floatingIPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
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
	if shared.NullSafeString(old.PortID) != "" && shared.NullSafeString(p.PortID) == "" {
		if e := c.DisassociateFloatingIP(ctx, s.ProjectID, s.AvailabilityZone, p.ID.ValueString()); e != nil {
			resp.Diagnostics.AddError("Unable to disassociate Nobus floating IP", shared.DiagnosticDetail(e))
			return
		}
	}
	tags, d := shared.SetToStringSlice(ctx, p.Tags)
	resp.Diagnostics.Append(d...)
	v, e := c.UpdateFloatingIP(ctx, s.ProjectID, s.AvailabilityZone, p.ID.ValueString(), client.FloatingIPUpdate{PortID: shared.OptionalString(p.PortID), FixedIPAddress: shared.OptionalString(p.FixedIPAddress), Description: shared.OptionalString(p.Description), Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to update Nobus floating IP", shared.DiagnosticDetail(e))
		return
	}
	applyFloatingIP(ctx, v, &p, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *floatingIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s floatingIPModel
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
	if e := c.DeleteFloatingIP(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus floating IP", shared.DiagnosticDetail(e))
	}
}
func (r *floatingIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shared.ImportID(ctx, req.ID, resp)
}
func applyFloatingIP(ctx context.Context, v *client.FloatingIP, m *floatingIPModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.FloatingNetworkID = types.StringValue(v.FloatingNetworkID)
	m.PortID = shared.StringOrNull(v.PortID)
	m.FixedIPAddress = shared.StringOrNull(v.FixedIPAddress)
	m.Description = shared.StringOrNull(v.Description)
	m.FloatingIPAddress = shared.StringOrNull(v.FloatingIPAddress)
	m.Status = shared.StringOrNull(v.Status)
	if v.Tags != nil {
		x, ds := types.SetValueFrom(ctx, types.StringType, v.Tags)
		d.Append(ds...)
		m.Tags = x
	}
}

var _ resource.ResourceWithConfigure = &floatingIPResource{}
var _ resource.ResourceWithImportState = &floatingIPResource{}
