// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package client implements a thin, dependency-free client for the public
// Nobus Cloud API (https://cloud-api.nobus.io/api/v3/openapi.json).
//
// All requests target paths under /api/v3 and are authenticated with a bearer
// token obtained from the Nobus API key endpoints. Nobus wraps every response
// in an envelope of the shape {"status": bool, "data": ...} and reports errors
// as {"message": string}.
//
// This package deliberately contains no Terraform specific code so that it can
// be reused and unit tested independently.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production Nobus Cloud API endpoint.
const DefaultBaseURL = "https://cloud-api.nobus.io"

// UserAgent is sent with every request to identify the provider.
const UserAgent = "terraform-provider-nobus"

// Config configures a Client.
type Config struct {
	// APIKey is the Nobus API key used as a bearer token. Required.
	APIKey string

	// BaseURL overrides the API endpoint. Defaults to DefaultBaseURL.
	BaseURL string

	// HTTPClient allows injecting a custom client, primarily for testing.
	HTTPClient *http.Client
}

// Client is a Nobus Cloud API client. It is safe for concurrent use.
type Client struct {
	apiKey    string
	baseURL   *url.URL
	http      *http.Client
	userAgent string
}

// New builds a Client from cfg.
func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("nobus: an API key is required")
	}

	raw := strings.TrimSpace(cfg.BaseURL)
	if raw == "" {
		raw = DefaultBaseURL
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	baseURL, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("nobus: invalid base URL %q: %w", cfg.BaseURL, err)
	}
	baseURL.Path = strings.TrimSuffix(baseURL.Path, "/")

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	return &Client{
		apiKey:    strings.TrimSpace(cfg.APIKey),
		baseURL:   baseURL,
		http:      httpClient,
		userAgent: UserAgent,
	}, nil
}

// BaseURL returns the configured API endpoint.
func (c *Client) BaseURL() string { return c.baseURL.String() }

// Options describes a single API call.
type Options struct {
	// Method defaults to GET.
	Method string

	// Path is the API path, for example "/api/v3/volume/". A leading slash is
	// optional. Relative paths are resolved against the configured base URL.
	Path string

	// Query holds URL query parameters. Empty values are omitted.
	Query map[string]string

	// Body, when non-nil, is JSON encoded as the request body.
	Body any

	// Result, when non-nil, receives the decoded response body.
	Result any

	// RawBody, when non-nil, receives the raw response body. It is used to
	// report the message Nobus returned when an operation reports
	// "status": false inside an HTTP 200 response.
	RawBody *[]byte
}

// Do performs an API request. It applies authentication, JSON encoding and
// decoding, query parameters, context cancellation and error translation.
//
// The caller is responsible for unwrapping the Nobus {"status", "data"}
// envelope: pass a result type that matches the documented response schema.
func (c *Client) Do(ctx context.Context, opts Options) error {
	method := opts.Method
	if method == "" {
		method = http.MethodGet
	}

	if opts.Path == "" {
		return fmt.Errorf("nobus: request path is required")
	}

	endpoint, err := c.baseURL.Parse(strings.TrimPrefix(opts.Path, "/"))
	if err != nil {
		return fmt.Errorf("nobus: invalid request path %q: %w", opts.Path, err)
	}
	if len(opts.Query) > 0 {
		q := endpoint.Query()
		for k, v := range opts.Query {
			if v == "" {
				continue
			}
			q.Set(k, v)
		}
		endpoint.RawQuery = q.Encode()
	}

	var reader io.Reader
	if opts.Body != nil {
		payload, err := json.Marshal(opts.Body)
		if err != nil {
			return fmt.Errorf("nobus: encoding request body for %s %s: %w", method, endpoint.Path, err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return fmt.Errorf("nobus: building request for %s %s: %w", method, endpoint.Path, err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if opts.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return &TransportError{Method: method, Path: endpoint.Path, Err: err}
	}
	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return &TransportError{Method: method, Path: endpoint.Path, Err: err}
	}

	if opts.RawBody != nil {
		*opts.RawBody = raw
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return newAPIError(method, endpoint.Path, res.StatusCode, raw)
	}

	if opts.Result == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, opts.Result); err != nil {
		return &DecodeError{Method: method, Path: endpoint.Path, StatusCode: res.StatusCode, Err: err}
	}

	return nil
}

// WaitFor polls condition until it reports completion or ctx is cancelled. It
// never imposes a deadline of its own and always honours context cancellation.
func WaitFor(ctx context.Context, interval time.Duration, condition func(context.Context) (bool, error)) error {
	if interval <= 0 {
		interval = 2 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		done, err := condition(ctx)
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
