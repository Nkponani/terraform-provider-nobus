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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type poolResource struct{ config *shared.Config }
type poolResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	ProjectID                    types.String `tfsdk:"project_id"`
	AvailabilityZone             types.String `tfsdk:"availability_zone"`
	Name                         types.String `tfsdk:"name"`
	Description                  types.String `tfsdk:"description"`
	LoadBalancerID               types.String `tfsdk:"load_balancer_id"`
	ListenerID                   types.String `tfsdk:"listener_id"`
	Protocol                     types.String `tfsdk:"protocol"`
	LBAlgorithm                  types.String `tfsdk:"lb_algorithm"`
	SessionPersistenceType       types.String `tfsdk:"session_persistence_type"`
	SessionPersistenceCookieName types.String `tfsdk:"session_persistence_cookie_name"`
	AdminStateUp                 types.Bool   `tfsdk:"admin_state_up"`
	Tags                         types.List   `tfsdk:"tags"`
	ProvisioningStatus           types.String `tfsdk:"provisioning_status"`
	OperatingStatus              types.String `tfsdk:"operating_status"`
	TLSEnabled                   types.Bool   `tfsdk:"tls_enabled"`
}
type poolAPI struct {
	apiResource
	LoadBalancer       *string `json:"loadbalancer"`
	Protocol           string  `json:"protocol"`
	LBAlgorithm        string  `json:"lb_algorithm"`
	SessionPersistence *struct {
		Type       string  `json:"type"`
		CookieName *string `json:"cookie_name"`
	} `json:"session_persistence"`
	TLSEnabled bool `json:"tls_enabled"`
}

func NewPoolResource() resource.Resource { return &poolResource{} }
func (r *poolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_pool"
}
func (r *poolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	r.config = config
}
func (r *poolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A backend pool for Nobus LBaaS.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Required: true}, "description": schema.StringAttribute{Optional: true},
		"load_balancer_id": schema.StringAttribute{Optional: true, PlanModifiers: shared.ReplaceStrings()}, "listener_id": schema.StringAttribute{Optional: true, PlanModifiers: shared.ReplaceStrings()},
		"protocol": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "lb_algorithm": schema.StringAttribute{Optional: true, Computed: true},
		"session_persistence_type": schema.StringAttribute{Optional: true}, "session_persistence_cookie_name": schema.StringAttribute{Optional: true},
		"admin_state_up": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)}, "tags": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
		"provisioning_status": schema.StringAttribute{Computed: true}, "operating_status": schema.StringAttribute{Computed: true}, "tls_enabled": schema.BoolAttribute{Computed: true},
	})}
}
func (r *poolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan poolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	tags, ds := stringsFromList(ctx, plan.Tags)
	resp.Diagnostics.Append(ds...)
	algorithm := plan.LBAlgorithm.ValueString()
	if algorithm == "" {
		algorithm = "ROUND_ROBIN"
	}
	body := map[string]any{"name": plan.Name.ValueString(), "description": optionalString(plan.Description), "loadbalancer_id": optionalString(plan.LoadBalancerID), "listener_id": optionalString(plan.ListenerID), "protocol": plan.Protocol.ValueString(), "lb_algorithm": algorithm, "session_persistence": persistenceValue(plan.SessionPersistenceType, plan.SessionPersistenceCookieName), "admin_state_up": plan.AdminStateUp.ValueBool(), "tags": tags}
	result, err := api.CreateLBaaSResource(ctx, client.LBaaSPools, scope.ProjectID, scope.AvailabilityZone, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create LBaaS pool", shared.DiagnosticDetail(err))
		return
	}
	var value poolAPI
	if err := decodeResource(result.Resource, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS pool", err.Error())
		return
	}
	plan.ProjectID, plan.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	applyPool(ctx, &value, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *poolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state poolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	raw, err := api.GetLBaaSResource(ctx, client.LBaaSPools, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read LBaaS pool", shared.DiagnosticDetail(err))
		return
	}
	var value poolAPI
	if err := decodeResource(raw, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS pool", err.Error())
		return
	}
	applyPool(ctx, &value, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *poolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state poolResourceModel
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
	tags, ds := stringsFromList(ctx, plan.Tags)
	resp.Diagnostics.Append(ds...)
	raw, err := api.UpdateLBaaSResource(ctx, client.LBaaSPools, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil, map[string]any{"name": plan.Name.ValueString(), "description": optionalString(plan.Description), "lb_algorithm": plan.LBAlgorithm.ValueString(), "admin_state_up": plan.AdminStateUp.ValueBool(), "tags": tags})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update LBaaS pool", shared.DiagnosticDetail(err))
		return
	}
	if !plan.SessionPersistenceType.Equal(state.SessionPersistenceType) || !plan.SessionPersistenceCookieName.Equal(state.SessionPersistenceCookieName) {
		raw, err = api.UpdateLBaaSSessionPersistence(ctx, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), persistenceValue(plan.SessionPersistenceType, plan.SessionPersistenceCookieName))
		if err != nil {
			resp.Diagnostics.AddError("Unable to update pool session persistence", shared.DiagnosticDetail(err))
			return
		}
	}
	var value poolAPI
	if decodeResource(raw, &value) == nil {
		applyPool(ctx, &value, &plan, &resp.Diagnostics)
	} else {
		plan.ID = state.ID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *poolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state poolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.DeleteLBaaSResource(ctx, client.LBaaSPools, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete LBaaS pool", shared.DiagnosticDetail(err))
	}
}
func (r *poolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
func applyPool(ctx context.Context, value *poolAPI, state *poolResourceModel, diagnostics *diag.Diagnostics) {
	state.ID = types.StringValue(value.ID)
	state.Name = stringPointerValue(value.Name)
	state.Description = stringPointerValue(value.Description)
	if value.LoadBalancer != nil {
		state.LoadBalancerID = stringPointerValue(value.LoadBalancer)
	}
	state.Protocol = plainStringValue(value.Protocol)
	state.LBAlgorithm = plainStringValue(value.LBAlgorithm)
	state.AdminStateUp = types.BoolValue(value.AdminStateUp)
	state.ProvisioningStatus = plainStringValue(value.ProvisioningStatus)
	state.OperatingStatus = plainStringValue(value.OperatingStatus)
	state.TLSEnabled = types.BoolValue(value.TLSEnabled)
	if value.SessionPersistence == nil {
		state.SessionPersistenceType, state.SessionPersistenceCookieName = types.StringNull(), types.StringNull()
	} else {
		state.SessionPersistenceType, state.SessionPersistenceCookieName = plainStringValue(value.SessionPersistence.Type), stringPointerValue(value.SessionPersistence.CookieName)
	}
	tags, ds := listFromStrings(ctx, value.Tags)
	diagnostics.Append(ds...)
	state.Tags = tags
}

var _ resource.ResourceWithConfigure = &poolResource{}
var _ resource.ResourceWithImportState = &poolResource{}
