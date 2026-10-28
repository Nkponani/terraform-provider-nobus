// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type actionResource struct{ config *shared.Config }
type actionResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	DatabaseID       types.String `tfsdk:"database_id"`
	DatabaseType     types.String `tfsdk:"database_type"`
	Action           types.String `tfsdk:"action"`
}

func NewActionResource() resource.Resource { return &actionResource{} }
func (r *actionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_action"
}
func (r *actionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.config = config
}
func (r *actionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Runs a one-shot Nobus database lifecycle action. Set action to `resume` to use the dedicated resume endpoint; other values use the general action endpoint.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "database_id": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()},
		"database_type": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()}, "action": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()},
	})}
}
func (r *actionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan actionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	var err error
	if strings.EqualFold(plan.Action.ValueString(), "resume") {
		err = api.ResumeDatabase(ctx, scope.ProjectID, scope.AvailabilityZone, plan.DatabaseID.ValueString())
	} else {
		err = api.DatabaseAction(ctx, scope.ProjectID, scope.AvailabilityZone, plan.DatabaseID.ValueString(), plan.DatabaseType.ValueString(), plan.Action.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to run Nobus database action", shared.DiagnosticDetail(err))
		return
	}
	plan.ID = types.StringValue(plan.DatabaseID.ValueString() + "/" + plan.Action.ValueString())
	plan.ProjectID, plan.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *actionResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse)       {}
func (r *actionResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {}
func (r *actionResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {}

// ImportState rejects imports because database actions are one-shot operations
// that produce no durable, readable state.
func (r *actionResource) ImportState(_ context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import is not supported for Nobus database actions",
		"Database actions are one-shot operations, not durable resources, so they cannot be imported. "+
			"Create a new nobus_database_action resource to run an action against a database.",
	)
}

type existingResource struct{ config *shared.Config }
type existingResourceModel struct {
	ID          types.String `tfsdk:"id"`
	PayloadJSON types.String `tfsdk:"payload_json"`
}

func NewExistingResource() resource.Resource { return &existingResource{} }
func (r *existingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_existing_databases"
}
func (r *existingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.config = config
}
func (r *existingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Registers one or more existing customer-managed databases. This is a create-only inventory operation; destroying this Terraform resource does not delete a database.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "payload_json": schema.StringAttribute{Required: true, Sensitive: true, PlanModifiers: shared.ReplaceStrings(), MarkdownDescription: "JSON array matching the OpenAPI ExistingDbSchema."},
	}}
}
func (r *existingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan existingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	payload := []byte(plan.PayloadJSON.ValueString())
	var values []map[string]any
	if err := json.Unmarshal(payload, &values); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("payload_json"), "Invalid existing-database JSON", err.Error())
		return
	}
	if len(values) == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("payload_json"), "Empty existing-database payload", "The JSON array must contain at least one database.")
		return
	}
	if err := api.RegisterExistingDatabases(ctx, payload); err != nil {
		resp.Diagnostics.AddError("Unable to register existing Nobus databases", shared.DiagnosticDetail(err))
		return
	}
	digest := sha256.Sum256(payload)
	plan.ID = types.StringValue(hex.EncodeToString(digest[:]))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *existingResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {}
func (r *existingResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (r *existingResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

// ImportState rejects imports because the existing-databases resource is a
// one-shot inventory registration with no read endpoint to refresh state from.
func (r *existingResource) ImportState(_ context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import is not supported for Nobus existing databases",
		"Registering existing databases is a one-shot inventory operation, not a durable resource, so it cannot be imported. "+
			"Create a new nobus_existing_databases resource to register existing databases.",
	)
}

var _ resource.ResourceWithConfigure = &actionResource{}
var _ resource.ResourceWithImportState = &actionResource{}
var _ resource.ResourceWithConfigure = &existingResource{}
var _ resource.ResourceWithImportState = &existingResource{}
