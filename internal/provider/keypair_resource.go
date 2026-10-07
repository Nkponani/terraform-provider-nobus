// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

type keypairResource struct{ config *providerConfig }
type keypairResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	PublicKey        types.String `tfsdk:"public_key"`
	Fingerprint      types.String `tfsdk:"fingerprint"`
	UserID           types.String `tfsdk:"user_id"`
	Type             types.String `tfsdk:"type"`
}

func NewKeypairResource() resource.Resource { return &keypairResource{} }
func (r *keypairResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keypair"
}
func (r *keypairResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, d := providerConfigFromResourceData(req.ProviderData)
	resp.Diagnostics.Append(d...)
	r.config = c
}
func (r *keypairResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{MarkdownDescription: "A Nobus SSH keypair.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "project_id": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace}, "availability_zone": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace},
		"name": schema.StringAttribute{Required: true, Validators: []validator.String{StringNotEmpty()}, PlanModifiers: replace}, "public_key": schema.StringAttribute{Computed: true}, "fingerprint": schema.StringAttribute{Computed: true}, "user_id": schema.StringAttribute{Computed: true}, "type": schema.StringAttribute{Computed: true},
	}}
}
func (r *keypairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p keypairResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	s, ok := r.config.resolveScope(p.ProjectID, p.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	k, e := c.CreateKeypair(ctx, client.CreateKeypairRequest{Name: p.Name.ValueString(), ProjectID: s.ProjectID, AvailabilityZone: s.AvailabilityZone})
	if e != nil {
		resp.Diagnostics.AddError("Unable to create Nobus keypair", diagnosticDetail(e))
		return
	}
	applyKeypair(k, &p)
	p.ProjectID = types.StringValue(s.ProjectID)
	p.AvailabilityZone = types.StringValue(s.AvailabilityZone)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *keypairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s keypairResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	scope, ok := r.config.resolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	ks, e := c.ListKeypairs(ctx, scope.ProjectID, scope.AvailabilityZone)
	if e != nil {
		resp.Diagnostics.AddError("Unable to read Nobus keypair", diagnosticDetail(e))
		return
	}
	for _, k := range ks {
		if k.ID != nil && *k.ID == s.ID.ValueString() {
			applyKeypair(&k, &s)
			resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}
func (r *keypairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Keypairs cannot be updated", "Change the name to replace the keypair.")
}
func (r *keypairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s keypairResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	c := requireConfigured(r.config, &resp.Diagnostics)
	if c == nil {
		return
	}
	scope, ok := r.config.resolveScope(s.ProjectID, s.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	if e := c.DeleteKeypair(ctx, scope.ProjectID, scope.AvailabilityZone, s.ID.ValueString()); e != nil && !client.IsNotFound(e) {
		resp.Diagnostics.AddError("Unable to delete Nobus keypair", diagnosticDetail(e))
	}
}
func (r *keypairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
func applyKeypair(k *client.Keypair, m *keypairResourceModel) {
	if k.ID != nil {
		m.ID = types.StringValue(*k.ID)
	}
	m.Name = stringOrNull(k.Name)
	m.PublicKey = stringOrNull(k.PublicKey)
	m.Fingerprint = stringOrNull(k.Fingerprint)
	m.UserID = stringOrNull(k.UserID)
	m.Type = stringOrNull(k.Type)
}

var _ resource.Resource = &keypairResource{}
var _ resource.ResourceWithConfigure = &keypairResource{}
var _ resource.ResourceWithImportState = &keypairResource{}
var _ = fmt.Sprintf
