// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/nobus-cloud/terraform-provider-nobus/internal/client"
)

// fakeNobus is a minimal in-memory implementation of the Nobus API operations
// used by the provider. It lets the resource lifecycle be exercised end to end
// without real infrastructure.
type fakeNobus struct {
	mu sync.Mutex

	networks  map[string]map[string]any
	volumes   map[string]map[string]any
	instances map[string]map[string]any
	// requests records the sequence of HTTP calls made by the provider.
	requests []string
	// volumeStatus is returned while a volume is provisioning.
	volumeStatus string
	// extra carries the additional routes of a specialised fake API, keyed by
	// "METHOD /path".
	extra map[string]http.HandlerFunc
}

// route dispatches a request to the specialised route when one is registered
// for its method and path, and otherwise to the base handler.
func (f *fakeNobus) route(base http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		handler, ok := f.extra[r.Method+" "+r.URL.Path]
		f.mu.Unlock()

		if ok {
			f.record(r)
			handler(w, r)
			return
		}
		base.ServeHTTP(w, r)
	})
}

func newFakeNobus() *fakeNobus {
	return &fakeNobus{
		networks:     map[string]map[string]any{},
		volumes:      map[string]map[string]any{},
		volumeStatus: "available",
	}
}

func (f *fakeNobus) record(r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
}

func (f *fakeNobus) callCount() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.requests...)
}

func (f *fakeNobus) write(w http.ResponseWriter, status bool, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "data": data})
}

