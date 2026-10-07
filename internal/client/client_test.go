// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient returns a client pointed at the given test server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	c, err := New(Config{
		APIKey:     "test-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("New: %s", err)
	}
	return c
}

func TestNewRequiresAPIKey(t *testing.T) {
	t.Parallel()

	if _, err := New(Config{}); err == nil {
		t.Fatal("expected an error when the API key is empty")
	}
}

func TestNewNormalisesBaseURL(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":                            DefaultBaseURL,
		"cloud-api.nobus.io":          "https://cloud-api.nobus.io",
		"https://cloud-api.nobus.io/": "https://cloud-api.nobus.io",
		"https://example.test/base/":  "https://example.test/base",
	}

	for input, expected := range cases {
		c, err := New(Config{APIKey: "k", BaseURL: input})
		if err != nil {
			t.Fatalf("New(%q): %s", input, err)
		}
		if c.BaseURL() != expected {
			t.Errorf("New(%q).BaseURL() = %q, want %q", input, c.BaseURL(), expected)
		}
	}
}

// TestAuthenticationHeader verifies that the API key is sent as a bearer token,
// as documented by the http bearer security schemes in the specification.
func TestAuthenticationHeader(t *testing.T) {
	t.Parallel()

	var gotAuth, gotAccept, gotAgent, gotContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotAgent = r.Header.Get("User-Agent")
		gotContentType = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"id":"n-1","name":"net"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	if _, err := c.GetNetwork(context.Background(), "p-1", "az-1", "n-1"); err != nil {
		t.Fatalf("GetNetwork: %s", err)
	}

	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-key")
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if gotAgent != UserAgent {
		t.Errorf("User-Agent = %q, want %q", gotAgent, UserAgent)
	}
	if gotContentType != "" {
		t.Errorf("Content-Type = %q, want empty on a GET request", gotContentType)
	}
}

// TestQueryParameters verifies the required project_id, availability_zone and
// network_id query parameters.
func TestQueryParameters(t *testing.T) {
	t.Parallel()

	var gotQuery map[string][]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"network":{"id":"n-1"}}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	if _, err := c.GetNetwork(context.Background(), "p-1", "az-1", "n-1"); err != nil {
		t.Fatalf("GetNetwork: %s", err)
	}

	expected := map[string]string{
		"project_id":        "p-1",
		"availability_zone": "az-1",
		"network_id":        "n-1",
	}
	for key, want := range expected {
		if got := gotQuery[key]; len(got) != 1 || got[0] != want {
			t.Errorf("query %s = %v, want [%s]", key, got, want)
		}
	}
}

// TestRequestBodyEncoding verifies that the create payload matches the
// documented request model.
func TestRequestBodyEncoding(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	var gotPath, gotMethod, gotContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"network_id":"n-1","subnet_id":"s-1","router_id":"r-1"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	out, err := c.CreateNetwork(context.Background(), CreateNetworkRequest{
		ProjectID:        "p-1",
		AvailabilityZone: "az-1",
		NetworkName:      "production-network",
		SubnetCIDR:       "10.0.0.0/24",
		RouterName:       "production-router",
		DNSNameservers:   []string{"1.1.1.1"},
	})
	if err != nil {
		t.Fatalf("CreateNetwork: %s", err)
	}

	if out.NetworkID != "n-1" || out.SubnetID != "s-1" || out.RouterID != "r-1" {
		t.Errorf("unexpected create payload: %+v", out)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/api/v3/network/" {
		t.Errorf("path = %s, want /api/v3/network/", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}

	for key, want := range map[string]any{
		"project_id":        "p-1",
		"availability_zone": "az-1",
		"network_name":      "production-network",
		"subnet_cidr":       "10.0.0.0/24",
		"router_name":       "production-router",
	} {
		if gotBody[key] != want {
			t.Errorf("body %s = %v, want %v", key, gotBody[key], want)
		}
	}

	// enable_snat defaults to true in the API and must be sent only when set.
	if _, present := gotBody["enable_snat"]; present {
		t.Error("enable_snat must be omitted when it is not configured")
	}
	// external_network_id is nullable and must be omitted when unset.
	if _, present := gotBody["external_network_id"]; present {
		t.Error("external_network_id must be omitted when it is not configured")
	}
}

// TestStatusFalseIsAnError verifies that an HTTP 200 response reporting
// "status": false is not treated as success.
func TestStatusFalseIsAnError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":false,"data":{"message":"cidr 10.0.0.0/24 overlaps an existing subnet"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, err := c.CreateNetwork(context.Background(), CreateNetworkRequest{ProjectID: "p-1"})
	if err == nil {
		t.Fatal("expected an error when the API reports status false")
	}
	if !strings.Contains(err.Error(), "overlaps an existing subnet") {
		t.Errorf("error should preserve the API message, got: %s", err)
	}
}

