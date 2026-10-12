// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package fos

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type objectResource struct{ config *shared.Config }

type objectResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	ContainerName    types.String `tfsdk:"container_name"`
	ObjectName       types.String `tfsdk:"object_name"`
	Source           types.String `tfsdk:"source"`
	SourceHash       types.String `tfsdk:"source_hash"`
	Bytes            types.Int64  `tfsdk:"bytes"`
	ContentType      types.String `tfsdk:"content_type"`
	LastModified     types.String `tfsdk:"last_modified"`
	ETag             types.String `tfsdk:"etag"`
}

func NewObjectResource() resource.Resource { return &objectResource{} }

func (r *objectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_fos_object"
}

func (r *objectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	r.config = config
}

func (r *objectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An object uploaded to a Nobus FOS container.",
		Attributes: shared.MergeAttributes(shared.ScopeAttributes(), map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true},
			"container_name": schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()},
			"object_name":    schema.StringAttribute{Required: true, PlanModifiers: shared.ReplaceStrings()},
			"source":         schema.StringAttribute{Optional: true, MarkdownDescription: "Local file uploaded during creation or update. Optional only to support importing an existing object."},
			"source_hash":    schema.StringAttribute{Optional: true, MarkdownDescription: "User-supplied content hash, commonly `filesha256(source)`. Changing it uploads the source again."},
			"bytes":          schema.Int64Attribute{Computed: true},
			"content_type":   schema.StringAttribute{Computed: true},
			"last_modified":  schema.StringAttribute{Computed: true},
			"etag":           schema.StringAttribute{Computed: true},
		}),
	}
}

func (r *objectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan objectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Source.IsNull() || plan.Source.IsUnknown() || strings.TrimSpace(plan.Source.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(path.Root("source"), "Missing FOS object source", "Set source to the local file that should be uploaded.")
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
	if err := api.UploadFOSObject(ctx, scope.ProjectID, scope.AvailabilityZone, plan.ContainerName.ValueString(), plan.ObjectName.ValueString(), plan.Source.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to upload Nobus FOS object", shared.DiagnosticDetail(err))
		return
	}
	plan.ID = types.StringValue(plan.ContainerName.ValueString() + "/" + plan.ObjectName.ValueString())
	plan.ProjectID = types.StringValue(scope.ProjectID)
	plan.AvailabilityZone = types.StringValue(scope.AvailabilityZone)
	object, err := findObject(ctx, api, scope.ProjectID, scope.AvailabilityZone, plan.ContainerName.ValueString(), plan.ObjectName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read uploaded Nobus FOS object", shared.DiagnosticDetail(err))
		return
	}
	if object != nil {
		applyObject(object, &plan)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *objectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state objectResourceModel
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
	object, err := findObject(ctx, api, scope.ProjectID, scope.AvailabilityZone, state.ContainerName.ValueString(), state.ObjectName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Nobus FOS object", shared.DiagnosticDetail(err))
		return
	}
	if object == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	applyObject(object, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *objectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan objectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Source.IsNull() || plan.Source.IsUnknown() || strings.TrimSpace(plan.Source.ValueString()) == "" {
		resp.Diagnostics.AddAttributeError(path.Root("source"), "Missing FOS object source", "Set source to the local file that should be uploaded.")
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
	if err := api.UploadFOSObject(ctx, scope.ProjectID, scope.AvailabilityZone, plan.ContainerName.ValueString(), plan.ObjectName.ValueString(), plan.Source.ValueString()); err != nil {
		resp.Diagnostics.AddError("Unable to update Nobus FOS object", shared.DiagnosticDetail(err))
		return
	}
	object, err := findObject(ctx, api, scope.ProjectID, scope.AvailabilityZone, plan.ContainerName.ValueString(), plan.ObjectName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read updated Nobus FOS object", shared.DiagnosticDetail(err))
		return
	}
	if object != nil {
		applyObject(object, &plan)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *objectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state objectResourceModel
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
	if err := api.DeleteFOSObject(ctx, scope.ProjectID, scope.AvailabilityZone, state.ContainerName.ValueString(), state.ObjectName.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Nobus FOS object", shared.DiagnosticDetail(err))
	}
}

func (r *objectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid FOS object import ID", fmt.Sprintf("Expected container/object, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("container_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_name"), parts[1])...)
}

func findObject(ctx context.Context, api *client.Client, projectID, availabilityZone, containerName, objectName string) (*client.FOSObject, error) {
	objects, err := api.ListFOSObjects(ctx, projectID, availabilityZone, containerName, client.FOSListOptions{Prefix: objectName, FullListing: true})
	if err != nil {
		return nil, err
	}
	for i := range objects {
		if objects[i].Name == objectName {
			return &objects[i], nil
		}
	}
	return nil, nil
}

func applyObject(object *client.FOSObject, model *objectResourceModel) {
	model.Bytes = types.Int64Value(object.Bytes)
	model.ContentType = shared.StringOrNull(object.ContentType)
	model.LastModified = shared.StringOrNull(object.LastModified)
	model.ETag = shared.StringOrNull(object.ETag)
}

var _ resource.ResourceWithConfigure = &objectResource{}
var _ resource.ResourceWithImportState = &objectResource{}