func (f *fakeNobus) handler() http.Handler {
	mux := http.NewServeMux()

	// The provider must send the API key as a bearer token on every call.
	auth := func(r *http.Request) bool {
		return r.Header.Get("Authorization") == "Bearer test-key"
	}

	mux.HandleFunc("/api/v3/network/", func(w http.ResponseWriter, r *http.Request) {
		f.record(r)
		if !auth(r) {
			f.write(w, false, map[string]any{"message": "unauthorized"})
			return
		}

		switch r.Method {
		case http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			id := "n-1"
			f.mu.Lock()
			f.networks[id] = map[string]any{
				"id":                    id,
				"name":                  body["network_name"],
				"project_id":            body["project_id"],
				"status":                "ACTIVE",
				"admin_state_up":        true,
				"shared":                false,
				"mtu":                   1500,
				"router_external":       false,
				"port_security_enabled": true,
				"tags":                  []any{},
				"subnets":               []any{"s-1"},
			}
			f.mu.Unlock()
			f.write(w, true, map[string]any{"network_id": id, "subnet_id": "s-1", "router_id": "r-1"})

		case http.MethodGet:
			f.mu.Lock()
			network, ok := f.networks[r.URL.Query().Get("network_id")]
			f.mu.Unlock()
			if !ok {
				f.write(w, false, map[string]any{"message": "network not found"})
				return
			}
			f.write(w, true, map[string]any{"network": network})

		case http.MethodPut:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			networkID, _ := body["network_id"].(string)
			networkFields, _ := body["network"].(map[string]any)
			f.mu.Lock()
			network, ok := f.networks[networkID]
			if ok {
				for key, value := range networkFields {
					if value != nil {
						network[key] = value
					}
				}
			}
			f.mu.Unlock()
			if !ok {
				f.write(w, false, map[string]any{"message": "network not found"})
				return
			}
			f.write(w, true, map[string]any{"network": network})

		case http.MethodDelete:
			f.mu.Lock()
			delete(f.networks, r.URL.Query().Get("network_id"))
			f.mu.Unlock()
			f.write(w, true, nil)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return mux
}

// newTestProviderConfig returns a provider configuration wired to the fake API.
func newTestProviderConfig(t *testing.T, server *httptest.Server) *providerConfig {
	t.Helper()

	apiClient, err := client.New(client.Config{
		APIKey:     "test-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("client.New: %s", err)
	}

	return &providerConfig{
		Client:           apiClient,
		ProjectID:        "p-1",
		AvailabilityZone: client.DefaultAvailabilityZone,
	}
}

// newNetworkResource returns a configured network resource and its schema.
func newNetworkResource(t *testing.T, config *providerConfig) (*networkResource, schema.Schema) {
	t.Helper()

	ctx := context.Background()

	r := &networkResource{config: config}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics.Errors())
	}

	// Wrap the fwschema so it satisfies the tfsdk.Schema the CRUD calls expect.
	return r, schemaResp.Schema
}

// TestNetworkResourceLifecycle drives create, read, update and delete of the
// network resource against a fake Nobus API.
func TestNetworkResourceLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := newFakeNobus()
	server := httptest.NewServer(fake.handler())
	defer server.Close()

	config := newTestProviderConfig(t, server)
	r, schema := newNetworkResource(t, config)

	plan := networkResourceModel{
		ID:                types.StringNull(),
		ProjectID:         types.StringNull(),
		AvailabilityZone:  types.StringNull(),
		Name:              types.StringValue("production-network"),
		SubnetCIDR:        types.StringValue("10.20.0.0/24"),
		RouterName:        types.StringValue("production-router"),
		ExternalNetworkID: types.StringNull(),
		DNSNameservers:    types.ListNull(types.StringType),
		EnableSNAT:        types.BoolValue(true),
		Description:       types.StringValue("primary"),
		AdminStateUp:      types.BoolValue(true),
		Shared:            types.BoolNull(),
		MTU:               types.Int64Value(1500),
		RouterExternal:    types.BoolNull(),
		PortSecurity:      types.BoolNull(),
		Tags:              types.SetNull(types.StringType),
		Status:            types.StringNull(),
		SubnetID:          types.StringNull(),
		RouterID:          types.StringNull(),
		SubnetIDs:         types.ListNull(types.StringType),
	}

	// --- Create ---
	createPlan := tfsdk.Plan{Schema: schema}
	if diags := createPlan.Set(ctx, plan); diags.HasError() {
		t.Fatalf("building the create plan: %v", diags.Errors())
	}

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
	r.Create(ctx, resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", createResp.Diagnostics.Errors())
	}

	var created networkResourceModel
	if diags := createResp.State.Get(ctx, &created); diags.HasError() {
		t.Fatalf("reading the created state: %v", diags.Errors())
	}

	if created.ID.ValueString() != "n-1" {
		t.Errorf("id = %q, want n-1", created.ID.ValueString())
	}
	if created.SubnetID.ValueString() != "s-1" {
		t.Errorf("subnet_id = %q, want s-1", created.SubnetID.ValueString())
	}
	if created.RouterID.ValueString() != "r-1" {
		t.Errorf("router_id = %q, want r-1", created.RouterID.ValueString())
	}
	// The provider default project must be recorded in state.
	if created.ProjectID.ValueString() != "p-1" {
		t.Errorf("project_id = %q, want p-1", created.ProjectID.ValueString())
	}
	if created.AvailabilityZone.ValueString() != client.DefaultAvailabilityZone {
		t.Errorf("availability_zone = %q, want %q", created.AvailabilityZone.ValueString(), client.DefaultAvailabilityZone)
	}

	// --- Read ---
	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schema}}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics.Errors())
	}

	var read networkResourceModel
	if diags := readResp.State.Get(ctx, &read); diags.HasError() {
		t.Fatalf("reading the refreshed state: %v", diags.Errors())
	}
	if read.Name.ValueString() != "production-network" {
		t.Errorf("name = %q, want production-network", read.Name.ValueString())
	}
	if read.Status.ValueString() != "ACTIVE" {
		t.Errorf("status = %q, want ACTIVE", read.Status.ValueString())
	}

	// --- Update (rename plus a mutable attribute) ---
	read.Description = types.StringValue("renamed-network")
	read.MTU = types.Int64Value(9000)

	updatePlan := tfsdk.Plan{Schema: schema}
	if diags := updatePlan.Set(ctx, read); diags.HasError() {
		t.Fatalf("building the update plan: %v", diags.Errors())
	}

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
	r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: readResp.State}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", updateResp.Diagnostics.Errors())
	}

	var updated networkResourceModel
	if diags := updateResp.State.Get(ctx, &updated); diags.HasError() {
		t.Fatalf("reading the updated state: %v", diags.Errors())
	}
	if updated.Description.ValueString() != "renamed-network" {
		t.Errorf("description = %q, want renamed-network", updated.Description.ValueString())
	}
	if updated.MTU.ValueInt64() != 9000 {
		t.Errorf("mtu = %d, want 9000", updated.MTU.ValueInt64())
	}

	// The immutable configuration values must survive the update.
	if updated.SubnetCIDR.ValueString() != "10.20.0.0/24" {
		t.Errorf("subnet_cidr was lost during update: %q", updated.SubnetCIDR.ValueString())
	}

	// --- Delete ---
	deleteResp := &resource.DeleteResponse{State: tfsdk.State{Schema: schema}}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", deleteResp.Diagnostics.Errors())
	}

	// A second delete must succeed because deletion is idempotent.
	deleteAgain := &resource.DeleteResponse{State: tfsdk.State{Schema: schema}}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, deleteAgain)
	if deleteAgain.Diagnostics.HasError() {
		t.Fatalf("repeated Delete should be idempotent, got: %v", deleteAgain.Diagnostics.Errors())
	}

	calls := fake.callCount()
	if len(calls) == 0 {
		t.Fatal("no API calls were recorded")
	}
	if !strings.Contains(strings.Join(calls, ","), "POST /api/v3/network/") {
		t.Errorf("expected a create call, got %v", calls)
	}
	if !strings.Contains(strings.Join(calls, ","), "DELETE /api/v3/network/") {
		t.Errorf("expected a delete call, got %v", calls)
	}
}

