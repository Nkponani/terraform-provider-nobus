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

type firewallGroupResource struct{ config *shared.Config }
type firewallGroupModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	AdminStateUp     types.Bool   `tfsdk:"admin_state_up"`
	IngressPolicyID  types.String `tfsdk:"ingress_firewall_policy_id"`
	EgressPolicyID   types.String `tfsdk:"egress_firewall_policy_id"`
	Ports            types.List   `tfsdk:"ports"`
	Tags             types.Set    `tfsdk:"tags"`
}

func NewFirewallGroupResource() resource.Resource { return &firewallGroupResource{} }
func (r *firewallGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall_group"
}
func (r *firewallGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *firewallGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus network firewall group.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true}, "description": schema.StringAttribute{Optional: true}, "admin_state_up": schema.BoolAttribute{Optional: true, Computed: true}, "ingress_firewall_policy_id": schema.StringAttribute{Optional: true}, "egress_firewall_policy_id": schema.StringAttribute{Optional: true}, "ports": schema.ListAttribute{Optional: true, ElementType: types.StringType}, "tags": schema.SetAttribute{Optional: true, ElementType: types.StringType}})}
}
func (r *firewallGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p firewallGroupModel
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
	ports, d1 := shared.ListToStringSlice(ctx, p.Ports)
	tags, d2 := shared.SetToStringSlice(ctx, p.Tags)
	resp.Diagnostics.Append(d1...)
	resp.Diagnostics.Append(d2...)
	v, e := c.CreateFirewallGroup(ctx, s.ProjectID, s.AvailabilityZone, client.FirewallGroupCreate{Name: p.Name.ValueString(), Description: shared.OptionalString(p.Description), AdminStateUp: shared.OptionalBool(p.AdminStateUp), IngressFirewallPolicyID: shared.OptionalString(p.IngressPolicyID), EgressFirewallPolicyID: shared.OptionalString(p.EgressPolicyID), Ports: ports, Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus firewall group", shared.DiagnosticDetail(e))
		return
	}
	applyFirewallGroup(ctx, v, &p, &resp.Diagnostics)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *firewallGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s firewallGroupModel
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
	v, e := c.GetFirewallGroup(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus firewall group", shared.DiagnosticDetail(e))
		return
	}
	applyFirewallGroup(ctx, v, &s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *firewallGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p firewallGroupModel
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
	ports, d1 := shared.ListToStringSlice(ctx, p.Ports)
	tags, d2 := shared.SetToStringSlice(ctx, p.Tags)
	resp.Diagnostics.Append(d1...)
	resp.Diagnostics.Append(d2...)
	v, e := c.UpdateFirewallGroup(ctx, s.ProjectID, s.AvailabilityZone, p.ID.ValueString(), client.FirewallGroupUpdate{Name: shared.OptionalString(p.Name), Description: shared.OptionalString(p.Description), AdminStateUp: shared.OptionalBool(p.AdminStateUp), IngressFirewallPolicyID: shared.OptionalString(p.IngressPolicyID), EgressFirewallPolicyID: shared.OptionalString(p.EgressPolicyID), Ports: ports, Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to update Nobus firewall group", shared.DiagnosticDetail(e))
		return
	}
	applyFirewallGroup(ctx, v, &p, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *firewallGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s firewallGroupModel
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
	if e := c.DeleteFirewallGroup(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus firewall group", shared.DiagnosticDetail(e))
	}
}
func (r *firewallGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shared.ImportID(ctx, req.ID, resp)
}
func applyFirewallGroup(ctx context.Context, v *client.FirewallGroup, m *firewallGroupModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.Name = shared.StringOrNull(v.Name)
	m.Description = shared.StringOrNull(v.Description)
	m.AdminStateUp = shared.BoolOrNull(v.AdminStateUp)
	m.IngressPolicyID = shared.StringOrNull(v.IngressFirewallPolicyID)
	m.EgressPolicyID = shared.StringOrNull(v.EgressFirewallPolicyID)
	if v.Ports != nil {
		x, ds := types.ListValueFrom(ctx, types.StringType, v.Ports)
		d.Append(ds...)
		m.Ports = x
	}
	if v.Tags != nil {
		x, ds := types.SetValueFrom(ctx, types.StringType, v.Tags)
		d.Append(ds...)
		m.Tags = x
	}
}

var _ resource.ResourceWithConfigure = &firewallGroupResource{}
var _ resource.ResourceWithImportState = &firewallGroupResource{}
