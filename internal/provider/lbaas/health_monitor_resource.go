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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type healthMonitorResource struct{ config *shared.Config }
type healthMonitorResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	AvailabilityZone   types.String `tfsdk:"availability_zone"`
	PoolID             types.String `tfsdk:"pool_id"`
	Name               types.String `tfsdk:"name"`
	Type               types.String `tfsdk:"type"`
	Delay              types.Int64  `tfsdk:"delay"`
	Timeout            types.Int64  `tfsdk:"timeout"`
	MaxRetries         types.Int64  `tfsdk:"max_retries"`
	MaxRetriesDown     types.Int64  `tfsdk:"max_retries_down"`
	URLPath            types.String `tfsdk:"url_path"`
	ExpectedCodes      types.String `tfsdk:"expected_codes"`
	HTTPMethod         types.String `tfsdk:"http_method"`
	AdminStateUp       types.Bool   `tfsdk:"admin_state_up"`
	Tags               types.List   `tfsdk:"tags"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`
	OperatingStatus    types.String `tfsdk:"operating_status"`
	DomainName         types.String `tfsdk:"domain_name"`
}
type healthMonitorAPI struct {
	apiResource
	Pool           string  `json:"pool"`
	Type           string  `json:"type"`
	Delay          int64   `json:"delay"`
	Timeout        int64   `json:"timeout"`
	MaxRetries     int64   `json:"max_retries"`
	MaxRetriesDown int64   `json:"max_retries_down"`
	URLPath        string  `json:"url_path"`
	ExpectedCodes  string  `json:"expected_codes"`
	HTTPMethod     string  `json:"http_method"`
	DomainName     *string `json:"domain_name"`
}

func NewHealthMonitorResource() resource.Resource { return &healthMonitorResource{} }
func (r *healthMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_health_monitor"
}
func (r *healthMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	r.config = config
}
func (r *healthMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A health monitor for a Nobus LBaaS pool.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "pool_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "name": schema.StringAttribute{Optional: true}, "type": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()},
		"delay": schema.Int64Attribute{Required: true}, "timeout": schema.Int64Attribute{Required: true}, "max_retries": schema.Int64Attribute{Required: true}, "max_retries_down": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(3)},
		"url_path": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("/")}, "expected_codes": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("200")}, "http_method": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("GET")},
		"admin_state_up": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)}, "tags": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType}, "provisioning_status": schema.StringAttribute{Computed: true}, "operating_status": schema.StringAttribute{Computed: true}, "domain_name": schema.StringAttribute{Computed: true},
	})}
}
func (r *healthMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan healthMonitorResourceModel
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
	body := map[string]any{"name": optionalString(plan.Name), "pool_id": plan.PoolID.ValueString(), "type": plan.Type.ValueString(), "delay": plan.Delay.ValueInt64(), "timeout": plan.Timeout.ValueInt64(), "max_retries": plan.MaxRetries.ValueInt64(), "max_retries_down": plan.MaxRetriesDown.ValueInt64(), "url_path": plan.URLPath.ValueString(), "expected_codes": plan.ExpectedCodes.ValueString(), "http_method": plan.HTTPMethod.ValueString(), "admin_state_up": plan.AdminStateUp.ValueBool(), "tags": tags}
	result, err := api.CreateLBaaSResource(ctx, client.LBaaSHealthMonitors, scope.ProjectID, scope.AvailabilityZone, body)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create LBaaS health monitor", shared.DiagnosticDetail(err))
		return
	}
	var value healthMonitorAPI
	if err := decodeResource(result.Resource, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS health monitor", err.Error())
		return
	}
	plan.ProjectID, plan.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	applyHealthMonitor(ctx, &value, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *healthMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state healthMonitorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	raw, err := api.GetLBaaSResource(ctx, client.LBaaSHealthMonitors, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil)
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read LBaaS health monitor", shared.DiagnosticDetail(err))
		return
	}
	var value healthMonitorAPI
	if err := decodeResource(raw, &value); err != nil {
		resp.Diagnostics.AddError("Unable to decode LBaaS health monitor", err.Error())
		return
	}
	applyHealthMonitor(ctx, &value, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *healthMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state healthMonitorResourceModel
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
	raw, err := api.UpdateLBaaSResource(ctx, client.LBaaSHealthMonitors, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil, map[string]any{"name": optionalString(plan.Name), "delay": plan.Delay.ValueInt64(), "timeout": plan.Timeout.ValueInt64(), "max_retries": plan.MaxRetries.ValueInt64(), "max_retries_down": plan.MaxRetriesDown.ValueInt64(), "url_path": plan.URLPath.ValueString(), "expected_codes": plan.ExpectedCodes.ValueString(), "http_method": plan.HTTPMethod.ValueString(), "admin_state_up": plan.AdminStateUp.ValueBool(), "tags": tags})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update LBaaS health monitor", shared.DiagnosticDetail(err))
		return
	}
	var value healthMonitorAPI
	if decodeResource(raw, &value) == nil {
		applyHealthMonitor(ctx, &value, &plan, &resp.Diagnostics)
	} else {
		plan.ID = state.ID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *healthMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state healthMonitorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.DeleteLBaaSResource(ctx, client.LBaaSHealthMonitors, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString(), nil); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete LBaaS health monitor", shared.DiagnosticDetail(err))
	}
}
func (r *healthMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
func applyHealthMonitor(ctx context.Context, value *healthMonitorAPI, state *healthMonitorResourceModel, diagnostics *diag.Diagnostics) {
	state.ID = types.StringValue(value.ID)
	state.PoolID = plainStringValue(value.Pool)
	state.Name = stringPointerValue(value.Name)
	state.Type = plainStringValue(value.Type)
	state.Delay = types.Int64Value(value.Delay)
	state.Timeout = types.Int64Value(value.Timeout)
	state.MaxRetries = types.Int64Value(value.MaxRetries)
	state.MaxRetriesDown = types.Int64Value(value.MaxRetriesDown)
	state.URLPath = plainStringValue(value.URLPath)
	state.ExpectedCodes = plainStringValue(value.ExpectedCodes)
	state.HTTPMethod = plainStringValue(value.HTTPMethod)
	state.AdminStateUp = types.BoolValue(value.AdminStateUp)
	state.ProvisioningStatus = plainStringValue(value.ProvisioningStatus)
	state.OperatingStatus = plainStringValue(value.OperatingStatus)
	state.DomainName = stringPointerValue(value.DomainName)
	tags, ds := listFromStrings(ctx, value.Tags)
	diagnostics.Append(ds...)
	state.Tags = tags
}

var _ resource.ResourceWithConfigure = &healthMonitorResource{}
var _ resource.ResourceWithImportState = &healthMonitorResource{}
