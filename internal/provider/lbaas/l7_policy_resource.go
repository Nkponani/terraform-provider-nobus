// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lbaas

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type l7PolicyResource struct{ config *shared.Config }
type l7PolicyResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	AvailabilityZone   types.String `tfsdk:"availability_zone"`
	ListenerID         types.String `tfsdk:"listener_id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Action             types.String `tfsdk:"action"`
	RedirectPoolID     types.String `tfsdk:"redirect_pool_id"`
	RedirectURL        types.String `tfsdk:"redirect_url"`
	RedirectPrefix     types.String `tfsdk:"redirect_prefix"`
	RedirectHTTPCode   types.Int64  `tfsdk:"redirect_http_code"`
	Position           types.Int64  `tfsdk:"position"`
	AdminStateUp       types.Bool   `tfsdk:"admin_state_up"`
	Tags               types.List   `tfsdk:"tags"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`
	OperatingStatus    types.String `tfsdk:"operating_status"`
}
type l7PolicyAPI struct {
	apiResource
	Listener         string  `json:"listener"`
	Action           string  `json:"action"`
	RedirectPool     *string `json:"redirect_pool"`
	RedirectURL      *string `json:"redirect_url"`
	RedirectPrefix   *string `json:"redirect_prefix"`
	RedirectHTTPCode int64   `json:"redirect_http_code"`
	Position         *int64  `json:"position"`
}

func NewL7PolicyResource() resource.Resource { return &l7PolicyResource{} }
func (r *l7PolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_l7_policy"
}
func (r *l7PolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	r.config = config
}
func (r *l7PolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "An L7 policy attached to a Nobus LBaaS listener.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "listener_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "name": schema.StringAttribute{Optional: true}, "description": schema.StringAttribute{Optional: true}, "action": schema.StringAttribute{Required: true},
		"redirect_pool_id": schema.StringAttribute{Optional: true}, "redirect_url": schema.StringAttribute{Optional: true}, "redirect_prefix": schema.StringAttribute{Optional: true}, "redirect_http_code": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(302)}, "position": schema.Int64Attribute{Optional: true},
		"admin_state_up": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)}, "tags": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType}, "provisioning_status": schema.StringAttribute{Computed: true}, "operating_status": schema.StringAttribute{Computed: true},
	})}
}
func (r *l7PolicyResource) body(ctx context.Context, p *l7PolicyResourceModel, diagnostics *diag.Diagnostics, includeListener bool) map[string]any {
	tags, ds := stringsFromList(ctx, p.Tags)
	diagnostics.Append(ds...)
	body := map[string]any{"name": optionalString(p.Name), "description": optionalString(p.Description), "action": p.Action.ValueString(), "redirect_pool_id": optionalString(p.RedirectPoolID), "redirect_url": optionalString(p.RedirectURL), "redirect_prefix": optionalString(p.RedirectPrefix), "redirect_http_code": p.RedirectHTTPCode.ValueInt64(), "position": nullableInt(p.Position), "admin_state_up": p.AdminStateUp.ValueBool(), "tags": tags}
	if includeListener {
		body["listener_id"] = p.ListenerID.ValueString()
	}
	return body
}
func (r *l7PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan l7PolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	result, err := api.CreateLBaaSResource(ctx, client.LBaaSL7Policies, scope.ProjectID, scope.AvailabilityZone, r.body(ctx, &plan, &resp.Diagnostics, true))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create LBaaS L7 policy", shared.DiagnosticDetail(err))
		return
	}
	var value l7PolicyAPI
	if err := decodeResource(result.Resource, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS L7 policy", err.Error())
		return
	}
	plan.ProjectID, plan.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	applyL7Policy(ctx, &value, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *l7PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state l7PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	raw, err := api.GetLBaaSResource(ctx, client.LBaaSL7Policies, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read LBaaS L7 policy", shared.DiagnosticDetail(err))
		return
	}
	var value l7PolicyAPI
	if err := decodeResource(raw, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS L7 policy", err.Error())
		return
	}
	applyL7Policy(ctx, &value, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *l7PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state l7PolicyResourceModel
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
	raw, err := api.UpdateLBaaSResource(ctx, client.LBaaSL7Policies, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil, r.body(ctx, &plan, &resp.Diagnostics, false))
	if err != nil {
		resp.Diagnostics.AddError("Unable to update LBaaS L7 policy", shared.DiagnosticDetail(err))
		return
	}
	var value l7PolicyAPI
	if decodeResource(raw, &value) == nil {
		applyL7Policy(ctx, &value, &plan, &resp.Diagnostics)
	} else {
		plan.ID = state.ID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *l7PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state l7PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.DeleteLBaaSResource(ctx, client.LBaaSL7Policies, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete LBaaS L7 policy", shared.DiagnosticDetail(err))
	}
}
func (r *l7PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
func applyL7Policy(ctx context.Context, v *l7PolicyAPI, s *l7PolicyResourceModel, diagnostics *diag.Diagnostics) {
	s.ID = types.StringValue(v.ID)
	s.ListenerID = plainStringValue(v.Listener)
	s.Name = stringPointerValue(v.Name)
	s.Description = stringPointerValue(v.Description)
	s.Action = plainStringValue(v.Action)
	s.RedirectPoolID = stringPointerValue(v.RedirectPool)
	s.RedirectURL = stringPointerValue(v.RedirectURL)
	s.RedirectPrefix = stringPointerValue(v.RedirectPrefix)
	s.RedirectHTTPCode = types.Int64Value(v.RedirectHTTPCode)
	if v.Position == nil {
		s.Position = types.Int64Null()
	} else {
		s.Position = types.Int64Value(*v.Position)
	}
	s.AdminStateUp = types.BoolValue(v.AdminStateUp)
	s.ProvisioningStatus = plainStringValue(v.ProvisioningStatus)
	s.OperatingStatus = plainStringValue(v.OperatingStatus)
	tags, ds := listFromStrings(ctx, v.Tags)
	diagnostics.Append(ds...)
	s.Tags = tags
}

var _ resource.ResourceWithConfigure = &l7PolicyResource{}
var _ resource.ResourceWithImportState = &l7PolicyResource{}
