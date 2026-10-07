// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

func replaceStrings() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()}
}
func scopeAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"project_id":        schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replaceStrings()},
		"availability_zone": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replaceStrings()},
	}
}
func mergeAttributes(base map[string]schema.Attribute, more map[string]schema.Attribute) map[string]schema.Attribute {
	for k, v := range more {
		base[k] = v
	}
	return base
}
func importID(ctx context.Context, id string, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

type subnetResource struct{ config *providerConfig }
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
}

func NewSubnetResource() resource.Resource { return &subnetResource{} }
func (r *subnetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subnet"
}
func (r *subnetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *subnetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A standalone Nobus network subnet.", Attributes: mergeAttributes(scopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "network_id": schema.StringAttribute{Required: true, Validators: []validator.String{StringNotEmpty()}, PlanModifiers: replaceStrings()}, "cidr": schema.StringAttribute{Required: true, Validators: []validator.String{StringNotEmpty()}, PlanModifiers: replaceStrings()}, "ip_version": schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}}, "name": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "description": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "gateway_ip": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "enable_dhcp": schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}}, "dns_nameservers": schema.ListAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()}}})}
}
func (r *subnetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p subnetModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	s, ok := r.config.resolveScope(p.ProjectID, p.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	dns, d := listToStringSlice(ctx, p.DNSNameservers)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, e := c.CreateSubnet(ctx, s.ProjectID, s.AvailabilityZone, client.SubnetCreate{NetworkID: p.NetworkID.ValueString(), CIDR: p.CIDR.ValueString(), IPVersion: p.IPVersion.ValueInt64(), Name: optionalString(p.Name), Description: optionalString(p.Description), GatewayIP: optionalString(p.GatewayIP), EnableDHCP: optionalBool(p.EnableDHCP), DNSNameservers: dns})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus subnet", diagnosticDetail(e))
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
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.resolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	v, e := c.GetSubnet(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus subnet", diagnosticDetail(e))
		return
	}
	applySubnet(ctx, v, &s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *subnetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Subnet update is not yet supported", "Replace the subnet to apply this change.")
}
func (r *subnetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s subnetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.resolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if e := c.DeleteSubnet(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus subnet", diagnosticDetail(e))
	}
}
func (r *subnetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID(ctx, req.ID, resp)
}
func applySubnet(ctx context.Context, v *client.Subnet, m *subnetModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.NetworkID = types.StringValue(v.NetworkID)
	m.CIDR = types.StringValue(v.CIDR)
	m.IPVersion = types.Int64Value(v.IPVersion)
	m.Name = stringOrNull(v.Name)
	m.Description = stringOrNull(v.Description)
	m.GatewayIP = stringOrNull(v.GatewayIP)
	m.EnableDHCP = boolOrNull(v.EnableDHCP)
	if v.DNSNameservers != nil {
		x, ds := types.ListValueFrom(ctx, types.StringType, v.DNSNameservers)
		d.Append(ds...)
		m.DNSNameservers = x
	}
}

type routerResource struct{ config *providerConfig }
type routerModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	AdminStateUp     types.Bool   `tfsdk:"admin_state_up"`
	Status           types.String `tfsdk:"status"`
}

func NewRouterResource() resource.Resource { return &routerResource{} }
func (r *routerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_router"
}
func (r *routerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *routerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A standalone Nobus network router.", Attributes: mergeAttributes(scopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true, Validators: []validator.String{StringNotEmpty()}, PlanModifiers: replaceStrings()}, "description": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "admin_state_up": schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}}, "status": schema.StringAttribute{Computed: true}})}
}
func (r *routerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p routerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	s, ok := r.config.resolveScope(p.ProjectID, p.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	v, e := c.CreateRouter(ctx, s.ProjectID, s.AvailabilityZone, client.RouterCreate{Name: p.Name.ValueString(), Description: optionalString(p.Description), AdminStateUp: optionalBool(p.AdminStateUp)})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus router", diagnosticDetail(e))
		return
	}
	applyRouter(v, &p)
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
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.resolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	v, e := c.GetRouter(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus router", diagnosticDetail(e))
		return
	}
	applyRouter(v, &s)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *routerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Router update is not yet supported", "Replace the router to apply this change.")
}
func (r *routerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s routerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.resolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if e := c.DeleteRouter(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus router", diagnosticDetail(e))
	}
}
func (r *routerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID(ctx, req.ID, resp)
}
func applyRouter(v *client.Router, m *routerModel) {
	m.ID = types.StringValue(v.ID)
	m.Name = stringOrNull(v.Name)
	m.Description = stringOrNull(v.Description)
	m.AdminStateUp = boolOrNull(v.AdminStateUp)
	m.Status = stringOrNull(v.Status)
}

type routerInterfaceResource struct{ config *providerConfig }
type routerInterfaceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	RouterID         types.String `tfsdk:"router_id"`
	SubnetID         types.String `tfsdk:"subnet_id"`
	PortID           types.String `tfsdk:"port_id"`
}

func NewRouterInterfaceResource() resource.Resource { return &routerInterfaceResource{} }
func (r *routerInterfaceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_router_interface"
}
func (r *routerInterfaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *routerInterfaceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Connects a Nobus router to a subnet or port.", Attributes: mergeAttributes(scopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "router_id": schema.StringAttribute{Required: true, PlanModifiers: replaceStrings()}, "subnet_id": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "port_id": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}})}
}
func (r *routerInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p routerInterfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if p.SubnetID.IsNull() && p.PortID.IsNull() {
		resp.Diagnostics.AddError("Missing router interface target", "Set subnet_id or port_id.")
		return
	}
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	s, ok := r.config.resolveScope(p.ProjectID, p.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if e := c.SetRouterInterface(ctx, s.ProjectID, s.AvailabilityZone, p.RouterID.ValueString(), p.SubnetID.ValueString(), p.PortID.ValueString(), true); e != nil {
		resp.Diagnostics.AddError("Unable to add Nobus router interface", diagnosticDetail(e))
		return
	}
	p.ID = types.StringValue(strings.Join([]string{p.RouterID.ValueString(), p.SubnetID.ValueString(), p.PortID.ValueString()}, "/"))
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *routerInterfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}
func (r *routerInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *routerInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s routerInterfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.resolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if e := c.SetRouterInterface(ctx, sc.ProjectID, sc.AvailabilityZone, s.RouterID.ValueString(), s.SubnetID.ValueString(), s.PortID.ValueString(), false); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to remove Nobus router interface", diagnosticDetail(e))
	}
}
func (r *routerInterfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid router interface import ID", "Use ROUTER_ID/SUBNET_ID/PORT_ID; leave the unused final component empty.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("router_id"), parts[0])...)
	if parts[1] != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subnet_id"), parts[1])...)
	}
	if parts[2] != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("port_id"), parts[2])...)
	}
}

var _ resource.ResourceWithConfigure = &subnetResource{}
var _ resource.ResourceWithImportState = &subnetResource{}
var _ resource.ResourceWithConfigure = &routerResource{}
var _ resource.ResourceWithImportState = &routerResource{}
var _ resource.ResourceWithConfigure = &routerInterfaceResource{}
var _ resource.ResourceWithImportState = &routerInterfaceResource{}
var _ = fmt.Sprintf