// TestNetworkReadRemovesDeletedResource verifies that a network deleted outside
// Terraform is dropped from state instead of producing an error.
func TestNetworkReadRemovesDeletedResource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := newFakeNobus()
	server := httptest.NewServer(fake.handler())
	defer server.Close()

	config := newTestProviderConfig(t, server)
	r, schema := newNetworkResource(t, config)

	// Build a state for a network that no longer exists in the API.
	stale := networkResourceModel{
		ID:                types.StringValue("n-missing"),
		ProjectID:         types.StringValue("p-1"),
		AvailabilityZone:  types.StringValue(client.DefaultAvailabilityZone),
		Name:              types.StringValue("gone"),
		SubnetCIDR:        types.StringValue("10.20.0.0/24"),
		RouterName:        types.StringValue("router"),
		ExternalNetworkID: types.StringNull(),
		DNSNameservers:    types.ListNull(types.StringType),
		EnableSNAT:        types.BoolValue(true),
		Description:       types.StringNull(),
		AdminStateUp:      types.BoolNull(),
		Shared:            types.BoolNull(),
		MTU:               types.Int64Null(),
		RouterExternal:    types.BoolNull(),
		PortSecurity:      types.BoolNull(),
		Tags:              types.SetNull(types.StringType),
		Status:            types.StringNull(),
		SubnetID:          types.StringNull(),
		RouterID:          types.StringNull(),
		SubnetIDs:         types.ListNull(types.StringType),
	}

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(ctx, stale); diags.HasError() {
		t.Fatalf("building the state: %v", diags.Errors())
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schema, Raw: state.Raw}}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read of a deleted network should not error: %v", readResp.Diagnostics.Errors())
	}

	// RemoveResource leaves a null raw value behind.
	if !readResp.State.Raw.IsNull() {
		t.Error("expected the resource to be removed from state")
	}
}

// TestProviderConfigureRequiresAPIKey verifies the provider reports a clear
// diagnostic when no credentials are available.
func TestProviderConfigureRequiresAPIKey(t *testing.T) {
	t.Setenv("NOBUS_API_KEY", "")

	ctx := context.Background()

	p := &NobusProvider{version: "test"}

	schemaResp := &provider.SchemaResponse{}
	p.Schema(ctx, provider.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("provider schema diagnostics: %v", schemaResp.Diagnostics.Errors())
	}

	// An entirely empty provider configuration, with the API key left unset.
	var empty NobusProviderModel
	empty.APIKey = types.StringNull()
	empty.Endpoint = types.StringNull()
	empty.ProjectID = types.StringNull()
	empty.AvailabilityZone = types.StringNull()

	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, empty); diags.HasError() {
		t.Fatalf("building the config: %v", diags.Errors())
	}
	config := tfsdk.Config{Schema: schemaResp.Schema, Raw: state.Raw}

	resp := &provider.ConfigureResponse{}
	p.Configure(ctx, provider.ConfigureRequest{Config: config}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a diagnostic about the missing API key")
	}

	found := false
	for _, d := range resp.Diagnostics.Errors() {
		if strings.Contains(strings.ToLower(d.Summary()), "api key") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an API key diagnostic, got: %v", resp.Diagnostics.Errors())
	}
}

