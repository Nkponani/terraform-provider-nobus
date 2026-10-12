// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package storage

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

// volume type
type volumeTypeResource struct{ config *shared.Config }
type volumeTypeModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	IsPublic         types.Bool   `tfsdk:"is_public"`
	ExtraSpecs       types.Map    `tfsdk:"extra_specs"`
}

func NewVolumeTypeResource() resource.Resource { return &volumeTypeResource{} }
func (r *volumeTypeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_type"
}
func (r *volumeTypeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *volumeTypeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Creates a Nobus block-storage volume type. The live v3 API exposes creation only; removing this Terraform resource removes it from state but cannot delete the remote volume type.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true, PlanModifiers: immutableString()}, "description": schema.StringAttribute{Optional: true, PlanModifiers: immutableString()}, "is_public": schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}}, "extra_specs": schema.MapAttribute{Optional: true, Computed: true, ElementType: types.StringType, PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()}}})}
}
func (r *volumeTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p volumeTypeModel
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
	specs, d := shared.MapToStringMap(ctx, p.ExtraSpecs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, e := c.CreateVolumeType(ctx, client.CreateVolumeTypeRequest{Name: p.Name.ValueString(), Description: shared.OptionalString(p.Description), IsPublic: shared.OptionalBool(p.IsPublic), ExtraSpecs: specs, ProjectID: s.ProjectID, AvailabilityZone: s.AvailabilityZone})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus volume type", shared.DiagnosticDetail(e))
		return
	}
	applyVolumeType(ctx, v, &p, &resp.Diagnostics)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}

// The live v3 API exposes only POST for volume types. State is therefore the
// create response and refresh/delete are intentionally local-only operations.
func (r *volumeTypeResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {}
func (r *volumeTypeResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Volume types cannot be updated", "The Nobus API does not expose a volume type update operation; replace it instead.")
}
func (r *volumeTypeResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}
func applyVolumeType(ctx context.Context, v *client.VolumeType, m *volumeTypeModel, d *diag.Diagnostics) {
	m.ID = types.StringValue(v.ID)
	m.Name = types.StringValue(v.Name)
	m.Description = shared.StringOrNull(v.Description)
	m.IsPublic = shared.BoolOrNull(v.IsPublic)
	if v.ExtraSpecs != nil {
		x, ds := types.MapValueFrom(ctx, types.StringType, v.ExtraSpecs)
		d.Append(ds...)
		m.ExtraSpecs = x
	}
}

var _ resource.ResourceWithConfigure = &volumeTypeResource{}
