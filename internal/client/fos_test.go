// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGetFOSAccountAcceptsStringAndNumericTotals(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != fosAccountPath || r.URL.Query().Get("project_id") != "project-1" {
			t.Errorf("request = %s %s, want %s with project_id", r.Method, r.URL.String(), fosAccountPath)
		}
		_, _ = w.Write([]byte(`{"status":true,"data":{"container_count":"2","object_count":7,"bytes_used":"1024"}}`))
	}))
	defer server.Close()

	account, err := newTestClient(t, server).GetFOSAccount(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if account.ContainerCount != 2 || account.ObjectCount != 7 || account.BytesUsed != 1024 {
		t.Fatalf("unexpected account totals: %+v", account)
	}
}

func TestFOSContainerRequests(t *testing.T) {
	t.Parallel()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.Method {
		case http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["project_id"] != "project-1" || body["availability_zone"] != "zone-1" || body["container_name"] != "assets" || body["is_public"] != true {
				t.Errorf("unexpected create body: %#v", body)
			}
			_, _ = w.Write([]byte(`{"status":true,"message":"created"}`))
		case http.MethodGet:
			query := r.URL.Query()
			if query.Get("project_id") != "project-1" || query.Get("availability_zone") != "zone-1" || query.Get("limit") != "25" || query.Get("prefix") != "ass" || query.Get("full_listing") != "true" {
				t.Errorf("unexpected list query: %v", query)
			}
			_, _ = w.Write([]byte(`{"status":true,"containers":[{"name":"assets","count":3,"bytes":99,"last_modified":"2026-10-12T10:00:00Z"}],"count":1}`))
		case http.MethodDelete:
			query := r.URL.Query()
			if query.Get("container_name") != "assets" || query.Get("project_id") != "project-1" {
				t.Errorf("unexpected delete query: %v", query)
			}
			_, _ = w.Write([]byte(`{"status":true,"message":"deleted"}`))
		default:
			t.Errorf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server)
	if err := c.CreateFOSContainer(context.Background(), "project-1", "zone-1", "assets", true); err != nil {
		t.Fatal(err)
	}
	containers, err := c.ListFOSContainers(context.Background(), "project-1", "zone-1", FOSListOptions{Limit: 25, Prefix: "ass", FullListing: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(containers) != 1 || containers[0].Name != "assets" || containers[0].Count != 3 || containers[0].Bytes != 99 {
		t.Fatalf("unexpected containers: %+v", containers)
	}
	if err := c.DeleteFOSContainer(context.Background(), "project-1", "zone-1", "assets"); err != nil {
		t.Fatal(err)
	}
	if requests != 3 {
		t.Fatalf("requests = %d, want 3", requests)
	}
}

func TestUploadFOSObjectMultipart(t *testing.T) {
	t.Parallel()
	source, err := os.CreateTemp(t.TempDir(), "fos-upload-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.WriteString("object content"); err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != fosObjectUploadPath {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		parts := map[string]string{}
		filenames := map[string]string{}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			content, err := io.ReadAll(part)
			if err != nil {
				t.Fatal(err)
			}
			parts[part.FormName()] = strings.TrimSpace(string(content))
			filenames[part.FormName()] = part.FileName()
		}
		var payload map[string]string
		if err := json.Unmarshal([]byte(parts["payload"]), &payload); err != nil {
			t.Fatalf("payload is not JSON: %q: %s", parts["payload"], err)
		}
		if payload["project_id"] != "project-1" || payload["availability_zone"] != "zone-1" || payload["container_name"] != "assets" {
			t.Errorf("unexpected payload: %v", payload)
		}
		if filenames["file_data"] != "report.txt" || parts["file_data"] != "object content" {
			t.Errorf("unexpected file part: filename=%q content=%q", filenames["file_data"], parts["file_data"])
		}
		_, _ = w.Write([]byte(`{"status":true,"message":"uploaded"}`))
	}))
	defer server.Close()

	if err := newTestClient(t, server).UploadFOSObject(context.Background(), "project-1", "zone-1", "assets", "report.txt", source.Name()); err != nil {
		t.Fatal(err)
	}
}

func TestListAndDownloadFOSObjects(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fosObjectsPath:
			query := r.URL.Query()
			if query.Get("container_name") != "assets" || query.Get("path") != "reports" || query.Get("delimiter") != "/" {
				t.Errorf("unexpected object list query: %v", query)
			}
			_, _ = w.Write([]byte(`{"status":true,"container":"assets","objects":[{"name":"reports/a.txt","bytes":4,"content_type":"text/plain","etag":"etag-1"}],"count":1}`))
		case fosObjectDownloadPath:
			if r.Header.Get("Accept") != "application/octet-stream" {
				t.Errorf("Accept = %q", r.Header.Get("Accept"))
			}
			if r.URL.Query().Get("file_name") != "reports/a.txt" {
				t.Errorf("unexpected download query: %v", r.URL.Query())
			}
			_, _ = w.Write([]byte{0x00, 0x01, 0xfe, 0xff})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server)
	objects, err := c.ListFOSObjects(context.Background(), "project-1", "zone-1", "assets", FOSListOptions{Path: "reports", Delimiter: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) != 1 || objects[0].Name != "reports/a.txt" || objects[0].Bytes != 4 {
		t.Fatalf("unexpected objects: %+v", objects)
	}
	content, err := c.DownloadFOSObject(context.Background(), "project-1", "zone-1", "assets", "reports/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string([]byte{0x00, 0x01, 0xfe, 0xff}) {
		t.Fatalf("unexpected content: %v", content)
	}
}