// fakeVolumeNobus extends the fake API with the volume and instance operations.
func fakeVolumeNobus() *fakeNobus {
	fake := newFakeNobus()

	fake.extra = map[string]http.HandlerFunc{}
	fake.extra[http.MethodPost+" /api/v3/volume/"] = func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			fake.mu.Lock()
			fake.volumes["v-1"] = map[string]any{
				"id":         "v-1",
				"name":       body["name"],
				"size":       body["size"],
				"status":     fake.volumeStatus,
				"project_id": body["project_id"],
				"metadata":   body["metadata"],
			}
			fake.mu.Unlock()
			fake.write(w, true, fake.volumes["v-1"])

		case http.MethodGet:
			fake.mu.Lock()
			volume, ok := fake.volumes[r.URL.Query().Get("volume_id")]
			fake.mu.Unlock()
			if !ok {
				fake.write(w, false, map[string]any{"message": "volume not found"})
				return
			}
			fake.write(w, true, volume)

		case http.MethodPut:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			fake.mu.Lock()
			if volume, ok := fake.volumes["v-1"]; ok {
				if body["name"] != nil {
					volume["name"] = body["name"]
				}
			}
			fake.mu.Unlock()
			fake.write(w, true, nil)

		case http.MethodDelete:
			fake.mu.Lock()
			delete(fake.volumes, r.URL.Query().Get("volume_id"))
			fake.mu.Unlock()
			fake.write(w, true, nil)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
	fake.extra[http.MethodGet+" /api/v3/volume/"] = fake.extra[http.MethodPost+" /api/v3/volume/"]
	fake.extra[http.MethodPut+" /api/v3/volume/"] = fake.extra[http.MethodPost+" /api/v3/volume/"]
	fake.extra[http.MethodDelete+" /api/v3/volume/"] = fake.extra[http.MethodPost+" /api/v3/volume/"]

	fake.extra[http.MethodPost+" /api/v3/volume/extend"] = func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		fake.mu.Lock()
		if volume, ok := fake.volumes[body["volume_id"].(string)]; ok {
			volume["size"] = body["new_size"]
		}
		fake.mu.Unlock()
		fake.write(w, true, nil)
	}

	return fake
}

// serve starts the fake API, merging the base routes with any specialised ones.
func (f *fakeNobus) serve(t *testing.T) *httptest.Server {
	t.Helper()

	primary := f.handler()

	// The fake routes on an explicit path table, so both the base operations
	// and any specialised routes are served by a single handler.
	return httptest.NewServer(f.route(primary))
}

