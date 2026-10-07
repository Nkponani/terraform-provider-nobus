// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

type portResource struct{ config *providerConfig }
type portModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	NetworkID        types.String `tfsdk:"network_id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	AdminStateUp     types.Bool   `tfsdk:"admin_state_up"`
	DeviceID         types.String `tfsdk:"device_id"`
	DeviceOwner      types.String `tfsdk:"device_owner"`
	SecurityGroups   types.List   `tfsdk:"security_groups"`
	PortSecurity     types.Bool   `tfsdk:"port_security_enabled"`
	DNSName          types.String `tfsdk:"dns_name"`
	Status           types.String `tfsdk:"status"`
	MACAddress       types.String `tfsdk:"mac_address"`
}

func NewPortResource() resource.Resource { return &portResource{} }
func (r *portResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port"
}
func (r *portResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *portResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus network port.", Attributes: mergeAttributes(scopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "network_id": schema.StringAttribute{Required: true, Validators: []validator.String{StringNotEmpty()}, PlanModifiers: replaceStrings()}, "name": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "description": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "admin_state_up": schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}}, "device_id": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "device_owner": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "security_groups": schema.ListAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()}}, "port_security_enabled": schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}}, "dns_name": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "status": schema.StringAttribute{Computed: true}, "mac_address": schema.StringAttribute{Computed: true}})}
}
func (r *portResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p portModel
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
	groups, d := listToStringSlice(ctx, p.SecurityGroups)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, e := c.CreatePort(ctx, s.ProjectID, s.AvailabilityZone, client.PortCreate{NetworkID: p.NetworkID.ValueString(), Name: optionalString(p.Name), Description: optionalString(p.Description), AdminStateUp: optionalBool(p.AdminStateUp), DeviceID: optionalString(p.DeviceID), DeviceOwner: optionalString(p.DeviceOwner), SecurityGroups: groups, PortSecurity: optionalBool(p.PortSecurity), DNSName: optionalString(p.DNSName)})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus port", diagnosticDetail(e))
		return
	}
	applyPort(ctx, v, &p, &resp.Diagnostics)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *portResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s portModel
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
	v, e := c.GetPort(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus port", diagnosticDetail(e))
		return
	}
	applyPort(ctx, v, &s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *portResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Port update is not yet supported", "Replace the port to apply this change.")
}
func (r *portResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s portModel
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
	if e := c.DeletePort(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus port", diagnosticDetail(e))
	}
}
func (r *portResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID(ctx, req.ID, resp)
}
func applyPort(ctx context.Context, v *client.Port, m *portModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.NetworkID = types.StringValue(v.NetworkID)
	m.Name = stringOrNull(v.Name)
	m.Description = stringOrNull(v.Description)
	m.AdminStateUp = boolOrNull(v.AdminStateUp)
	m.DeviceID = stringOrNull(v.DeviceID)
	m.DeviceOwner = stringOrNull(v.DeviceOwner)
	m.PortSecurity = boolOrNull(v.PortSecurity)
	m.DNSName = stringOrNull(v.DNSName)
	m.Status = stringOrNull(v.Status)
	m.MACAddress = stringOrNull(v.MACAddress)
	if v.SecurityGroups != nil {
		x, ds := types.ListValueFrom(ctx, types.StringType, v.SecurityGroups)
		d.Append(ds...)
		m.SecurityGroups = x
	}
}

type floatingIPResource struct{ config *providerConfig }
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
}

func NewFloatingIPResource() resource.Resource { return &floatingIPResource{} }
func (r *floatingIPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_floating_ip"
}
func (r *floatingIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *floatingIPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus floating IP allocation and optional port association.", Attributes: mergeAttributes(scopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "floating_network_id": schema.StringAttribute{Required: true, PlanModifiers: replaceStrings()}, "port_id": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "fixed_ip_address": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "description": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "floating_ip_address": schema.StringAttribute{Computed: true}, "status": schema.StringAttribute{Computed: true}})}
}
func (r *floatingIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p floatingIPModel
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
	v, e := c.CreateFloatingIP(ctx, s.ProjectID, s.AvailabilityZone, client.FloatingIPCreate{FloatingNetworkID: p.FloatingNetworkID.ValueString(), PortID: optionalString(p.PortID), FixedIPAddress: optionalString(p.FixedIPAddress), Description: optionalString(p.Description)})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus floating IP", diagnosticDetail(e))
		return
	}
	applyFloatingIP(v, &p)
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
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.resolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	v, e := c.GetFloatingIP(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus floating IP", diagnosticDetail(e))
		return
	}
	applyFloatingIP(v, &s)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *floatingIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Floating IP update is not yet supported", "Replace the floating IP to change its association.")
}
func (r *floatingIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s floatingIPModel
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
	if e := c.DeleteFloatingIP(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus floating IP", diagnosticDetail(e))
	}
}
func (r *floatingIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID(ctx, req.ID, resp)
}
func applyFloatingIP(v *client.FloatingIP, m *floatingIPModel) {
	m.ID = types.StringValue(v.ID)
	m.FloatingNetworkID = types.StringValue(v.FloatingNetworkID)
	m.PortID = stringOrNull(v.PortID)
	m.FixedIPAddress = stringOrNull(v.FixedIPAddress)
	m.Description = stringOrNull(v.Description)
	m.FloatingIPAddress = stringOrNull(v.FloatingIPAddress)
	m.Status = stringOrNull(v.Status)
}

type securityGroupResource struct{ config *providerConfig }
type securityGroupModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
}

func NewSecurityGroupResource() resource.Resource { return &securityGroupResource{} }
func (r *securityGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}
func (r *securityGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *securityGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus network security group.", Attributes: mergeAttributes(scopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true, Validators: []validator.String{StringNotEmpty()}, PlanModifiers: replaceStrings()}, "description": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}})}
}
func (r *securityGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p securityGroupModel
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
	v, e := c.CreateSecurityGroup(ctx, s.ProjectID, s.AvailabilityZone, client.SecurityGroupCreate{Name: p.Name.ValueString(), Description: optionalString(p.Description)})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus security group", diagnosticDetail(e))
		return
	}
	applySecurityGroup(v, &p)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *securityGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s securityGroupModel
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
	v, e := c.GetSecurityGroup(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus security group", diagnosticDetail(e))
		return
	}
	applySecurityGroup(v, &s)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *securityGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Security group update is not yet supported", "Replace the security group to apply this change.")
}
func (r *securityGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s securityGroupModel
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
	if e := c.DeleteSecurityGroup(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus security group", diagnosticDetail(e))
	}
}
func (r *securityGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importID(ctx, req.ID, resp)
}
func applySecurityGroup(v *client.SecurityGroup, m *securityGroupModel) {
	m.ID = types.StringValue(v.ID)
	m.Name = stringOrNull(v.Name)
	m.Description = stringOrNull(v.Description)
}

type securityGroupRuleResource struct{ config *providerConfig }
type securityGroupRuleModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	SecurityGroupID  types.String `tfsdk:"security_group_id"`
	Direction        types.String `tfsdk:"direction"`
	EtherType        types.String `tfsdk:"ethertype"`
	Protocol         types.String `tfsdk:"protocol"`
	PortRangeMin     types.Int64  `tfsdk:"port_range_min"`
	PortRangeMax     types.Int64  `tfsdk:"port_range_max"`
	RemoteIPPrefix   types.String `tfsdk:"remote_ip_prefix"`
	RemoteGroupID    types.String `tfsdk:"remote_group_id"`
	Description      types.String `tfsdk:"description"`
}

func NewSecurityGroupRuleResource() resource.Resource { return &securityGroupRuleResource{} }
func (r *securityGroupRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_security_group_rule"
}
func (r *securityGroupRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *securityGroupRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	ints := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{MarkdownDescription: "A rule in a Nobus network security group.", Attributes: mergeAttributes(scopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "security_group_id": schema.StringAttribute{Required: true, PlanModifiers: replaceStrings()}, "direction": schema.StringAttribute{Required: true, Validators: []validator.String{StringOneOf("ingress", "egress")}, PlanModifiers: replaceStrings()}, "ethertype": schema.StringAttribute{Optional: true, Computed: true, Validators: []validator.String{StringOneOf("IPv4", "IPv6")}, PlanModifiers: replaceStrings()}, "protocol": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "port_range_min": schema.Int64Attribute{Optional: true, PlanModifiers: ints}, "port_range_max": schema.Int64Attribute{Optional: true, PlanModifiers: ints}, "remote_ip_prefix": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "remote_group_id": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}, "description": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()}})}
}
func (r *securityGroupRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p securityGroupRuleModel
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
	ether := "IPv4"
	if !p.EtherType.IsNull() && !p.EtherType.IsUnknown() {
		ether = p.EtherType.ValueString()
	}
	v, e := c.CreateSecurityGroupRule(ctx, s.ProjectID, s.AvailabilityZone, client.SecurityGroupRuleCreate{SecurityGroupID: p.SecurityGroupID.ValueString(), Direction: p.Direction.ValueString(), EtherType: ether, Protocol: optionalString(p.Protocol), PortRangeMin: optionalInt64(p.PortRangeMin), PortRangeMax: optionalInt64(p.PortRangeMax), RemoteIPPrefix: optionalString(p.RemoteIPPrefix), RemoteGroupID: optionalString(p.RemoteGroupID), Description: optionalString(p.Description)})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus security group rule", diagnosticDetail(e))
		return
	}
	applySecurityGroupRule(v, &p)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *securityGroupRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s securityGroupRuleModel
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
	v, e := c.GetSecurityGroupRule(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus security group rule", diagnosticDetail(e))
		return
	}
	applySecurityGroupRule(v, &s)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *securityGroupRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *securityGroupRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s securityGroupRuleModel
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
	if e := c.DeleteSecurityGroupRule(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus security group rule", diagnosticDetail(e))
	}
}
func (r *securityGroupRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
func applySecurityGroupRule(v *client.SecurityGroupRule, m *securityGroupRuleModel) {
	m.ID = types.StringValue(v.ID)
	m.SecurityGroupID = types.StringValue(v.SecurityGroupID)
	m.Direction = types.StringValue(v.Direction)
	m.EtherType = types.StringValue(v.EtherType)
	m.Protocol = stringOrNull(v.Protocol)
	m.PortRangeMin = int64OrNull(v.PortRangeMin)
	m.PortRangeMax = int64OrNull(v.PortRangeMax)
	m.RemoteIPPrefix = stringOrNull(v.RemoteIPPrefix)
	m.RemoteGroupID = stringOrNull(v.RemoteGroupID)
	m.Description = stringOrNull(v.Description)
}

var _ resource.ResourceWithConfigure = &portResource{}
var _ resource.ResourceWithImportState = &portResource{}
var _ resource.ResourceWithConfigure = &floatingIPResource{}
var _ resource.ResourceWithImportState = &floatingIPResource{}
var _ resource.ResourceWithConfigure = &securityGroupResource{}
var _ resource.ResourceWithImportState = &securityGroupResource{}
var _ resource.ResourceWithConfigure = &securityGroupRuleResource{}
var _ resource.ResourceWithImportState = &securityGroupRuleResource{}
