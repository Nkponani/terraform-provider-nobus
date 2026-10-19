// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestCreateVolumeRequestMapping verifies the volume create payload against
// VolumeCreateRequestSchema, including the members the API requires.
func TestCreateVolumeRequestMapping(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"id":"v-1","name":"data","size":100,"status":"available"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	name := "production-data"
	encrypted := true
	volume, err := c.CreateVolume(context.Background(), CreateVolumeRequest{
		ProjectID:        "p-1",
		AvailabilityZone: "nobus-wa-az1",
		Size:             100,
		Name:             &name,
		Metadata:         map[string]string{"env": "prod"},
		Encrypted:        &encrypted,
	})
	if err != nil {
		t.Fatalf("CreateVolume: %s", err)
	}

	if volume.ID != "v-1" || volume.Size != 100 {
		t.Errorf("unexpected volume: %+v", volume)
	}
	if gotBody["size"] != float64(100) {
		t.Errorf("size = %v, want 100", gotBody["size"])
	}
	if gotBody["name"] != "production-data" {
		t.Errorf("name = %v, want production-data", gotBody["name"])
	}
	if gotBody["project_id"] != "p-1" {
		t.Errorf("project_id = %v, want p-1", gotBody["project_id"])
	}
	if _, ok := gotBody["multiattach"]; ok {
		t.Error("request body must not contain multiattach")
	}
	if gotBody["encrypted"] != true {
		t.Errorf("encrypted = %v, want true", gotBody["encrypted"])
	}

	metadata, ok := gotBody["metadata"].(map[string]any)
	if !ok || metadata["env"] != "prod" {
		t.Errorf("metadata = %v, want env=prod", gotBody["metadata"])
	}
}

// TestCreateInstanceRequiredFields verifies that the instance create payload
// carries every member the API requires.
func TestCreateInstanceRequiredFields(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"id":"i-1","name":"web-01","status":"ACTIVE"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	networkID := "n-1"
	instance, err := c.CreateInstance(context.Background(), CreateInstanceRequest{
		ProjectID:        "p-1",
		AvailabilityZone: "nobus-wa-az1",
		Name:             "web-01",
		Image:            "ubuntu-22-04",
		Flavor:           "small",
		KeyName:          "ops",
		NetworkID:        &networkID,
		SecurityGroups:   []string{"default"},
	})
	if err != nil {
		t.Fatalf("CreateInstance: %s", err)
	}

	if instance.ID != "i-1" {
		t.Errorf("id = %q, want i-1", instance.ID)
	}

	// These five members are listed as required by CreateInstanceRequestSchema.
	for _, key := range []string{"project_id", "availability_zone", "name", "image", "flavor", "key_name"} {
		if _, ok := gotBody[key]; !ok {
			t.Errorf("request body is missing the required member %q", key)
		}
	}
	if gotBody["network_id"] != "n-1" {
		t.Errorf("network_id = %v, want n-1", gotBody["network_id"])
	}
	if gotBody["nics"] != "auto" {
		t.Errorf("nics = %v, want auto", gotBody["nics"])
	}

	// Optional members that were not configured must be omitted entirely.
	for _, key := range []string{"admin_pass", "userdata", "volume_size"} {
		if _, ok := gotBody[key]; ok {
			t.Errorf("request body must omit the unset member %q", key)
		}
	}
}

// TestDeleteInstanceCarriesDeleteVolume verifies the documented delete_volume
// query parameter.
func TestDeleteInstanceCarriesDeleteVolume(t *testing.T) {
	t.Parallel()

	for _, deleteVolume := range []bool{true, false} {
		var got string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got = r.URL.Query().Get("delete_volume")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":true,"data":"deleted"}`))
		}))

		c := newTestClient(t, server)
		if err := c.DeleteInstance(context.Background(), "p-1", "az-1", "i-1", deleteVolume); err != nil {
			t.Fatalf("DeleteInstance: %s", err)
		}
		server.Close()

		want := "false"
		if deleteVolume {
			want = "true"
		}
		if got != want {
			t.Errorf("delete_volume = %q, want %q", got, want)
		}
	}
}

