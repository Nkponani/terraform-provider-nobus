// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type migrationResource struct{ config *shared.Config }
type migrationResourceModel struct {
	ID          types.String `tfsdk:"id"`
	PayloadJSON types.String `tfsdk:"payload_json"`
}

func NewMigrationResource() resource.Resource { return &migrationResource{} }
func (r *migrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_migration"
}
func (r *migrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	config, ds := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(ds...)
	r.config = config
}
func (r *migrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Backfills Nobus inventory records for an existing Kubernetes cluster without provisioning or deleting infrastructure.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "payload_json": schema.StringAttribute{Required: true, Sensitive: true, PlanModifiers: shared.ReplaceStrings(), MarkdownDescription: "JSON object matching K8sMigrationRequestSchema."},
	}}
}
func (r *migrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan migrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	api := shared.RequireConfigured(r.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	payload := []byte(plan.PayloadJSON.ValueString())
	var value map[string]any
	if err := json.Unmarshal(payload, &value); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("payload_json"), "Invalid Kubernetes migration JSON", err.Error())
		return
	}
	if _, ok := value["nodes"].([]any); !ok {
		resp.Diagnostics.AddAttributeError(path.Root("payload_json"), "Missing Kubernetes migration nodes", "payload_json must contain the nodes array required by K8sMigrationRequestSchema.")
		return
	}
	if err := api.MigrateKubernetesCluster(ctx, payload); err != nil {
		resp.Diagnostics.AddError("Unable to migrate existing Kubernetes cluster", shared.DiagnosticDetail(err))
		return
	}
	if id, ok := value["cluster_id"].(string); ok && id != "" {
		plan.ID = types.StringValue(id)
	} else {
		digest := sha256.Sum256(payload)
		plan.ID = types.StringValue(hex.EncodeToString(digest[:]))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *migrationResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {}
func (r *migrationResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
}
func (r *migrationResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
}

// ImportState rejects imports because the migration resource is a one-shot
// backfill operation with no read endpoint to refresh state from.
func (r *migrationResource) ImportState(_ context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.AddError(
		"Import is not supported for Nobus Kubernetes migration",
		"Migration is a one-shot inventory backfill operation, not a durable resource, so it cannot be imported. "+
			"Create a new nobus_kubernetes_migration resource to backfill an existing cluster.",
	)
}

var _ resource.ResourceWithConfigure = &migrationResource{}
var _ resource.ResourceWithImportState = &migrationResource{}
