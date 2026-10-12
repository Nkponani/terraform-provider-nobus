// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type taskStatusDataSource struct{ config *shared.Config }
type taskStatusDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	TaskID   types.String `tfsdk:"task_id"`
	Status   types.String `tfsdk:"status"`
	DataJSON types.String `tfsdk:"data_json"`
}

func NewTaskStatusDataSource() datasource.DataSource { return &taskStatusDataSource{} }
func (d *taskStatusDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_task_status"
}
func (d *taskStatusDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.config = config
}
func (d *taskStatusDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Reads a Nobus Kubernetes provisioning task status.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "task_id": schema.StringAttribute{Required: true}, "status": schema.StringAttribute{Computed: true}, "data_json": schema.StringAttribute{Computed: true, Sensitive: true},
	}}
}
func (d *taskStatusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state taskStatusDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	value, err := api.GetKubernetesTaskStatus(ctx, state.TaskID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Kubernetes task status", shared.DiagnosticDetail(err))
		return
	}
	state.ID = state.TaskID
	state.Status = stringValueOrNull(value.Status)
	state.DataJSON = rawJSONValue(value.Data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

var _ datasource.DataSourceWithConfigure = &taskStatusDataSource{}
