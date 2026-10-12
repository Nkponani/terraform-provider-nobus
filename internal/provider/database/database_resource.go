// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package database

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type databaseResource struct{ config *shared.Config }

type databaseResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	UserEmail        types.String `tfsdk:"user_email"`
	FlavorName       types.String `tfsdk:"flavor_name"`
	FlavorID         types.String `tfsdk:"flavor_id"`
	Name             types.String `tfsdk:"name"`
	DatabaseType     types.String `tfsdk:"database_type"`
	DatabaseVersion  types.String `tfsdk:"database_version"`
	Size             types.Int64  `tfsdk:"size"`
	InstallType      types.String `tfsdk:"install_type"`
	CustomerUsername types.String `tfsdk:"customer_username"`
	CustomerPassword types.String `tfsdk:"customer_password"`
	Zones            types.List   `tfsdk:"zones"`
	ZabbixID         types.String `tfsdk:"zabbix_id"`
	Encrypted        types.Bool   `tfsdk:"encrypted"`
	TaskID           types.String `tfsdk:"task_id"`
	Status           types.String `tfsdk:"status"`
	CreatedAt        types.String `tfsdk:"created_at"`
	InstanceCount    types.Int64  `tfsdk:"instance_count"`
	InstancesJSON    types.String `tfsdk:"instances_json"`
}

func NewDatabaseResource() resource.Resource { return &databaseResource{} }

func (r *databaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

func (r *databaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.config = config
}

func (r *databaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus managed database. Size, compute flavor, and the customer password can be changed in place through their dedicated API operations.", Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
		"id":                schema.StringAttribute{Computed: true},
		"user_email":        schema.StringAttribute{Required: true, PlanModifiers: replace},
		"flavor_name":       schema.StringAttribute{Required: true, PlanModifiers: replace, MarkdownDescription: "Flavor name used during initial provisioning."},
		"flavor_id":         schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Flavor ID used by the resize-flavor API. Changing it resizes the database instances."},
		"name":              schema.StringAttribute{Required: true, PlanModifiers: replace},
		"database_type":     schema.StringAttribute{Required: true, PlanModifiers: replace},
		"database_version":  schema.StringAttribute{Required: true, PlanModifiers: replace},
		"size":              schema.Int64Attribute{Required: true, MarkdownDescription: "Database volume size in GB. It can only be increased."},
		"install_type":      schema.StringAttribute{Required: true, PlanModifiers: replace},
		"customer_username": schema.StringAttribute{Required: true, PlanModifiers: replace},
		"customer_password": schema.StringAttribute{Required: true, Sensitive: true, MarkdownDescription: "Changing this value calls the database credential-rotation endpoint."},
		"zones":             schema.ListAttribute{Required: true, ElementType: types.StringType, PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()}},
		"zabbix_id":         schema.StringAttribute{Optional: true, PlanModifiers: replace},
		"encrypted":         schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
		"task_id":           schema.StringAttribute{Computed: true},
		"status":            schema.StringAttribute{Computed: true},
		"created_at":        schema.StringAttribute{Computed: true},
		"instance_count":    schema.Int64Attribute{Computed: true},
		"instances_json":    schema.StringAttribute{Computed: true, Sensitive: true},
	})}
}

