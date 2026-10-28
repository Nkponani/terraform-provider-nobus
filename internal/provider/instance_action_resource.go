// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

var supportedInstanceActions = []string{
	"shutdown", "reboot", "resize", "confirm_resize", "revert_resize",
	"suspend", "resume", "lock", "unlock", "attach_interface",
	"detach_interface", "attach_security_group", "remove_security_group",
	"create_image", "change_password", "clear_password", "set_tags",
	"delete_tag", "console_output", "console_url",
}

type instanceActionResource struct{ config *providerConfig }

type instanceActionModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectID         types.String `tfsdk:"project_id"`
	AvailabilityZone  types.String `tfsdk:"availability_zone"`
	InstanceID        types.String `tfsdk:"instance_id"`
	Action            types.String `tfsdk:"action"`
	RebootType        types.String `tfsdk:"reboot_type"`
	FlavorID          types.String `tfsdk:"flavor_id"`
	PortID            types.String `tfsdk:"port_id"`
	NetworkID         types.String `tfsdk:"network_id"`
	FixedIP           types.String `tfsdk:"fixed_ip"`
	SecurityGroupName types.String `tfsdk:"security_group_name"`
	ImageName         types.String `tfsdk:"image_name"`
	NewPassword       types.String `tfsdk:"new_password"`
	Tags              types.List   `tfsdk:"tags"`
	Tag               types.String `tfsdk:"tag"`
	Result            types.String `tfsdk:"result"`
}

func NewInstanceActionResource() resource.Resource { return &instanceActionResource{} }
func (r *instanceActionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_action"
}
func (r *instanceActionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *instanceActionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceList := []planmodifier.List{listplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Runs a one-shot Nobus compute operation. Changing any input runs a replacement action.",
		Attributes: mergeAttributes(scopeAttributes(), map[string]schema.Attribute{
			"id":                  schema.StringAttribute{Computed: true},
			"instance_id":         schema.StringAttribute{Required: true, Validators: []validator.String{StringNotEmpty()}, PlanModifiers: replaceStrings()},
			"action":              schema.StringAttribute{Required: true, Validators: []validator.String{StringOneOf(supportedInstanceActions...)}, PlanModifiers: replaceStrings()},
			"reboot_type":         schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()},
			"flavor_id":           schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()},
			"port_id":             schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()},
			"network_id":          schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()},
			"fixed_ip":            schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()},
			"security_group_name": schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()},
			"image_name":          schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()},
			"new_password":        schema.StringAttribute{Optional: true, Sensitive: true, PlanModifiers: replaceStrings()},
			"tags":                schema.ListAttribute{Optional: true, ElementType: types.StringType, PlanModifiers: replaceList},
			"tag":                 schema.StringAttribute{Optional: true, PlanModifiers: replaceStrings()},
			"result":              schema.StringAttribute{Computed: true},
		}),
	}
}
func (r *instanceActionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p instanceActionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	s, ok := r.config.ResolveScope(p.ProjectID, p.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	tags, d := listToStringSlice(ctx, p.Tags)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	action := p.Action.ValueString()
	required := map[string]types.String{}
	switch action {
	case "reboot":
		required["reboot_type"] = p.RebootType
	case "resize":
		required["flavor_id"] = p.FlavorID
	case "detach_interface":
		required["port_id"] = p.PortID
	case "attach_security_group", "remove_security_group":
		required["security_group_name"] = p.SecurityGroupName
	case "create_image":
		required["image_name"] = p.ImageName
	case "change_password":
		required["new_password"] = p.NewPassword
	case "delete_tag":
		required["tag"] = p.Tag
	}
	for name, value := range required {
		if nullSafeString(value) == "" {
			resp.Diagnostics.AddError("Missing instance action input", name+" is required for action "+action)
			return
		}
	}
	if action == "set_tags" && len(tags) == 0 {
		resp.Diagnostics.AddError("Missing instance action input", "tags is required for action set_tags")
		return
	}
	result, err := c.InstanceAction(ctx, action, client.InstanceActionRequest{
		ProjectID: s.ProjectID, AvailabilityZone: s.AvailabilityZone, ServerID: p.InstanceID.ValueString(),
		RebootType: optionalString(p.RebootType), FlavorID: optionalString(p.FlavorID), PortID: optionalString(p.PortID),
		NetworkID: optionalString(p.NetworkID), FixedIP: optionalString(p.FixedIP), SecurityGroupName: optionalString(p.SecurityGroupName),
		ImageName: optionalString(p.ImageName), NewPassword: optionalString(p.NewPassword), Tags: tags, Tag: optionalString(p.Tag),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to run Nobus instance action", diagnosticDetail(err))
		return
	}
	h := sha256.Sum256([]byte(strings.Join([]string{p.InstanceID.ValueString(), action, p.FlavorID.ValueString(), p.PortID.ValueString(), p.Tag.ValueString()}, "|")))
	p.ID = types.StringValue(hex.EncodeToString(h[:12]))
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	if len(result) == 0 {
		p.Result = types.StringValue("null")
	} else {
		p.Result = types.StringValue(string(result))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *instanceActionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}
func (r *instanceActionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}
func (r *instanceActionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

// ImportState rejects imports because instance actions are one-shot
// operations that produce no durable, readable state.
func (r *instanceActionResource) ImportState(_ context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import is not supported for Nobus instance actions",
		"Instance actions are one-shot operations, not durable resources, so they cannot be imported. "+
			"Create a new nobus_instance_action resource to run an operation against an instance.",
	)
}

var _ resource.Resource = &instanceActionResource{}
var _ resource.ResourceWithConfigure = &instanceActionResource{}
var _ resource.ResourceWithImportState = &instanceActionResource{}
