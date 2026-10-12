// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package networking

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type securityGroupRuleResource struct{ config *shared.Config }
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
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *securityGroupRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	ints := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{MarkdownDescription: "A rule in a Nobus network security group.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "security_group_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "direction": schema.StringAttribute{Required: true, Validators: []validator.String{shared.StringOneOf("ingress", "egress")}, PlanModifiers: shared.ReplaceStrings()}, "ethertype": schema.StringAttribute{Optional: true, Computed: true, Validators: []validator.String{shared.StringOneOf("IPv4", "IPv6")}, PlanModifiers: shared.ReplaceStrings()}, "protocol": schema.StringAttribute{Optional: true, PlanModifiers: shared.ReplaceStrings()}, "port_range_min": schema.Int64Attribute{Optional: true, PlanModifiers: ints}, "port_range_max": schema.Int64Attribute{Optional: true, PlanModifiers: ints}, "remote_ip_prefix": schema.StringAttribute{Optional: true, PlanModifiers: shared.ReplaceStrings()}, "remote_group_id": schema.StringAttribute{Optional: true, PlanModifiers: shared.ReplaceStrings()}, "description": schema.StringAttribute{Optional: true, PlanModifiers: shared.ReplaceStrings()}})}
}
func (r *securityGroupRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p securityGroupRuleModel
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
	ether := "IPv4"
	if !p.EtherType.IsNull() && !p.EtherType.IsUnknown() {
		ether = p.EtherType.ValueString()
	}
	v, e := c.CreateSecurityGroupRule(ctx, s.ProjectID, s.AvailabilityZone, client.SecurityGroupRuleCreate{SecurityGroupID: p.SecurityGroupID.ValueString(), Direction: p.Direction.ValueString(), EtherType: ether, Protocol: shared.OptionalString(p.Protocol), PortRangeMin: shared.OptionalInt64(p.PortRangeMin), PortRangeMax: shared.OptionalInt64(p.PortRangeMax), RemoteIPPrefix: shared.OptionalString(p.RemoteIPPrefix), RemoteGroupID: shared.OptionalString(p.RemoteGroupID), Description: shared.OptionalString(p.Description)})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus security group rule", shared.DiagnosticDetail(e))
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
	c := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.ResolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	v, e := c.GetSecurityGroupRule(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus security group rule", shared.DiagnosticDetail(e))
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
	c := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	sc, ok := r.config.ResolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if e := c.DeleteSecurityGroupRule(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus security group rule", shared.DiagnosticDetail(e))
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
	m.Protocol = shared.StringOrNull(v.Protocol)
	m.PortRangeMin = shared.Int64OrNull(v.PortRangeMin)
	m.PortRangeMax = shared.Int64OrNull(v.PortRangeMax)
	m.RemoteIPPrefix = shared.StringOrNull(v.RemoteIPPrefix)
	m.RemoteGroupID = shared.StringOrNull(v.RemoteGroupID)
	m.Description = shared.StringOrNull(v.Description)
}

var _ resource.ResourceWithConfigure = &securityGroupRuleResource{}
var _ resource.ResourceWithImportState = &securityGroupRuleResource{}
