// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networking

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type routerResource struct{ config *shared.Config }
type routerModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	AvailabilityZone  types.String `tfsdk:"availability_zone"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	AdminStateUp      types.Bool   `tfsdk:"admin_state_up"`
	Status            types.String `tfsdk:"status"`
	ExternalNetworkID types.String `tfsdk:"external_network_id"`
	EnableSNAT        types.Bool   `tfsdk:"enable_snat"`
	Tags              types.Set    `tfsdk:"tags"`
	ExternalFixedIPs  types.List   `tfsdk:"external_fixed_ips"`
}

func NewRouterResource() resource.Resource { return &routerResource{} }
func (r *routerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_router"
}
func (r *routerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *routerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A standalone Nobus network router.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}, "name": schema.StringAttribute{Required: true, Validators: []validator.String{shared.StringNotEmpty()}}, "description": schema.StringAttribute{Optional: true}, "admin_state_up": schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}, "status": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}, "external_network_id": schema.StringAttribute{Optional: true}, "enable_snat": schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}, "external_fixed_ips": schema.ListAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()}, ElementType: types.StringType}, "tags": schema.SetAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}, ElementType: types.StringType}})}
}
func (r *routerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p routerModel
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
	fixed, fd := shared.ListToStringSlice(ctx, p.ExternalFixedIPs)
	resp.Diagnostics.Append(fd...)
	fixedAPI := make([]client.ExternalFixedIP, 0, len(fixed))
	for _, ip := range fixed {
		fixedAPI = append(fixedAPI, client.ExternalFixedIP{IPAddress: ip})
	}
	var gateway *client.ExternalGatewayInfo
	if shared.NullSafeString(p.ExternalNetworkID) != "" {
		gateway = &client.ExternalGatewayInfo{NetworkID: p.ExternalNetworkID.ValueString(), EnableSNAT: shared.OptionalBool(p.EnableSNAT), ExternalFixedIPs: fixedAPI}
	}
	v, e := c.CreateRouter(ctx, s.ProjectID, s.AvailabilityZone, client.RouterCreate{Name: p.Name.ValueString(), Description: shared.OptionalString(p.Description), AdminStateUp: shared.OptionalBool(p.AdminStateUp), Tags: tags, Gateway: gateway})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus router", shared.DiagnosticDetail(e))
		return
	}
	applyRouter(ctx, v, &p, &resp.Diagnostics)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *routerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s routerModel
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
	v, e := c.GetRouter(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus router", shared.DiagnosticDetail(e))
		return
	}
	applyRouter(ctx, v, &s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *routerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p, old routerModel
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
	if shared.NullSafeString(old.ExternalNetworkID) != "" && shared.NullSafeString(p.ExternalNetworkID) == "" {
		if e := c.ClearRouterGateway(ctx, s.ProjectID, s.AvailabilityZone, p.ID.ValueString()); e != nil {
			resp.Diagnostics.AddError("Unable to clear Nobus router gateway", shared.DiagnosticDetail(e))
			return
		}
	}
	tags, d := shared.SetToStringSlice(ctx, p.Tags)
	resp.Diagnostics.Append(d...)
	fixed, fd := shared.ListToStringSlice(ctx, p.ExternalFixedIPs)
	resp.Diagnostics.Append(fd...)
	fixedAPI := make([]client.ExternalFixedIP, 0, len(fixed))
	for _, ip := range fixed {
		fixedAPI = append(fixedAPI, client.ExternalFixedIP{IPAddress: ip})
	}
	var gateway *client.ExternalGatewayInfo
	if shared.NullSafeString(p.ExternalNetworkID) != "" {
		gateway = &client.ExternalGatewayInfo{NetworkID: p.ExternalNetworkID.ValueString(), EnableSNAT: shared.OptionalBool(p.EnableSNAT), ExternalFixedIPs: fixedAPI}
	}
	v, e := c.UpdateRouter(ctx, s.ProjectID, s.AvailabilityZone, p.ID.ValueString(), client.RouterUpdate{Name: shared.OptionalString(p.Name), Description: shared.OptionalString(p.Description), AdminStateUp: shared.OptionalBool(p.AdminStateUp), ExternalGatewayInfo: gateway, Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to update Nobus router", shared.DiagnosticDetail(e))
		return
	}
	applyRouter(ctx, v, &p, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *routerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s routerModel
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
	if e := c.DeleteRouter(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus router", shared.DiagnosticDetail(e))
	}
}
func (r *routerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shared.ImportID(ctx, req.ID, resp)
}
func applyRouter(ctx context.Context, v *client.Router, m *routerModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.Name = shared.StringOrNull(v.Name)
	m.Description = shared.StringOrNull(v.Description)
	m.AdminStateUp = shared.BoolOrNull(v.AdminStateUp)
	m.Status = shared.StringOrNull(v.Status)
	if v.Gateway != nil {
		m.ExternalNetworkID = types.StringValue(v.Gateway.NetworkID)
		m.EnableSNAT = shared.BoolOrNull(v.Gateway.EnableSNAT)
		fixed := make([]string, 0, len(v.Gateway.ExternalFixedIPs))
		for _, ip := range v.Gateway.ExternalFixedIPs {
			fixed = append(fixed, ip.IPAddress)
		}
		x, ds := types.ListValueFrom(ctx, types.StringType, fixed)
		d.Append(ds...)
		m.ExternalFixedIPs = x
	} else {
		m.ExternalNetworkID = types.StringNull()
		m.EnableSNAT = types.BoolNull()
		m.ExternalFixedIPs = types.ListNull(types.StringType)
	}
	if v.Tags != nil {
		x, ds := types.SetValueFrom(ctx, types.StringType, v.Tags)
		d.Append(ds...)
		m.Tags = x
	}
}

var _ resource.ResourceWithConfigure = &routerResource{}
var _ resource.ResourceWithImportState = &routerResource{}