// TestUpdateVolumePayload verifies that only the mutable members are sent.
func TestUpdateVolumePayload(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":null}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	name := "renamed"
	if err := c.UpdateVolume(context.Background(), UpdateVolumeRequest{
		ProjectID:        "p-1",
		AvailabilityZone: "az-1",
		Name:             &name,
	}); err != nil {
		t.Fatalf("UpdateVolume: %s", err)
	}

	if gotBody["name"] != "renamed" {
		t.Errorf("name = %v, want renamed", gotBody["name"])
	}
	// size is not part of VolumeUpdateRequestSchema; growth uses /volume/extend.
	if _, ok := gotBody["size"]; ok {
		t.Error("the update model must not carry size")
	}
}

// TestWaitForVolumeReadyPolls verifies that the waiter polls the API status
// rather than assuming success, and succeeds once the volume is available.
func TestWaitForVolumeReadyPolls(t *testing.T) {
	t.Parallel()

	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		status := "creating"
		if n >= 3 {
			status = "available"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"id":"v-1","size":10,"status":"` + status + `"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	volume, err := c.WaitForVolumeReady(context.Background(), "p-1", "az-1", "v-1", nil, time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForVolumeReady: %s", err)
	}

	if volume.Status == nil || *volume.Status != VolumeStateAvailable {
		t.Errorf("status = %v, want available", volume.Status)
	}
	if atomic.LoadInt32(&calls) < 3 {
		t.Errorf("expected the waiter to poll at least 3 times, got %d", calls)
	}
}

// TestWaitForVolumeReadyFailsOnErrorState verifies that a terminal error state
// aborts the wait instead of timing out.
func TestWaitForVolumeReadyFailsOnErrorState(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"id":"v-1","size":10,"status":"error"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, err := c.WaitForVolumeReady(context.Background(), "p-1", "az-1", "v-1", nil, time.Millisecond)
	if err == nil {
		t.Fatal("expected an error when the volume enters the error state")
	}
	if !strings.Contains(err.Error(), "error") {
		t.Errorf("unexpected error: %s", err)
	}
}

// TestWaitForInstanceReady verifies the instance waiter honours the ready state
// list.
func TestWaitForInstanceReady(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"id":"i-1","name":"web","status":"BUILDING"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.WaitForInstanceReady(ctx, "p-1", "az-1", "i-1", nil, time.Millisecond)
	if err == nil {
		t.Fatal("expected cancellation because the instance never becomes ACTIVE")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("the error should report cancellation, got: %s", err)
	}
}

// TestIsNotFoundAfterRemoteDelete verifies the read path used to drop a resource
// from state when it disappears outside Terraform.
func TestReadAfterRemoteDelete(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":false,"data":{"message":"volume v-1 not found"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)

	if _, err := c.GetVolume(context.Background(), "p-1", "az-1", "v-1"); !IsNotFound(err) {
		t.Errorf("GetVolume error should be recognised as not found, got: %v", err)
	}
	if _, err := c.GetInstance(context.Background(), "p-1", "az-1", "i-1"); !IsNotFound(err) {
		t.Errorf("GetInstance error should be recognised as not found, got: %v", err)
	}
	if _, err := c.GetNetwork(context.Background(), "p-1", "az-1", "n-1"); !IsNotFound(err) {
		t.Errorf("GetNetwork error should be recognised as not found, got: %v", err)
	}
}

// TestListNetworksPagination verifies the documented limit and marker parameters.
func TestListNetworksPagination(t *testing.T) {
	t.Parallel()

	var gotQuery map[string][]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"networks":[{"id":"n-1"},{"id":"n-2"}]}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	networks, err := c.ListNetworks(context.Background(), "p-1", "az-1", 2, "marker-1")
	if err != nil {
		t.Fatalf("ListNetworks: %s", err)
	}

	if len(networks) != 2 {
		t.Errorf("got %d networks, want 2", len(networks))
	}
	if got := gotQuery["limit"]; len(got) != 1 || got[0] != "2" {
		t.Errorf("limit = %v, want [2]", got)
	}
	if got := gotQuery["marker"]; len(got) != 1 || got[0] != "marker-1" {
		t.Errorf("marker = %v, want [marker-1]", got)
	}
}