func (r *databaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan databaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	var zones []string
	resp.Diagnostics.Append(plan.Zones.ElementsAs(ctx, &zones, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	encrypted := plan.Encrypted.ValueBool()
	result, err := api.CreateDatabase(ctx, client.DatabaseCreateRequest{
		ProjectID: scope.ProjectID, AvailabilityZone: scope.AvailabilityZone, UserEmail: plan.UserEmail.ValueString(),
		FlavorName: plan.FlavorName.ValueString(), DatabaseName: plan.Name.ValueString(), DatabaseType: plan.DatabaseType.ValueString(),
		DatabaseVersion: plan.DatabaseVersion.ValueString(), DatabaseSize: plan.Size.ValueInt64(), InstallType: plan.InstallType.ValueString(),
		CustomerUserName: plan.CustomerUsername.ValueString(), CustomerPassword: plan.CustomerPassword.ValueString(), Zones: zones,
		ZabbixID: plan.ZabbixID.ValueString(), Encrypted: &encrypted,
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Nobus database", shared.DiagnosticDetail(err))
		return
	}
	if result.DatabaseID == "" {
		resp.Diagnostics.AddError("Nobus database API returned no database ID", "The create request succeeded but its response data did not contain database_id, so Terraform cannot track the database.")
		return
	}
	plan.ID = types.StringValue(result.DatabaseID)
	plan.ProjectID, plan.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	plan.TaskID = stringOrNull(result.TaskID)
	if plan.FlavorID.IsUnknown() {
		plan.FlavorID = types.StringNull()
	}
	plan.Status, plan.CreatedAt, plan.InstanceCount, plan.InstancesJSON = types.StringNull(), types.StringNull(), types.Int64Null(), types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	value, err := api.GetDatabase(ctx, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Nobus database", shared.DiagnosticDetail(err))
		return
	}
	if value.DatabaseName != "" {
		state.Name = types.StringValue(value.DatabaseName)
	}
	if value.DatabaseType != "" {
		state.DatabaseType = types.StringValue(value.DatabaseType)
	}
	if value.DatabaseVersion != "" {
		state.DatabaseVersion = types.StringValue(value.DatabaseVersion)
	}
	installType := value.InstallationType
	if installType == "" {
		installType = value.InstallType
	}
	if installType != "" {
		state.InstallType = types.StringValue(installType)
	}
	state.Status, state.CreatedAt = stringOrNull(value.Status), stringOrNull(value.CreatedAt)
	state.InstanceCount = types.Int64Value(value.InstanceCount)
	state.InstancesJSON = rawOrNull(value.Instances)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *databaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state databaseResourceModel
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
	databaseID := state.ID.ValueString()
	if !plan.Size.Equal(state.Size) {
		if plan.Size.ValueInt64() < state.Size.ValueInt64() {
			resp.Diagnostics.AddAttributeError(path.Root("size"), "Database size cannot be reduced", fmt.Sprintf("The current size is %d GB; requested %d GB.", state.Size.ValueInt64(), plan.Size.ValueInt64()))
			return
		}
		if err := api.ResizeDatabase(ctx, scope.ProjectID, scope.AvailabilityZone, databaseID, plan.Size.ValueInt64()); err != nil {
			resp.Diagnostics.AddError("Unable to resize Nobus database", shared.DiagnosticDetail(err))
			return
		}
	}
	if !plan.FlavorID.Equal(state.FlavorID) && !plan.FlavorID.IsNull() && !plan.FlavorID.IsUnknown() {
		if err := api.ResizeDatabaseFlavor(ctx, scope.ProjectID, scope.AvailabilityZone, databaseID, plan.FlavorID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Unable to resize Nobus database flavor", shared.DiagnosticDetail(err))
			return
		}
	}
	if !plan.CustomerPassword.Equal(state.CustomerPassword) {
		if err := api.ChangeDatabasePassword(ctx, scope.ProjectID, scope.AvailabilityZone, databaseID, plan.DatabaseType.ValueString(), plan.CustomerPassword.ValueString()); err != nil {
			resp.Diagnostics.AddError("Unable to rotate Nobus database password", shared.DiagnosticDetail(err))
			return
		}
	}
	plan.ID, plan.TaskID, plan.Status, plan.CreatedAt, plan.InstanceCount, plan.InstancesJSON = state.ID, state.TaskID, state.Status, state.CreatedAt, state.InstanceCount, state.InstancesJSON
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *databaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state databaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.DeleteDatabase(ctx, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Nobus database", shared.DiagnosticDetail(err))
	}
}

func (r *databaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func stringOrNull(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}
func rawOrNull(value []byte) types.String {
	if len(value) == 0 || string(value) == "null" {
		return types.StringNull()
	}
	return types.StringValue(string(value))
}

var _ resource.ResourceWithConfigure = &databaseResource{}
var _ resource.ResourceWithImportState = &databaseResource{}
