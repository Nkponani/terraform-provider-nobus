// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networking

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type portResource struct{ config *shared.Config }
type portModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectID           types.String `tfsdk:"project_id"`
	AvailabilityZone    types.String `tfsdk:"availability_zone"`
	NetworkID           types.String `tfsdk:"network_id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	AdminStateUp        types.Bool   `tfsdk:"admin_state_up"`
	DeviceID            types.String `tfsdk:"device_id"`
	DeviceOwner         types.String `tfsdk:"device_owner"`
	SecurityGroups      types.List   `tfsdk:"security_groups"`
	PortSecurity        types.Bool   `tfsdk:"port_security_enabled"`
	DNSName             types.String `tfsdk:"dns_name"`
	Status              types.String `tfsdk:"status"`
	MACAddress          types.String `tfsdk:"mac_address"`
	Tags                types.Set    `tfsdk:"tags"`
	FixedIPs            types.List   `tfsdk:"fixed_ips"`
	AllowedAddressPairs types.List   `tfsdk:"allowed_address_pairs"`
	BindingHostID       types.String `tfsdk:"binding_host_id"`
	BindingVNICType     types.String `tfsdk:"binding_vnic_type"`
	BindingProfile      types.Map    `tfsdk:"binding_profile"`
	ExtraDHCPOptions    types.List   `tfsdk:"extra_dhcp_options"`
}

func NewPortResource() resource.Resource { return &portResource{} }
func (r *portResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port"
}
func (r *portResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *portResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus network port.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "network_id": schema.StringAttribute{Required: true, Validators: []validator.String{shared.StringNotEmpty()}, PlanModifiers: shared.ReplaceStrings()}, "name": schema.StringAttribute{Optional: true}, "description": schema.StringAttribute{Optional: true}, "admin_state_up": schema.BoolAttribute{Optional: true, Computed: true}, "device_id": schema.StringAttribute{Optional: true}, "device_owner": schema.StringAttribute{Optional: true}, "security_groups": schema.ListAttribute{Optional: true, ElementType: types.StringType}, "fixed_ips": schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"subnet_id": schema.StringAttribute{Required: true}, "ip_address": schema.StringAttribute{Optional: true, Computed: true}}}}, "allowed_address_pairs": schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"ip_address": schema.StringAttribute{Required: true}, "mac_address": schema.StringAttribute{Optional: true, Computed: true}}}}, "binding_host_id": schema.StringAttribute{Optional: true, Computed: true}, "binding_vnic_type": schema.StringAttribute{Optional: true, Computed: true}, "binding_profile": schema.MapAttribute{Optional: true, Computed: true, ElementType: types.StringType}, "extra_dhcp_options": schema.ListNestedAttribute{Optional: true, Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{"name": schema.StringAttribute{Required: true}, "value": schema.StringAttribute{Required: true}}}}, "port_security_enabled": schema.BoolAttribute{Optional: true, Computed: true}, "dns_name": schema.StringAttribute{Optional: true}, "tags": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType}, "status": schema.StringAttribute{Computed: true}, "mac_address": schema.StringAttribute{Computed: true}})}
}
func (r *portResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p portModel
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
	groups, d := shared.ListToStringSlice(ctx, p.SecurityGroups)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	tags, td := shared.SetToStringSlice(ctx, p.Tags)
	resp.Diagnostics.Append(td...)
	fixed, fd := fixedIPsFromList(ctx, p.FixedIPs)
	pairs, pd := portAddressPairs(ctx, p.AllowedAddressPairs)
	extra, ed := extraDHCPOptionsFromList(ctx, p.ExtraDHCPOptions)
	profile, bd := shared.MapToStringMap(ctx, p.BindingProfile)
	resp.Diagnostics.Append(fd...)
	resp.Diagnostics.Append(pd...)
	resp.Diagnostics.Append(ed...)
	resp.Diagnostics.Append(bd...)
	profileAny := map[string]any{}
	for k, v := range profile {
		profileAny[k] = v
	}
	v, e := c.CreatePort(ctx, s.ProjectID, s.AvailabilityZone, client.PortCreate{NetworkID: p.NetworkID.ValueString(), Name: shared.OptionalString(p.Name), Description: shared.OptionalString(p.Description), AdminStateUp: shared.OptionalBool(p.AdminStateUp), DeviceID: shared.OptionalString(p.DeviceID), DeviceOwner: shared.OptionalString(p.DeviceOwner), SecurityGroups: groups, FixedIPs: fixed, AllowedAddressPairs: pairs, BindingHostID: shared.OptionalString(p.BindingHostID), BindingVNICType: shared.OptionalString(p.BindingVNICType), BindingProfile: profileAny, ExtraDHCPOptions: extra, PortSecurity: shared.OptionalBool(p.PortSecurity), DNSName: shared.OptionalString(p.DNSName), Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus port", shared.DiagnosticDetail(e))
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
	c := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.ResolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	v, e := c.GetPort(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus port", shared.DiagnosticDetail(e))
		return
	}
	applyPort(ctx, v, &s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *portResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p portModel
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
	groups, d1 := shared.ListToStringSlice(ctx, p.SecurityGroups)
	tags, d2 := shared.SetToStringSlice(ctx, p.Tags)
	fixed, d3 := fixedIPsFromList(ctx, p.FixedIPs)
	pairs, d4 := portAddressPairs(ctx, p.AllowedAddressPairs)
	extra, d5 := extraDHCPOptionsFromList(ctx, p.ExtraDHCPOptions)
	profile, d6 := shared.MapToStringMap(ctx, p.BindingProfile)
	profileAny := map[string]any{}
	for k, v := range profile {
		profileAny[k] = v
	}
	resp.Diagnostics.Append(d1...)
	resp.Diagnostics.Append(d2...)
	resp.Diagnostics.Append(d3...)
	resp.Diagnostics.Append(d4...)
	resp.Diagnostics.Append(d5...)
	resp.Diagnostics.Append(d6...)
	v, e := c.UpdatePort(ctx, s.ProjectID, s.AvailabilityZone, p.ID.ValueString(), client.PortUpdate{Name: shared.OptionalString(p.Name), Description: shared.OptionalString(p.Description), AdminStateUp: shared.OptionalBool(p.AdminStateUp), DeviceID: shared.OptionalString(p.DeviceID), DeviceOwner: shared.OptionalString(p.DeviceOwner), SecurityGroups: groups, FixedIPs: fixed, AllowedAddressPairs: pairs, BindingHostID: shared.OptionalString(p.BindingHostID), BindingVNICType: shared.OptionalString(p.BindingVNICType), BindingProfile: profileAny, ExtraDHCPOptions: extra, PortSecurity: shared.OptionalBool(p.PortSecurity), DNSName: shared.OptionalString(p.DNSName), Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to update Nobus port", shared.DiagnosticDetail(e))
		return
	}
	applyPort(ctx, v, &p, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *portResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s portModel
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
	if e := c.DeletePort(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus port", shared.DiagnosticDetail(e))
	}
}
func (r *portResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shared.ImportID(ctx, req.ID, resp)
}
func applyPort(ctx context.Context, v *client.Port, m *portModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.NetworkID = types.StringValue(v.NetworkID)
	m.Name = shared.StringOrNull(v.Name)
	m.Description = shared.StringOrNull(v.Description)
	m.AdminStateUp = shared.BoolOrNull(v.AdminStateUp)
	m.DeviceID = shared.StringOrNull(v.DeviceID)
	m.DeviceOwner = shared.StringOrNull(v.DeviceOwner)
	m.PortSecurity = shared.BoolOrNull(v.PortSecurity)
	m.DNSName = shared.StringOrNull(v.DNSName)
	m.Status = shared.StringOrNull(v.Status)
	m.MACAddress = shared.StringOrNull(v.MACAddress)
	if v.SecurityGroups != nil {
		x, ds := types.ListValueFrom(ctx, types.StringType, v.SecurityGroups)
		d.Append(ds...)
		m.SecurityGroups = x
	}
	if v.Tags != nil {
		x, ds := types.SetValueFrom(ctx, types.StringType, v.Tags)
		d.Append(ds...)
		m.Tags = x
	}
	if v.FixedIPs != nil {
		x, ds := fixedIPsValue(ctx, v.FixedIPs)
		d.Append(ds...)
		m.FixedIPs = x
	}
	if v.AllowedAddressPairs != nil {
		x, ds := portAddressPairsValue(ctx, v.AllowedAddressPairs)
		d.Append(ds...)
		m.AllowedAddressPairs = x
	}
	if v.ExtraDHCPOptions != nil {
		x, ds := extraDHCPOptionsValue(ctx, v.ExtraDHCPOptions)
		d.Append(ds...)
		m.ExtraDHCPOptions = x
	}
	m.BindingHostID = shared.StringOrNull(v.BindingHostID)
	m.BindingVNICType = shared.StringOrNull(v.BindingVNICType)
	if v.BindingProfile != nil {
		profile := map[string]string{}
		for k, x := range v.BindingProfile {
			profile[k] = fmt.Sprint(x)
		}
		x, ds := types.MapValueFrom(ctx, types.StringType, profile)
		d.Append(ds...)
		m.BindingProfile = x
	}
}

var _ resource.ResourceWithConfigure = &portResource{}
var _ resource.ResourceWithImportState = &portResource{}
