// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lbaas

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type securityGroupRuleResource struct{ config *shared.Config }
type securityGroupRuleResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	LoadBalancerID   types.String `tfsdk:"load_balancer_id"`
	Port             types.Int64  `tfsdk:"port"`
	Source           types.String `tfsdk:"source"`
}

func NewSecurityGroupRuleResource() resource.Resource { return &securityGroupRuleResource{} }
func (r *securityGroupRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_security_group_rule"
}
func (r *securityGroupRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	r.config = config
}
func (r *securityGroupRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "A security-group rule attached to a Nobus LBaaS load balancer.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "load_balancer_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "port": schema.Int64Attribute{Required: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}}, "source": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings(), MarkdownDescription: "Source CIDR sent as the API `from` field."},
	})}
}
func (r *securityGroupRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan securityGroupRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if _, err := api.CreateLBaaSSecurityGroupRule(ctx, scope.ProjectID, scope.AvailabilityZone, plan.LoadBalancerID.ValueString(), plan.Port.ValueInt64(), plan.Source.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to create LBaaS security-group rule", shared.DiagnosticDetail(err))
		return
	}
	plan.ID = types.StringValue(fmt.Sprintf("%s/%d/%s", plan.LoadBalancerID.ValueString(), plan.Port.ValueInt64(), plan.Source.ValueString()))
	plan.ProjectID, plan.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *securityGroupRuleResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {
}
func (r *securityGroupRuleResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (r *securityGroupRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state securityGroupRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.DeleteLBaaSSecurityGroupRule(ctx, scope.ProjectID, scope.AvailabilityZone, state.LoadBalancerID.ValueString(), state.Port.ValueInt64(), state.Source.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete LBaaS security-group rule", shared.DiagnosticDetail(err))
	}
}

// ImportState rejects imports because the LBaaS security-group rule endpoint
// only supports creation and deletion; there is no read to refresh state.
func (r *securityGroupRuleResource) ImportState(_ context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import is not supported for Nobus LBaaS security-group rules",
		"The Nobus API does not expose a security-group rule read endpoint, so an existing rule cannot be imported. "+
			"Create a new nobus_lbaas_security_group_rule resource instead.",
	)
}

type failoverResource struct{ config *shared.Config }
type failoverResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	LoadBalancerID   types.String `tfsdk:"load_balancer_id"`
	Trigger          types.String `tfsdk:"trigger"`
}

func NewFailoverResource() resource.Resource { return &failoverResource{} }
func (r *failoverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_failover"
}
func (r *failoverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	r.config = config
}
func (r *failoverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Triggers a one-shot LBaaS load-balancer failover. Change trigger to run it again.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "load_balancer_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "trigger": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}})}
}
func (r *failoverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan failoverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.FailoverLBaaSLoadBalancer(ctx, scope.ProjectID, scope.AvailabilityZone, plan.LoadBalancerID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to fail over Nobus load balancer", shared.DiagnosticDetail(err))
		return
	}
	plan.ID = types.StringValue(plan.LoadBalancerID.ValueString() + "/" + plan.Trigger.ValueString())
	plan.ProjectID, plan.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *failoverResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {}
func (r *failoverResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (r *failoverResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

// ImportState rejects imports because failover is a one-shot operation that
// produces no durable, readable state.
func (r *failoverResource) ImportState(_ context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import is not supported for Nobus LBaaS failover",
		"Failover is a one-shot operation, not a durable resource, so it cannot be imported. "+
			"Create a new nobus_lbaas_failover resource to trigger a failover.",
	)
}

type provisionResource struct{ config *shared.Config }
type provisionResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	PayloadJSON      types.String `tfsdk:"payload_json"`
	TaskID           types.String `tfsdk:"task_id"`
	State            types.String `tfsdk:"state"`
}

func NewProvisionResource() resource.Resource { return &provisionResource{} }
func (r *provisionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lbaas_provision"
}
func (r *provisionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	r.config = config
}
func (r *provisionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Provisions a load balancer and its nested listeners/pools in one API call. payload_json follows LoadBalancerProvisionSchema.", Attributes: map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "project_id": schema.StringAttribute{Computed: true}, "availability_zone": schema.StringAttribute{Computed: true}, "payload_json": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "task_id": schema.StringAttribute{Computed: true}, "state": schema.StringAttribute{Computed: true}}}
}
func (r *provisionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan provisionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(plan.PayloadJSON.ValueString()), &payload); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("payload_json"), "Invalid LBaaS provision JSON", err.Error())
		return
	}
	result, err := api.ProvisionLBaaS(ctx, []byte(plan.PayloadJSON.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Unable to provision Nobus LBaaS stack", shared.DiagnosticDetail(err))
		return
	}
	var value apiResource
	if err := decodeResource(result.Resource, &value); err != nil || value.ID == "" {
		resp.Diagnostics.AddError("Nobus LBaaS provision API returned no load balancer ID", "The provision request succeeded but Terraform cannot safely track or delete the resulting load balancer.")
		return
	}
	plan.ID = types.StringValue(value.ID)
	if v, ok := payload["project_id"].(string); ok {
		plan.ProjectID = types.StringValue(v)
	}
	if v, ok := payload["availability_zone"].(string); ok {
		plan.AvailabilityZone = types.StringValue(v)
	}
	plan.TaskID, plan.State = plainStringValue(result.TaskID), plainStringValue(result.State)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *provisionResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {}
func (r *provisionResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (r *provisionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state provisionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if state.ProjectID.IsNull() || state.AvailabilityZone.IsNull() {
		return
	}
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	if err := api.DeleteLBaaSResource(ctx, client.LBaaSLoadBalancers, state.ProjectID.ValueString(), state.AvailabilityZone.ValueString(), state.ID.ValueString(), nil); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete provisioned LBaaS load balancer", shared.DiagnosticDetail(err))
	}
}

// ImportState rejects imports because the provision resource is a one-shot
// provisioning operation with no read endpoint to refresh state from.
func (r *provisionResource) ImportState(_ context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import is not supported for Nobus LBaaS provision",
		"Provisioning is a one-shot operation that produces a load balancer stack, not a readable resource. "+
			"Create a new nobus_lbaas_provision resource, or manage the individual load balancer, listener, and pool resources directly.",
	)
}

var _ resource.ResourceWithConfigure = &securityGroupRuleResource{}
var _ resource.ResourceWithImportState = &securityGroupRuleResource{}

var _ resource.ResourceWithConfigure = &failoverResource{}
var _ resource.ResourceWithImportState = &failoverResource{}

var _ resource.ResourceWithConfigure = &provisionResource{}
var _ resource.ResourceWithImportState = &provisionResource{}
