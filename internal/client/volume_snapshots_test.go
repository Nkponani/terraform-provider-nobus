// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStorageListUsesItemsEnvelope(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{"items":[{"id":"s-1","volume_id":"v-1"}]}}`))
	}))
	defer s.Close()
	items, e := newTestClient(t, s).ListSnapshots(context.Background(), "p", "az", nil)
	if e != nil {
		t.Fatal(e)
	}
	if len(items) != 1 || items[0].ID != "s-1" {
		t.Fatalf("unexpected snapshots: %#v", items)
	}
}
