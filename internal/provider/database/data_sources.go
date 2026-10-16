// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package database

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

type databasesDataSource struct{ config *shared.Config }
type databasesDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	ResultCount      types.Int64  `tfsdk:"result_count"`
	IDs              types.List   `tfsdk:"ids"`
	Names            types.List   `tfsdk:"names"`
	DatabasesJSON    types.String `tfsdk:"databases_json"`
}

func NewDatabasesDataSource() datasource.DataSource { return &databasesDataSource{} }
func (d *databasesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_databases"
}
func (d *databasesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.config = config
}
func (d *databasesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Lists Nobus managed databases.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "project_id": schema.StringAttribute{Optional: true, Computed: true}, "availability_zone": schema.StringAttribute{Optional: true, Computed: true},
		"result_count": schema.Int64Attribute{Computed: true}, "ids": schema.ListAttribute{Computed: true, ElementType: types.StringType}, "names": schema.ListAttribute{Computed: true, ElementType: types.StringType},
		"databases_json": schema.StringAttribute{Computed: true},
	}}
}
func (d *databasesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state databasesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	values, err := api.ListDatabases(ctx, scope.ProjectID, scope.AvailabilityZone)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Nobus databases", shared.DiagnosticDetail(err))
		return
	}
	ids, names := make([]string, 0, len(values)), make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.DatabaseID)
		names = append(names, value.DatabaseName)
	}
	idList, d1 := types.ListValueFrom(ctx, types.StringType, ids)
	nameList, d2 := types.ListValueFrom(ctx, types.StringType, names)
	resp.Diagnostics.Append(d1...)
	resp.Diagnostics.Append(d2...)
	raw, err := json.Marshal(values)
	if err != nil {
		resp.Diagnostics.AddError("Unable to encode Nobus databases", err.Error())
		return
	}
	state.ID = types.StringValue(scope.ProjectID + "/" + scope.AvailabilityZone)
	state.ProjectID, state.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	state.ResultCount, state.IDs, state.Names, state.DatabasesJSON = types.Int64Value(int64(len(values))), idList, nameList, types.StringValue(string(raw))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type credentialsDataSource struct{ config *shared.Config }
type credentialsDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	DatabaseID       types.String `tfsdk:"database_id"`
	DatabaseName     types.String `tfsdk:"database_name"`
	DatabaseType     types.String `tfsdk:"database_type"`
	InstallType      types.String `tfsdk:"install_type"`
	Status           types.String `tfsdk:"status"`
	VolumeSize       types.Int64  `tfsdk:"volume_size"`
	CustomerName     types.String `tfsdk:"customer_name"`
	CustomerPassword types.String `tfsdk:"customer_password"`
	InstancesJSON    types.String `tfsdk:"instances_json"`
}