// TestVolumeResourceLifecycle drives create, read, grow, rename and delete.
func TestVolumeResourceLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := fakeVolumeNobus()
	server := fake.serve(t)
	defer server.Close()

	config := newTestProviderConfig(t, server)
	r := &volumeResource{config: config}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics.Errors())
	}
	schema := schemaResp.Schema

	plan := volumeResourceModel{
		ID:               types.StringNull(),
		ProjectID:        types.StringNull(),
		AvailabilityZone: types.StringNull(),
		Name:             types.StringValue("production-data"),
		Description:      types.StringNull(),
		Size:             types.Int64Value(10),
		VolumeType:       types.StringNull(),
		SnapshotID:       types.StringNull(),
		SourceVolumeID:   types.StringNull(),
		ImageRef:         types.StringNull(),
		Metadata:         mustMap(t, map[string]string{"env": "prod"}),
		Status:           types.StringNull(),
		Bootable:         types.BoolNull(),
		Encrypted:        types.BoolNull(),
		CreatedAt:        types.StringNull(),
		UpdatedAt:        types.StringNull(),
		AttachmentIDs:    types.ListNull(types.StringType),
		ServerIDs:        types.ListNull(types.StringType),
	}

	createPlan := tfsdk.Plan{Schema: schema}
	if diags := createPlan.Set(ctx, plan); diags.HasError() {
		t.Fatalf("building the create plan: %v", diags.Errors())
	}

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
	r.Create(ctx, resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", createResp.Diagnostics.Errors())
	}

	var created volumeResourceModel
	if diags := createResp.State.Get(ctx, &created); diags.HasError() {
		t.Fatalf("reading the created state: %v", diags.Errors())
	}
	if created.ID.ValueString() != "v-1" {
		t.Errorf("id = %q, want v-1", created.ID.ValueString())
	}
	if created.Status.ValueString() != "available" {
		t.Errorf("status = %q, want available", created.Status.ValueString())
	}
	if created.ProjectID.ValueString() != "p-1" {
		t.Errorf("project_id = %q, want the provider default p-1", created.ProjectID.ValueString())
	}

	// --- Grow the volume in place ---
	grown := created
	grown.Size = types.Int64Value(25)

	growPlan := tfsdk.Plan{Schema: schema}
	if diags := growPlan.Set(ctx, grown); diags.HasError() {
		t.Fatalf("building the grow plan: %v", diags.Errors())
	}

	growResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
	r.Update(ctx, resource.UpdateRequest{Plan: growPlan, State: createResp.State}, growResp)
	if growResp.Diagnostics.HasError() {
		t.Fatalf("Update (grow): %v", growResp.Diagnostics.Errors())
	}

	var expanded volumeResourceModel
	if diags := growResp.State.Get(ctx, &expanded); diags.HasError() {
		t.Fatalf("reading the grown state: %v", diags.Errors())
	}
	if expanded.Size.ValueInt64() != 25 {
		t.Errorf("size = %d, want 25", expanded.Size.ValueInt64())
	}

	// --- Rename ---
	renamed := expanded
	renamed.Name = types.StringValue("production-data-renamed")

	renamePlan := tfsdk.Plan{Schema: schema}
	if diags := renamePlan.Set(ctx, renamed); diags.HasError() {
		t.Fatalf("building the rename plan: %v", diags.Errors())
	}

	renameResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
	r.Update(ctx, resource.UpdateRequest{Plan: renamePlan, State: growResp.State}, renameResp)
	if renameResp.Diagnostics.HasError() {
		t.Fatalf("Update (rename): %v", renameResp.Diagnostics.Errors())
	}

	var afterRename volumeResourceModel
	if diags := renameResp.State.Get(ctx, &afterRename); diags.HasError() {
		t.Fatalf("reading the renamed state: %v", diags.Errors())
	}
	if afterRename.Name.ValueString() != "production-data-renamed" {
		t.Errorf("name = %q, want production-data-renamed", afterRename.Name.ValueString())
	}

	// --- Shrinking must fail with an explanatory diagnostic ---
	shrunk := afterRename
	shrunk.Size = types.Int64Value(5)

	shrinkPlan := tfsdk.Plan{Schema: schema}
	if diags := shrinkPlan.Set(ctx, shrunk); diags.HasError() {
		t.Fatalf("building the shrink plan: %v", diags.Errors())
	}

	shrinkResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
	r.Update(ctx, resource.UpdateRequest{Plan: shrinkPlan, State: renameResp.State}, shrinkResp)
	if !shrinkResp.Diagnostics.HasError() {
		t.Fatal("expected an error when shrinking a volume")
	}
	if !strings.Contains(shrinkResp.Diagnostics.Errors()[0].Summary(), "shrink") {
		t.Errorf("unexpected diagnostic: %s", shrinkResp.Diagnostics.Errors()[0].Summary())
	}

	// --- Delete ---
	deleteResp := &resource.DeleteResponse{State: tfsdk.State{Schema: schema}}
	r.Delete(ctx, resource.DeleteRequest{State: renameResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", deleteResp.Diagnostics.Errors())
	}

	// Deleting again must be idempotent.
	again := &resource.DeleteResponse{State: tfsdk.State{Schema: schema}}
	r.Delete(ctx, resource.DeleteRequest{State: renameResp.State}, again)
	if again.Diagnostics.HasError() {
		t.Fatalf("repeated Delete should be idempotent: %v", again.Diagnostics.Errors())
	}
}