// TestGetVolumeDecodesArrayShape verifies that a read by id tolerates the
// array envelope the API returns for freshly created volumes.
func TestGetVolumeDecodesArrayShape(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":[{"id":"v-1","size":1,"status":"available","name":"race-probe"}]}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	volume, err := c.GetVolume(context.Background(), "p-1", "az-1", "v-1")
	if err != nil {
		t.Fatalf("GetVolume: %s", err)
	}
	if volume.ID != "v-1" {
		t.Errorf("id = %q, want v-1", volume.ID)
	}
	if volume.Status == nil || *volume.Status != VolumeStateAvailable {
		t.Errorf("status = %v, want available", volume.Status)
	}
}

// TestGetVolumeDecodesObjectShape verifies the settled object envelope keeps
// decoding after the array-tolerant change.
func TestGetVolumeDecodesObjectShape(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"id":"v-1","size":1,"status":"available"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	volume, err := c.GetVolume(context.Background(), "p-1", "az-1", "v-1")
	if err != nil {
		t.Fatalf("GetVolume: %s", err)
	}
	if volume.ID != "v-1" {
		t.Errorf("id = %q, want v-1", volume.ID)
	}
}

// TestGetVolumeEmptyArrayReportsMissing verifies that an empty data array
// reports a missing volume rather than decoding a bogus record.
func TestGetVolumeEmptyArrayReportsMissing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":[]}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	volume, err := c.GetVolume(context.Background(), "p-1", "az-1", "v-1")
	if err == nil {
		t.Fatal("expected an error when data is an empty array")
	}
	if volume != nil {
		t.Errorf("volume = %+v, want nil", volume)
	}
}

// TestWaitForVolumeReadyAcceptsArrayShape verifies readiness polling survives
// the array envelope returned right after a create.
func TestWaitForVolumeReadyAcceptsArrayShape(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":[{"id":"v-1","size":1,"status":"available"}]}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	volume, err := c.WaitForVolumeReady(context.Background(), "p-1", "az-1", "v-1", nil, time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForVolumeReady: %s", err)
	}
	if volume.ID != "v-1" {
		t.Errorf("id = %q, want v-1", volume.ID)
	}
}

// TestWaitForVolumeDeletedAcceptsArrayThenMissing verifies that the delete
// wait keeps polling through the array envelope and finishes on a 404.
func TestWaitForVolumeDeletedAcceptsArrayThenMissing(t *testing.T) {
	t.Parallel()

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if atomic.AddInt32(&calls, 1) == 1 {
			_, _ = w.Write([]byte(`{"status":true,"data":[{"id":"v-1","status":"available"}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Unable to Get volume: Volume v-1 could not be found. (HTTP 404)"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	if err := c.WaitForVolumeDeleted(context.Background(), "p-1", "az-1", "v-1", time.Millisecond); err != nil {
		t.Fatalf("WaitForVolumeDeleted: %s", err)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Errorf("expected at least 2 polls, got %d", calls)
	}
}

// TestWaitForVolumeDeletedFailsFastOnDeletingError verifies that a terminal
// error_deleting state aborts the wait immediately instead of polling until
// the context deadline.
func TestWaitForVolumeDeletedFailsFastOnDeletingError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"id":"v-1","status":"error_deleting"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.WaitForVolumeDeleted(ctx, "p-1", "az-1", "v-1", time.Millisecond)
	if err == nil {
		t.Fatal("expected an error when the volume is stuck in error_deleting")
	}
	if !strings.Contains(err.Error(), "error_deleting") {
		t.Errorf("error should name the state, got: %s", err)
	}
	if !strings.Contains(err.Error(), "v-1") {
		t.Errorf("error should name the volume id, got: %s", err)
	}
}
