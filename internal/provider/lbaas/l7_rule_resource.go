// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lbaas

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type l7RuleResource struct{ config *shared.Config }
type l7RuleResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	AvailabilityZone   types.String `tfsdk:"availability_zone"`
	L7PolicyID         types.String `tfsdk:"l7_policy_id"`
	Type               types.String `tfsdk:"type"`
	CompareType        types.String `tfsdk:"compare_type"`
	Value              types.String `tfsdk:"value"`
	Key                types.String `tfsdk:"key"`
	Invert             types.Bool   `tfsdk:"invert"`
	AdminStateUp       types.Bool   `tfsdk:"admin_state_up"`
	Tags               types.List   `tfsdk:"tags"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`
	OperatingStatus    types.String `tfsdk:"operating_status"`
}
type l7RuleAPI struct {
	apiResource
	L7Policy    string  `json:"l7policy"`
	Type        string  `json:"type"`
	CompareType string  `json:"compare_type"`
	Value       string  `json:"value"`
	Key         *string `json:"key"`
	Invert      bool    `json:"invert"`
}

func NewL7RuleResource() resource.Resource { return &l7RuleResource{} }
func (r *l7RuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_l7_rule"
}
func (r *l7RuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	r.config = config
}
func (r *l7RuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "An L7 rule within a Nobus LBaaS L7 policy.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "l7_policy_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "type": schema.StringAttribute{Required: true}, "compare_type": schema.StringAttribute{Required: true}, "value": schema.StringAttribute{Required: true}, "key": schema.StringAttribute{Optional: true}, "invert": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)}, "admin_state_up": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)}, "tags": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType}, "provisioning_status": schema.StringAttribute{Computed: true}, "operating_status": schema.StringAttribute{Computed: true},
	})}
}
func (r *l7RuleResource) body(ctx context.Context, p *l7RuleResourceModel, diagnostics *diag.Diagnostics, includePolicy bool) map[string]any {
	tags, ds := stringsFromList(ctx, p.Tags)
	diagnostics.Append(ds...)
	body := map[string]any{"type": p.Type.ValueString(), "compare_type": p.CompareType.ValueString(), "value": p.Value.ValueString(), "key": optionalString(p.Key), "invert": p.Invert.ValueBool(), "admin_state_up": p.AdminStateUp.ValueBool(), "tags": tags}
	if includePolicy {
		body["l7policy_id"] = p.L7PolicyID.ValueString()
	}
	return body
}
func (r *l7RuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan l7RuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	result, err := api.CreateLBaaSResource(ctx, client.LBaaSL7Rules, scope.ProjectID, scope.AvailabilityZone, r.body(ctx, &plan, &resp.Diagnostics, true))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create LBaaS L7 rule", shared.DiagnosticDetail(err))
		return
	}
	var value l7RuleAPI
	if err := decodeResource(result.Resource, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS L7 rule", err.Error())
		return
	}
	plan.ProjectID, plan.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	applyL7Rule(ctx, &value, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *l7RuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state l7RuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	parents := map[string]string{"l7policy_id": state.L7PolicyID.ValueString()}
	raw, err := api.GetLBaaSResource(ctx, client.LBaaSL7Rules, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), parents)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read LBaaS L7 rule", shared.DiagnosticDetail(err))
		return
	}
	var value l7RuleAPI
	if err := decodeResource(raw, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS L7 rule", err.Error())
		return
	}
	applyL7Rule(ctx, &value, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *l7RuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state l7RuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	parents := map[string]string{"l7policy_id": plan.L7PolicyID.ValueString()}
	raw, err := api.UpdateLBaaSResource(ctx, client.LBaaSL7Rules, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), parents, r.body(ctx, &plan, &resp.Diagnostics, false))
	if err != nil {
		resp.Diagnostics.AddError("Unable to update LBaaS L7 rule", shared.DiagnosticDetail(err))
		return
	}
	var value l7RuleAPI
	if decodeResource(raw, &value) == nil {
		applyL7Rule(ctx, &value, &plan, &resp.Diagnostics)
	} else {
		plan.ID = state.ID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *l7RuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state l7RuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.DeleteLBaaSResource(ctx, client.LBaaSL7Rules, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), map[string]string{"l7policy_id": state.L7PolicyID.ValueString()}); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete LBaaS L7 rule", shared.DiagnosticDetail(err))
	}
}
func (r *l7RuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid LBaaS L7 rule import ID", "Use L7_POLICY_ID/L7_RULE_ID.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("l7_policy_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
func applyL7Rule(ctx context.Context, v *l7RuleAPI, s *l7RuleResourceModel, diagnostics *diag.Diagnostics) {
	s.ID = types.StringValue(v.ID)
	s.L7PolicyID = plainStringValue(v.L7Policy)
	s.Type = plainStringValue(v.Type)
	s.CompareType = plainStringValue(v.CompareType)
	s.Value = plainStringValue(v.Value)
	s.Key = stringPointerValue(v.Key)
	s.Invert = types.BoolValue(v.Invert)
	s.AdminStateUp = types.BoolValue(v.AdminStateUp)
	s.ProvisioningStatus = plainStringValue(v.ProvisioningStatus)
	s.OperatingStatus = plainStringValue(v.OperatingStatus)
	tags, ds := listFromStrings(ctx, v.Tags)
	diagnostics.Append(ds...)
	s.Tags = tags
}

var _ resource.ResourceWithConfigure = &l7RuleResource{}
var _ resource.ResourceWithImportState = &l7RuleResource{}