// mustMap builds a Terraform map value or fails the test.
func mustMap(t *testing.T, value map[string]string) types.Map {
	t.Helper()

	result, diags := types.MapValueFrom(context.Background(), types.StringType, value)
	if diags.HasError() {
		t.Fatalf("building a map value: %v", diags.Errors())
	}
	return result
}

// fakeInstanceNobus extends the fake API with the instance operations.
func fakeInstanceNobus() *fakeNobus {
	fake := newFakeNobus()
	fake.instances = map[string]map[string]any{}
	fake.extra = map[string]http.HandlerFunc{}

	handle := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			fake.mu.Lock()
			fake.instances["i-1"] = map[string]any{
				"id":          "i-1",
				"name":        body["name"],
				"status":      "ACTIVE",
				"key_name":    body["key_name"],
				"project_id":  body["project_id"],
				"addresses":   map[string]any{"private": []any{map[string]any{"addr": "10.98.0.5"}}},
				"flavor":      map[string]any{"id": "f-1", "name": "small"},
				"image":       map[string]any{"id": "img-1", "name": "ubuntu-22-04"},
				"volumes":     []any{map[string]any{"id": "v-1"}},
				"description": nil,
			}
			fake.mu.Unlock()
			fake.write(w, true, fake.instances["i-1"])

		case http.MethodGet:
			fake.mu.Lock()
			instance, ok := fake.instances[r.URL.Query().Get("server_id")]
			fake.mu.Unlock()
			if !ok {
				fake.write(w, false, map[string]any{"message": "instance not found"})
				return
			}
			fake.write(w, true, instance)

		case http.MethodDelete:
			fake.mu.Lock()
			delete(fake.instances, r.URL.Query().Get("server_id"))
			fake.mu.Unlock()
			fake.write(w, true, "deleted")

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}

	fake.extra[http.MethodPost+" /api/v3/instance"] = handle
	fake.extra[http.MethodGet+" /api/v3/instance"] = handle
	fake.extra[http.MethodDelete+" /api/v3/instance"] = handle

	// The update endpoint only needs to accept the documented members.
	fake.extra[http.MethodPost+" /api/v3/instance/update-server"] = func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		fake.mu.Lock()
		if instance, ok := fake.instances[body["server_id"].(string)]; ok && body["name"] != nil {
			instance["name"] = body["name"]
		}
		fake.mu.Unlock()
		fake.write(w, true, "ok")
	}

	return fake
}

