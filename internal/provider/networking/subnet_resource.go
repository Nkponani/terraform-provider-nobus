// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networking

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type subnetResource struct{ config *shared.Config }
type subnetModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	NetworkID        types.String `tfsdk:"network_id"`
	CIDR             types.String `tfsdk:"cidr"`
	IPVersion        types.Int64  `tfsdk:"ip_version"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	GatewayIP        types.String `tfsdk:"gateway_ip"`
	EnableDHCP       types.Bool   `tfsdk:"enable_dhcp"`
	DNSNameservers   types.List   `tfsdk:"dns_nameservers"`
	Tags             types.Set    `tfsdk:"tags"`
	HostRoutes       types.List   `tfsdk:"host_routes"`
	AllocationPools  types.List   `tfsdk:"allocation_pools"`
}

func NewSubnetResource() resource.Resource { return &subnetResource{} }
func (r *subnetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet"
}
func (r *subnetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *subnetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A standalone Nobus network subnet.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}, "network_id": schema.StringAttribute{Required: true, Validators: []validator.String{shared.StringNotEmpty()}, PlanModifiers: shared.ReplaceStrings()}, "cidr": schema.StringAttribute{Required: true, Validators: []validator.String{shared.StringNotEmpty()}, PlanModifiers: shared.ReplaceStrings()}, "ip_version": schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}}, "name": schema.StringAttribute{Optional: true}, "description": schema.StringAttribute{Optional: true}, "gateway_ip": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}, "enable_dhcp": schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}, "dns_nameservers": schema.ListAttribute{Optional: true, ElementType: types.StringType}, "host_routes": schema.ListNestedAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()}, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"destination": schema.StringAttribute{Required: true}, "next_hop": schema.StringAttribute{Required: true}}}}, "allocation_pools": schema.ListNestedAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()}, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"start": schema.StringAttribute{Required: true}, "end": schema.StringAttribute{Required: true}}}}, "tags": schema.SetAttribute{Computed: true, PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()}, ElementType: types.StringType}})}
}
func (r *subnetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p subnetModel
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
	dns, d := shared.ListToStringSlice(ctx, p.DNSNameservers)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	routes, rd := hostRoutesFromList(ctx, p.HostRoutes)
	pools, pd := allocationPoolsFromList(ctx, p.AllocationPools)
	resp.Diagnostics.Append(rd...)
	resp.Diagnostics.Append(pd...)
	v, e := c.CreateSubnet(ctx, s.ProjectID, s.AvailabilityZone, client.SubnetCreate{NetworkID: p.NetworkID.ValueString(), CIDR: p.CIDR.ValueString(), IPVersion: p.IPVersion.ValueInt64(), Name: shared.OptionalString(p.Name), Description: shared.OptionalString(p.Description), GatewayIP: shared.OptionalString(p.GatewayIP), EnableDHCP: shared.OptionalBool(p.EnableDHCP), DNSNameservers: dns, HostRoutes: routes, AllocationPools: pools})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus subnet", shared.DiagnosticDetail(e))
		return
	}
	applySubnet(ctx, v, &p, &resp.Diagnostics)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *subnetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s subnetModel
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
	v, e := c.GetSubnet(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus subnet", shared.DiagnosticDetail(e))
		return
	}
	applySubnet(ctx, v, &s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *subnetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p subnetModel
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
	dns, d := shared.ListToStringSlice(ctx, p.DNSNameservers)
	resp.Diagnostics.Append(d...)
	routes, rd := hostRoutesFromList(ctx, p.HostRoutes)
	pools, pd := allocationPoolsFromList(ctx, p.AllocationPools)
	resp.Diagnostics.Append(rd...)
	resp.Diagnostics.Append(pd...)
	v, e := c.UpdateSubnet(ctx, s.ProjectID, s.AvailabilityZone, p.ID.ValueString(), client.SubnetUpdate{Name: shared.OptionalString(p.Name), Description: shared.OptionalString(p.Description), GatewayIP: shared.OptionalString(p.GatewayIP), EnableDHCP: shared.OptionalBool(p.EnableDHCP), DNSNameservers: dns, HostRoutes: routes, AllocationPools: pools})
	if e != nil {
		resp.Diagnostics.AddError("Unable to update Nobus subnet", shared.DiagnosticDetail(e))
		return
	}
	applySubnet(ctx, v, &p, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *subnetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s subnetModel
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
	if e := c.DeleteSubnet(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus subnet", shared.DiagnosticDetail(e))
	}
}
func (r *subnetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shared.ImportID(ctx, req.ID, resp)
}
func applySubnet(ctx context.Context, v *client.Subnet, m *subnetModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.NetworkID = types.StringValue(v.NetworkID)
	m.CIDR = types.StringValue(v.CIDR)
	m.IPVersion = types.Int64Value(v.IPVersion)
	m.Name = shared.StringOrNull(v.Name)
	m.Description = shared.StringOrNull(v.Description)
	m.GatewayIP = shared.StringOrNull(v.GatewayIP)
	m.EnableDHCP = shared.BoolOrNull(v.EnableDHCP)
	if len(v.DNSNameservers) > 0 {
		x, ds := types.ListValueFrom(ctx, types.StringType, v.DNSNameservers)
		d.Append(ds...)
		m.DNSNameservers = x
	}
	if v.Tags != nil {
		x, ds := types.SetValueFrom(ctx, types.StringType, v.Tags)
		d.Append(ds...)
		m.Tags = x
	}
	if v.HostRoutes != nil {
		x, ds := hostRoutesToList(ctx, v.HostRoutes)
		d.Append(ds...)
		m.HostRoutes = x
	} else if m.HostRoutes.IsUnknown() {
		m.HostRoutes = types.ListNull(hostRouteType)
	}
	if v.AllocationPools != nil {
		x, ds := allocationPoolsToList(ctx, v.AllocationPools)
		d.Append(ds...)
		m.AllocationPools = x
	} else if m.AllocationPools.IsUnknown() {
		m.AllocationPools = types.ListNull(allocationPoolType)
	}
}

var _ resource.ResourceWithConfigure = &subnetResource{}
var _ resource.ResourceWithImportState = &subnetResource{}
