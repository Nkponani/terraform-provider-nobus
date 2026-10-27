// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package database

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
	"github.com/nobus-cloud/terraform-provider-nobus/internal/provider/shared"
)

func newDatabaseTestResource(t *testing.T, encrypted *bool) (*databaseResource, schema.Schema) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v3/database" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("project_id") != "p-1" || r.URL.Query().Get("database_id") != "db-1" {
			http.Error(w, "unexpected query", http.StatusBadRequest)
			return
		}
		response := map[string]any{
			"database_id":       "db-1",
			"database_name":     "orders",
			"database_type":     "postgresql",
			"database_version":  "16",
			"installation_type": "single",
			"project_id":        "p-1",
			"created_at":        "2026-01-01T00:00:00Z",
			"status":            "active",
			"instance_count":    1,
		}
		if encrypted != nil {
			response["encrypted"] = *encrypted
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encoding the fake response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	api, err := client.New(client.Config{
		APIKey:     "test-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	r := &databaseResource{config: &shared.Config{
		Client:           api,
		ProjectID:        "p-1",
		AvailabilityZone: client.DefaultAvailabilityZone,
	}}
	schemaResponse := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResponse)
	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResponse.Diagnostics.Errors())
	}
	return r, schemaResponse.Schema
}

func TestDatabaseReadReadsEncryption(t *testing.T) {
	t.Parallel()

	encrypted := true
	r, schema := newDatabaseTestResource(t, &encrypted)
	ctx := context.Background()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(ctx, databaseResourceModel{
		ID:               types.StringValue("db-1"),
		ProjectID:        types.StringValue("p-1"),
		AvailabilityZone: types.StringValue(client.DefaultAvailabilityZone),
		Zones:            types.ListNull(types.StringType),
		Encrypted:        types.BoolNull(),
	}); diags.HasError() {
		t.Fatalf("building the imported state: %v", diags.Errors())
	}

	response := &resource.ReadResponse{State: tfsdk.State{Schema: schema}}
	r.Read(ctx, resource.ReadRequest{State: state}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Read: %v", response.Diagnostics.Errors())
	}

	var read databaseResourceModel
	if diags := response.State.Get(ctx, &read); diags.HasError() {
		t.Fatalf("reading refreshed state: %v", diags.Errors())
	}
	if read.Encrypted.IsNull() {
		t.Fatal("encrypted is null after reading an encrypted database")
	}
	if !read.Encrypted.ValueBool() {
		t.Errorf("encrypted = %v, want true", read.Encrypted.ValueBool())
	}
}

func TestDatabaseImportReadsEncryption(t *testing.T) {
	t.Parallel()

	encrypted := true
	r, schema := newDatabaseTestResource(t, &encrypted)
	ctx := context.Background()

	importState := tfsdk.State{Schema: schema}
	if diags := importState.Set(ctx, databaseResourceModel{
		Zones: types.ListNull(types.StringType),
	}); diags.HasError() {
		t.Fatalf("building the import state: %v", diags.Errors())
	}

	importResponse := &resource.ImportStateResponse{State: importState}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "db-1"}, importResponse)
	if importResponse.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", importResponse.Diagnostics.Errors())
	}

	readResponse := &resource.ReadResponse{State: tfsdk.State{Schema: schema}}
	r.Read(ctx, resource.ReadRequest{State: importResponse.State}, readResponse)
	if readResponse.Diagnostics.HasError() {
		t.Fatalf("Read after import: %v", readResponse.Diagnostics.Errors())
	}

	var imported databaseResourceModel
	if diags := readResponse.State.Get(ctx, &imported); diags.HasError() {
		t.Fatalf("reading imported state: %v", diags.Errors())
	}
	if !imported.Encrypted.ValueBool() {
		t.Errorf("encrypted = %v, want true after import", imported.Encrypted.ValueBool())
	}
}

func TestDatabaseReadKeepsEncryptionWhenAPIOmitsIt(t *testing.T) {
	t.Parallel()

	r, schema := newDatabaseTestResource(t, nil)
	ctx := context.Background()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(ctx, databaseResourceModel{
		ID:               types.StringValue("db-1"),
		ProjectID:        types.StringValue("p-1"),
		AvailabilityZone: types.StringValue(client.DefaultAvailabilityZone),
		Zones:            types.ListNull(types.StringType),
		Encrypted:        types.BoolValue(true),
	}); diags.HasError() {
		t.Fatalf("building the prior state: %v", diags.Errors())
	}

	response := &resource.ReadResponse{State: tfsdk.State{Schema: schema}}
	r.Read(ctx, resource.ReadRequest{State: state}, response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Read: %v", response.Diagnostics.Errors())
	}

	var read databaseResourceModel
	if diags := response.State.Get(ctx, &read); diags.HasError() {
		t.Fatalf("reading refreshed state: %v", diags.Errors())
	}
	if read.Encrypted.IsNull() || !read.Encrypted.ValueBool() {
		t.Errorf("encrypted = %v, want the prior value true preserved", read.Encrypted)
	}
}
