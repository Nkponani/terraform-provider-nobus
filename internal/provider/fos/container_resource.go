// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fos

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type containerResource struct{ config *shared.Config }

type containerResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	IsPublic         types.Bool   `tfsdk:"is_public"`
	ObjectCount      types.Int64  `tfsdk:"object_count"`
	Bytes            types.Int64  `tfsdk:"bytes"`
	LastModified     types.String `tfsdk:"last_modified"`
}

func NewContainerResource() resource.Resource { return &containerResource{} }

func (r *containerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fos_container"
}

func (r *containerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.config = config
}

func (r *containerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Nobus FOS object-storage container.",
		Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
			"id":            schema.StringAttribute{Computed: true},
			"name":          schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()},
			"is_public":     schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"object_count":  schema.Int64Attribute{Computed: true},
			"bytes":         schema.Int64Attribute{Computed: true},
			"last_modified": schema.StringAttribute{Computed: true},
		}),
	}
}

func (r *containerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan containerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil {
		return
	}
	scope, ok := r.config.ResolveScope(plan.ProjectID, plan.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	isPublic := false
	if !plan.IsPublic.IsNull() && !plan.IsPublic.IsUnknown() {
		isPublic = plan.IsPublic.ValueBool()
	}
	if err := api.CreateFOSContainer(ctx, scope.ProjectID, scope.AvailabilityZone, plan.Name.ValueString(), isPublic); err != nil {
		resp.Diagnostics.AddError("Unable to create Nobus FOS container", shared.DiagnosticDetail(err))
		return
	}
	plan.ID = types.StringValue(plan.Name.ValueString())
	plan.ProjectID = types.StringValue(scope.ProjectID)
	plan.AvailabilityZone = types.StringValue(scope.AvailabilityZone)
	plan.IsPublic = types.BoolValue(isPublic)
	if container, err := findContainer(ctx, api, scope.ProjectID, scope.AvailabilityZone, plan.Name.ValueString()); err == nil && container != nil {
		applyContainer(container, &plan)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *containerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state containerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	name := state.ID.ValueString()
	if !state.Name.IsNull() && !state.Name.IsUnknown() {
		name = state.Name.ValueString()
	}
	container, err := findContainer(ctx, api, scope.ProjectID, scope.AvailabilityZone, name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Nobus FOS container", shared.DiagnosticDetail(err))
		return
	}
	if container == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(container.Name)
	state.Name = types.StringValue(container.Name)
	applyContainer(container, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *containerResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}

func (r *containerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state containerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil {
		return
	}
	scope, ok := r.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if err := api.DeleteFOSContainer(ctx, scope.ProjectID, scope.AvailabilityZone, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Nobus FOS container", shared.DiagnosticDetail(err))
	}
}

func (r *containerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

func findContainer(ctx context.Context, api *client.Client, projectID, availabilityZone, name string) (*client.FOSContainer, error) {
	containers, err := api.ListFOSContainers(ctx, projectID, availabilityZone, client.FOSListOptions{Prefix: name, FullListing: true})
	if err != nil {
		return nil, err
	}
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i], nil
		}
	}
	return nil, nil
}

func applyContainer(container *client.FOSContainer, model *containerResourceModel) {
	model.ObjectCount = types.Int64Value(container.Count)
	model.Bytes = types.Int64Value(container.Bytes)
	model.LastModified = shared.StringOrNull(container.LastModified)
}

var _ resource.ResourceWithConfigure = &containerResource{}
var _ resource.ResourceWithImportState = &containerResource{}