func NewCredentialsDataSource() datasource.DataSource { return &credentialsDataSource{} }
func (d *credentialsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_credentials"
}
func (d *credentialsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.config = config
}
func (d *credentialsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Reads connection credentials for a Nobus managed database. Credentials are stored in Terraform state as sensitive values.", Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true}, "project_id": schema.StringAttribute{Optional: true, Computed: true}, "availability_zone": schema.StringAttribute{Optional: true, Computed: true}, "database_id": schema.StringAttribute{Required: true},
		"database_name": schema.StringAttribute{Computed: true}, "database_type": schema.StringAttribute{Computed: true}, "install_type": schema.StringAttribute{Computed: true}, "status": schema.StringAttribute{Computed: true},
		"volume_size": schema.Int64Attribute{Computed: true}, "customer_name": schema.StringAttribute{Computed: true, Sensitive: true}, "customer_password": schema.StringAttribute{Computed: true, Sensitive: true}, "instances_json": schema.StringAttribute{Computed: true, Sensitive: true},
	}}
}
func (d *credentialsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state credentialsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(state.ProjectID, state.AvailabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	value, err := api.GetDatabaseCredentials(ctx, scope.ProjectID, scope.AvailabilityZone, state.DatabaseID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Nobus database credentials", shared.DiagnosticDetail(err))
		return
	}
	state.ID = state.DatabaseID
	state.ProjectID, state.AvailabilityZone = types.StringValue(scope.ProjectID), types.StringValue(scope.AvailabilityZone)
	state.DatabaseName, state.DatabaseType, state.InstallType, state.Status = stringOrNull(value.DatabaseName), stringOrNull(value.DatabaseType), stringOrNull(value.InstallType), stringOrNull(value.Status)
	state.VolumeSize, state.CustomerName, state.CustomerPassword, state.InstancesJSON = types.Int64Value(value.VolumeSize), stringOrNull(value.CustomerName), stringOrNull(value.CustomerPassword), rawOrNull(value.Instances)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type rawKind string

const (
	rawCreation rawKind = "creation_status"
	rawStatus   rawKind = "status"
	rawZabbix   rawKind = "zabbix_credentials"
	rawLogs     rawKind = "logs"
)

type rawDataSource struct {
	config *shared.Config
	kind   rawKind
}

func NewCreationStatusDataSource() datasource.DataSource    { return &rawDataSource{kind: rawCreation} }
func NewStatusDataSource() datasource.DataSource            { return &rawDataSource{kind: rawStatus} }
func NewZabbixCredentialsDataSource() datasource.DataSource { return &rawDataSource{kind: rawZabbix} }
func NewLogsDataSource() datasource.DataSource              { return &rawDataSource{kind: rawLogs} }
func (d *rawDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_" + string(d.kind)
}
func (d *rawDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	config, diags := shared.FromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	d.config = config
}
func (d *rawDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := map[string]schema.Attribute{"id": schema.StringAttribute{Computed: true}, "project_id": schema.StringAttribute{Optional: true, Computed: true}, "availability_zone": schema.StringAttribute{Optional: true, Computed: true}, "database_id": schema.StringAttribute{Required: true}, "result_json": schema.StringAttribute{Computed: true, Sensitive: d.kind == rawZabbix || d.kind == rawLogs}}
	if d.kind == rawStatus {
		attributes["replica_check"] = schema.BoolAttribute{Optional: true}
	}
	if d.kind == rawLogs {
		attributes["database_type"] = schema.StringAttribute{Required: true}
		attributes["line_count"] = schema.Int64Attribute{Required: true}
	}
	resp.Schema = schema.Schema{MarkdownDescription: "Reads the Nobus database " + string(d.kind) + " endpoint and exposes its complete JSON response.", Attributes: attributes}
}
func (d *rawDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var projectID, availabilityZone, databaseID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project_id"), &projectID)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("availability_zone"), &availabilityZone)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("database_id"), &databaseID)...)
	api := shared.RequireConfigured(d.config, &resp.Diagnostics)
	if api == nil || resp.Diagnostics.HasError() {
		return
	}
	scope, ok := d.config.ResolveScope(projectID, availabilityZone, &resp.Diagnostics)
	if !ok {
		return
	}
	var raw json.RawMessage
	var err error
	switch d.kind {
	case rawCreation:
		raw, err = api.GetDatabaseCreationStatus(ctx, scope.ProjectID, scope.AvailabilityZone, databaseID.ValueString())
	case rawStatus:
		var replicaCheck types.Bool
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("replica_check"), &replicaCheck)...)
		var check *bool
		if !replicaCheck.IsNull() && !replicaCheck.IsUnknown() {
			value := replicaCheck.ValueBool()
			check = &value
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("replica_check"), replicaCheck)...)
		raw, err = api.GetDatabaseStatus(ctx, scope.ProjectID, scope.AvailabilityZone, databaseID.ValueString(), check)
	case rawZabbix:
		raw, err = api.GetDatabaseZabbixCredentials(ctx, scope.ProjectID, scope.AvailabilityZone, databaseID.ValueString())
	case rawLogs:
		var databaseType types.String
		var lineCount types.Int64
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("database_type"), &databaseType)...)
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("line_count"), &lineCount)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_type"), databaseType)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("line_count"), lineCount)...)
		raw, err = api.GetDatabaseLogs(ctx, scope.ProjectID, scope.AvailabilityZone, databaseID.ValueString(), databaseType.ValueString(), lineCount.ValueInt64())
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Nobus database "+string(d.kind), shared.DiagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), databaseID.ValueString())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), databaseID.ValueString())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), scope.ProjectID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("availability_zone"), scope.AvailabilityZone)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("result_json"), rawOrNull(raw))...)
}

var _ datasource.DataSourceWithConfigure = &databasesDataSource{}
var _ datasource.DataSourceWithConfigure = &credentialsDataSource{}
var _ datasource.DataSourceWithConfigure = &rawDataSource{}