// TestInstanceResourceLifecycle drives create, read, rename and delete of an
// instance, including the network dependency in the create payload.
func TestInstanceResourceLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := fakeInstanceNobus()
	server := fake.serve(t)
	defer server.Close()

	config := newTestProviderConfig(t, server)
	r := &instanceResource{config: config}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics.Errors())
	}
	schema := schemaResp.Schema

	plan := instanceResourceModel{
		ID:                  types.StringNull(),
		ProjectID:           types.StringNull(),
		AvailabilityZone:    types.StringNull(),
		Name:                types.StringValue("web-01"),
		Description:         types.StringValue("Public web server"),
		Hostname:            types.StringNull(),
		Image:               types.StringValue("ubuntu-22-04"),
		Flavor:              types.StringValue("small"),
		KeyName:             types.StringValue("operations"),
		NetworkID:           types.StringValue("n-1"),
		VolumeSize:          types.Int64Value(20),
		SecurityGroups:      mustList(t, []string{"default"}),
		AdminPass:           types.StringNull(),
		UserData:            types.StringNull(),
		DeleteVolume:        types.BoolValue(true),
		InstanceReadyStates: types.ListNull(types.StringType),
		Status:              types.StringNull(),
		Addresses:           types.StringNull(),
		VolumeIDs:           types.ListNull(types.StringType),
		FlavorID:            types.StringNull(),
		FlavorName:          types.StringNull(),
		ImageID:             types.StringNull(),
		ImageName:           types.StringNull(),
		CreatedAt:           types.StringNull(),
		UpdatedAt:           types.StringNull(),
	}

	createPlan := tfsdk.Plan{Schema: schema}
	if diags := createPlan.Set(ctx, plan); diags.HasError() {
		t.Fatalf("building the create plan: %v", diags.Errors())
	}

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
	r.Create(ctx, resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", createResp.Diagnostics.Errors())
	}

	var created instanceResourceModel
	if diags := createResp.State.Get(ctx, &created); diags.HasError() {
		t.Fatalf("reading the created state: %v", diags.Errors())
	}

	if created.ID.ValueString() != "i-1" {
		t.Errorf("id = %q, want i-1", created.ID.ValueString())
	}
	if created.Status.ValueString() != "ACTIVE" {
		t.Errorf("status = %q, want ACTIVE", created.Status.ValueString())
	}
	if created.ProjectID.ValueString() != "p-1" {
		t.Errorf("project_id = %q, want the provider default p-1", created.ProjectID.ValueString())
	}
	if created.FlavorID.ValueString() != "f-1" {
		t.Errorf("flavor_id = %q, want f-1", created.FlavorID.ValueString())
	}
	if !strings.Contains(created.Addresses.ValueString(), "10.98.0.5") {
		t.Errorf("addresses = %q, want it to contain 10.98.0.5", created.Addresses.ValueString())
	}
	// The configuration-only attributes must be preserved verbatim.
	if created.NetworkID.ValueString() != "n-1" {
		t.Errorf("network_id = %q, want n-1", created.NetworkID.ValueString())
	}
	if created.Image.ValueString() != "ubuntu-22-04" || created.Flavor.ValueString() != "small" {
		t.Errorf("image/flavor were lost: %q / %q", created.Image.ValueString(), created.Flavor.ValueString())
	}

	// --- Read ---
	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schema}}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics.Errors())
	}

	// --- Rename ---
	renamed := created
	renamed.Name = types.StringValue("web-02")

	updatePlan := tfsdk.Plan{Schema: schema}
	if diags := updatePlan.Set(ctx, renamed); diags.HasError() {
		t.Fatalf("building the update plan: %v", diags.Errors())
	}

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schema}}
	r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: readResp.State}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", updateResp.Diagnostics.Errors())
	}

	var afterRename instanceResourceModel
	if diags := updateResp.State.Get(ctx, &afterRename); diags.HasError() {
		t.Fatalf("reading the renamed state: %v", diags.Errors())
	}
	if afterRename.Name.ValueString() != "web-02" {
		t.Errorf("name = %q, want web-02", afterRename.Name.ValueString())
	}
	if afterRename.NetworkID.ValueString() != "n-1" {
		t.Errorf("network_id must survive an update, got %q", afterRename.NetworkID.ValueString())
	}

	// --- Delete ---
	deleteResp := &resource.DeleteResponse{State: tfsdk.State{Schema: schema}}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", deleteResp.Diagnostics.Errors())
	}

	again := &resource.DeleteResponse{State: tfsdk.State{Schema: schema}}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, again)
	if again.Diagnostics.HasError() {
		t.Fatalf("repeated Delete should be idempotent: %v", again.Diagnostics.Errors())
	}
}

// mustList builds a Terraform list value or fails the test.
func mustList(t *testing.T, value []string) types.List {
	t.Helper()

	result, diags := types.ListValueFrom(context.Background(), types.StringType, value)
	if diags.HasError() {
		t.Fatalf("building a list value: %v", diags.Errors())
	}
	return result
}

