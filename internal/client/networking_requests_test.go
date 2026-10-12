// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdatedNetworkingRequestEnvelopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		call                   func(*Client) error
		idKey, objectKey, path string
	}{
		{"floating ip", func(c *Client) error {
			_, e := c.UpdateFloatingIP(context.Background(), "p", "az", "f-1", FloatingIPUpdate{})
			return e
		}, "floatingip_id", "floatingip", floatingIPsPath},
		{"security group", func(c *Client) error {
			_, e := c.UpdateSecurityGroup(context.Background(), "p", "az", "sg-1", SecurityGroupUpdate{})
			return e
		}, "securitygroup_id", "security_group", securityGroupsPath},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var body map[string]any
			var path string
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.Path
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":true,"data":{}}`))
			}))
			defer s.Close()
			if e := tt.call(newTestClient(t, s)); e != nil {
				t.Fatal(e)
			}
			if path != tt.path {
				t.Errorf("path=%q want %q", path, tt.path)
			}
			if body[tt.idKey] == nil {
				t.Errorf("missing %s in %#v", tt.idKey, body)
			}
			if body[tt.objectKey] == nil {
				t.Errorf("missing %s in %#v", tt.objectKey, body)
			}
		})
	}
}

func TestClearPortAddressPairsUsesQuery(t *testing.T) {
	t.Parallel()
	var query map[string][]string
	var bodyBytes int64
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		bodyBytes = r.ContentLength
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":null}`))
	}))
	defer s.Close()
	if e := newTestClient(t, s).ClearPortAddressPairs(context.Background(), "p", "az", "port-1"); e != nil {
		t.Fatal(e)
	}
	if query["port_id"][0] != "port-1" || query["project_id"][0] != "p" || query["availability_zone"][0] != "az" {
		t.Errorf("unexpected query: %#v", query)
	}
	if bodyBytes > 0 {
		t.Errorf("clear address pairs must not send a request body")
	}
}