// TestAPIErrorDecoding verifies the documented {"message": string} error body.
func TestAPIErrorDecoding(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"requested image is unavailable in the selected availability zone"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, err := c.GetNetwork(context.Background(), "p-1", "az-1", "n-1")
	if err == nil {
		t.Fatal("expected an error for HTTP 400")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected an *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", apiErr.StatusCode)
	}
	want := "requested image is unavailable in the selected availability zone"
	if apiErr.Message != want {
		t.Errorf("message = %q, want %q", apiErr.Message, want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Error() should include the API message, got: %s", err)
	}
}

// TestIsNotFound covers the ways a missing object surfaces in this API.
func TestIsNotFound(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		err     error
		want    bool
		comment string
	}{
		{"nil", nil, false, "a nil error is not a missing object"},
		{"404", &APIError{StatusCode: 404, Message: "Not Found"}, true, "HTTP 404"},
		{"400 not found", &APIError{StatusCode: 400, Message: "Network n-1 not found"}, true, "message on 400"},
		{"does not exist", &APIError{StatusCode: 500, Message: "Network does not exist"}, true, "phrasing variant"},
		{"status false", &StatusError{Message: "network not found"}, true, "HTTP 200 with status false"},
		{"unrelated", &APIError{StatusCode: 400, Message: "cidr overlaps an existing subnet"}, false, "a real error is not a missing object"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.want {
				t.Errorf("IsNotFound(%v) = %v, want %v (%s)", tc.err, got, tc.want, tc.comment)
			}
		})
	}
}

// TestDecodeError verifies that a malformed body is reported clearly.
func TestDecodeError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":"not-a-network"}`))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	network, err := c.GetNetwork(context.Background(), "p-1", "az-1", "n-1")
	if err == nil {
		t.Fatal("expected a decode error when data has the wrong shape")
	}
	if network != nil {
		t.Errorf("expected no network, got %+v", network)
	}
	if !strings.Contains(err.Error(), "decoding response") {
		t.Errorf("unexpected error: %s", err)
	}
}

// TestContextCancellation verifies that requests honour context cancellation.
func TestContextCancellation(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	c := newTestClient(t, server)
	if _, err := c.GetNetwork(ctx, "p-1", "az-1", "n-1"); err == nil {
		t.Fatal("expected the request to fail once the context deadline passed")
	}
}

// TestWaitFor verifies the polling helper used by the resource waiters.
func TestWaitFor(t *testing.T) {
	t.Parallel()

	t.Run("returns when the condition is met", func(t *testing.T) {
		calls := 0
		err := WaitFor(context.Background(), time.Millisecond, func(context.Context) (bool, error) {
			calls++
			return calls == 3, nil
		})
		if err != nil {
			t.Fatalf("WaitFor: %s", err)
		}
		if calls != 3 {
			t.Errorf("condition called %d times, want 3", calls)
		}
	})

	t.Run("propagates errors", func(t *testing.T) {
		sentinel := errors.New("boom")
		err := WaitFor(context.Background(), time.Millisecond, func(context.Context) (bool, error) {
			return false, sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("WaitFor returned %v, want %v", err, sentinel)
		}
	})

	t.Run("stops when Terraform cancels the context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := WaitFor(ctx, time.Millisecond, func(context.Context) (bool, error) { return false, nil })
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("WaitFor returned %v, want context cancellation", err)
		}
	})
}