// TestVolumeResourceCreate verifies a volume can be created with the minimal
// supported configuration.
func TestVolumeResourceCreate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := fakeVolumeNobus()
	server := fake.serve(t)
	defer server.Close()

	config := newTestProviderConfig(t, server)
	r := &volumeResource{config: config}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics.Errors())
	}

	model := volumeResourceModel{
		ID:               types.StringNull(),
		ProjectID:        types.StringNull(),
		AvailabilityZone: types.StringNull(),
		Name:             types.StringValue("data"),
		Description:      types.StringNull(),
		Size:             types.Int64Value(10),
		VolumeType:       types.StringNull(),
		SnapshotID:       types.StringNull(),
		SourceVolumeID:   types.StringNull(),
		ImageRef:         types.StringNull(),
		Metadata:         types.MapNull(types.StringType),
		Status:           types.StringNull(),
		Bootable:         types.BoolNull(),
		Encrypted:        types.BoolNull(),
		CreatedAt:        types.StringNull(),
		UpdatedAt:        types.StringNull(),
		AttachmentIDs:    types.ListNull(types.StringType),
		ServerIDs:        types.ListNull(types.StringType),
	}

	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := plan.Set(ctx, model); diags.HasError() {
		t.Fatalf("building the plan: %v", diags.Errors())
	}

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create should succeed: %v", resp.Diagnostics.Errors())
	}

	var created volumeResourceModel
	if diags := resp.State.Get(ctx, &created); diags.HasError() {
		t.Fatalf("reading the created state: %v", diags.Errors())
	}
	if created.ID.ValueString() == "" {
		t.Error("expected the volume to be created")
	}
}

// TestVolumeComputedAttributesAreKnownAfterApply reproduces a second bug found
// by a real `terraform apply`: Terraform requires every value to be known after
// apply, so an Optional+Computed attribute must never be left unknown.
func TestInstanceComputedValuesAreKnownAfterCreate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fake := fakeInstanceNobus()
	server := fake.serve(t)
	defer server.Close()

	config := newTestProviderConfig(t, server)
	r := &instanceResource{config: config}

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics.Errors())
	}
	schema := schemaResp.Schema

	// security_groups is intentionally left null, as it would be when the
	// configuration omits it.
	plan := instanceResourceModel{
		ID:                  types.StringNull(),
		ProjectID:           types.StringNull(),
		AvailabilityZone:    types.StringNull(),
		Name:                types.StringValue("api-01"),
		Description:         types.StringNull(),
		Hostname:            types.StringNull(),
		Image:               types.StringValue("ubuntu-22-04"),
		Flavor:              types.StringValue("medium"),
		KeyName:             types.StringValue("operations"),
		NetworkID:           types.StringValue("n-1"),
		VolumeSize:          types.Int64Null(),
		SecurityGroups:      types.ListNull(types.StringType),
		AdminPass:           types.StringNull(),
		UserData:            types.StringNull(),
		DeleteVolume:        types.BoolNull(),
		InstanceReadyStates: types.ListNull(types.StringType),
		Status:              types.StringNull(),
		Addresses:           types.StringNull(),
		VolumeIDs:           types.ListNull(types.StringType),
		FlavorID:            types.StringNull(),
		FlavorName:          types.StringNull(),
		ImageID:             types.StringNull(),
		ImageName:           types.StringNull(),
		CreatedAt:           types.StringNull(),
		UpdatedAt:           types.StringNull(),
	}

	createPlan := tfsdk.Plan{Schema: schema}
	if diags := createPlan.Set(ctx, plan); diags.HasError() {
		t.Fatalf("building the create plan: %v", diags.Errors())
	}

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schema}}
	r.Create(ctx, resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", createResp.Diagnostics.Errors())
	}

	var created instanceResourceModel
	if diags := createResp.State.Get(ctx, &created); diags.HasError() {
		t.Fatalf("reading the created state: %v", diags.Errors())
	}

	// Every value must be known: no unknown may survive the apply.
	assertNoUnknown(t, "nobus_instance", createResp.State.Raw)

	// volume_size was omitted, so it stays null rather than becoming unknown.
	if !created.VolumeSize.IsNull() {
		t.Errorf("volume_size should stay null when omitted, got %v", created.VolumeSize)
	}
}

// assertNoUnknown walks a state value and fails on any unknown attribute.
//
// Terraform rejects an apply that leaves any value unknown, so this mirrors the
// check Terraform itself performs.
func assertNoUnknown(t *testing.T, address string, value tftypes.Value) {
	t.Helper()

	var unknowns []string

	err := tftypes.Walk(value, func(path *tftypes.AttributePath, v tftypes.Value) (bool, error) {
		if !v.IsKnown() {
			unknowns = append(unknowns, path.String())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("walking the state: %s", err)
	}

	for _, path := range unknowns {
		t.Errorf("%s%s is unknown after apply", address, path)
	}
}
