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

type standaloneNetworkResource struct{ config *shared.Config }
type standaloneNetworkModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	AdminStateUp     types.Bool   `tfsdk:"admin_state_up"`
	Shared           types.Bool   `tfsdk:"shared"`
	MTU              types.Int64  `tfsdk:"mtu"`
	RouterExternal   types.Bool   `tfsdk:"router_external"`
	PortSecurity     types.Bool   `tfsdk:"port_security_enabled"`
	Tags             types.Set    `tfsdk:"tags"`
	Status           types.String `tfsdk:"status"`
	SubnetIDs        types.List   `tfsdk:"subnet_ids"`
}

func NewStandaloneNetworkResource() resource.Resource { return &standaloneNetworkResource{} }
func (r *standaloneNetworkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_standalone_network"
}
func (r *standaloneNetworkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *standaloneNetworkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A standalone Nobus network created without an automatic subnet or router.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true}, "description": schema.StringAttribute{Optional: true}, "admin_state_up": schema.BoolAttribute{Optional: true, Computed: true}, "shared": schema.BoolAttribute{Optional: true, Computed: true}, "mtu": schema.Int64Attribute{Optional: true, Computed: true}, "router_external": schema.BoolAttribute{Optional: true, Computed: true}, "port_security_enabled": schema.BoolAttribute{Optional: true, Computed: true}, "tags": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType}, "status": schema.StringAttribute{Computed: true}, "subnet_ids": schema.ListAttribute{Computed: true, ElementType: types.StringType}})}
}
func (r *standaloneNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p standaloneNetworkModel
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
	v, e := c.CreateStandaloneNetwork(ctx, s.ProjectID, s.AvailabilityZone, client.StandaloneNetworkCreate{Name: p.Name.ValueString(), Description: shared.OptionalString(p.Description), AdminStateUp: shared.OptionalBool(p.AdminStateUp), Shared: shared.OptionalBool(p.Shared), MTU: shared.OptionalInt64(p.MTU), RouterExternal: shared.OptionalBool(p.RouterExternal), PortSecurityEnabled: shared.OptionalBool(p.PortSecurity), Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create standalone Nobus network", shared.DiagnosticDetail(e))
		return
	}
	applyStandaloneNetwork(ctx, v, &p, &resp.Diagnostics)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *standaloneNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s standaloneNetworkModel
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
	v, e := c.GetNetwork(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read standalone Nobus network", shared.DiagnosticDetail(e))
		return
	}
	applyStandaloneNetwork(ctx, v, &s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *standaloneNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p standaloneNetworkModel
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
	v, e := c.UpdateNetwork(ctx, client.NetworkUpdateRequest{ProjectID: s.ProjectID, AvailabilityZone: s.AvailabilityZone, NetworkID: p.ID.ValueString(), Network: client.NetworkUpdate{Name: shared.OptionalString(p.Name), Description: shared.OptionalString(p.Description), AdminStateUp: shared.OptionalBool(p.AdminStateUp), Shared: shared.OptionalBool(p.Shared), MTU: shared.OptionalInt64(p.MTU), RouterExternal: shared.OptionalBool(p.RouterExternal), PortSecurityEnabled: shared.OptionalBool(p.PortSecurity), Tags: &tags}})
	if e != nil {
		resp.Diagnostics.AddError("Unable to update standalone Nobus network", shared.DiagnosticDetail(e))
		return
	}
	applyStandaloneNetwork(ctx, v, &p, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *standaloneNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s standaloneNetworkModel
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
	if e := c.DeleteNetwork(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete standalone Nobus network", shared.DiagnosticDetail(e))
	}
}
func (r *standaloneNetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shared.ImportID(ctx, req.ID, resp)
}
func applyStandaloneNetwork(ctx context.Context, v *client.Network, m *standaloneNetworkModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.Name = shared.StringOrNull(v.Name)
	m.Description = shared.StringOrNull(v.Description)
	m.AdminStateUp = shared.BoolOrNull(v.AdminStateUp)
	m.Shared = shared.BoolOrNull(v.Shared)
	m.MTU = shared.Int64OrNull(v.MTU)
	m.RouterExternal = shared.BoolOrNull(v.RouterExternal)
	m.PortSecurity = shared.BoolOrNull(v.PortSecurityEnabled)
	m.Status = shared.StringOrNull(v.Status)
	if v.Tags != nil {
		x, ds := types.SetValueFrom(ctx, types.StringType, v.Tags)
		d.Append(ds...)
		m.Tags = x
	}
	if v.Subnets != nil {
		x, ds := types.ListValueFrom(ctx, types.StringType, v.Subnets)
		d.Append(ds...)
		m.SubnetIDs = x
	}
}

var _ resource.ResourceWithConfigure = &standaloneNetworkResource{}
var _ resource.ResourceWithImportState = &standaloneNetworkResource{}
