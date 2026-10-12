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

type qosPolicyResource struct{ config *shared.Config }
type qosPolicyModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Shared           types.Bool   `tfsdk:"shared"`
	Tags             types.Set    `tfsdk:"tags"`
}

func NewQosPolicyResource() resource.Resource { return &qosPolicyResource{} }
func (r *qosPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_qos_policy"
}
func (r *qosPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *qosPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus network QoS policy.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true}, "description": schema.StringAttribute{Optional: true}, "shared": schema.BoolAttribute{Optional: true, Computed: true}, "tags": schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType}})}
}
func (r *qosPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p qosPolicyModel
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
	v, e := c.CreateQosPolicy(ctx, s.ProjectID, s.AvailabilityZone, client.QosPolicyCreate{Name: p.Name.ValueString(), Description: shared.OptionalString(p.Description), Shared: shared.OptionalBool(p.Shared), Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus QoS policy", shared.DiagnosticDetail(e))
		return
	}
	applyQosPolicy(ctx, v, &p, &resp.Diagnostics)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *qosPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s qosPolicyModel
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
	v, e := c.GetQosPolicy(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString())
	if e != nil {
		if client.IsNotFound(e) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to read Nobus QoS policy", shared.DiagnosticDetail(e))
		return
	}
	applyQosPolicy(ctx, v, &s, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *qosPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p qosPolicyModel
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
	v, e := c.UpdateQosPolicy(ctx, s.ProjectID, s.AvailabilityZone, p.ID.ValueString(), client.QosPolicyUpdate{Name: shared.OptionalString(p.Name), Description: shared.OptionalString(p.Description), Shared: shared.OptionalBool(p.Shared), Tags: tags})
	if e != nil {
		resp.Diagnostics.AddError("Unable to update Nobus QoS policy", shared.DiagnosticDetail(e))
		return
	}
	applyQosPolicy(ctx, v, &p, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *qosPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s qosPolicyModel
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
	if e := c.DeleteQosPolicy(ctx, sc.ProjectID, sc.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus QoS policy", shared.DiagnosticDetail(e))
	}
}
func (r *qosPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	shared.ImportID(ctx, req.ID, resp)
}
func applyQosPolicy(ctx context.Context, v *client.QosPolicy, m *qosPolicyModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.Name = shared.StringOrNull(v.Name)
	m.Description = shared.StringOrNull(v.Description)
	m.Shared = shared.BoolOrNull(v.Shared)
	if v.Tags != nil {
		x, ds := types.SetValueFrom(ctx, types.StringType, v.Tags)
		d.Append(ds...)
		m.Tags = x
	}
}

var _ resource.ResourceWithConfigure = &qosPolicyResource{}
var _ resource.ResourceWithImportState = &qosPolicyResource{}
