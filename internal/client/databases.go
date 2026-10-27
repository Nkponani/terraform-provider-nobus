// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	databasePath               = "/api/v3/database"
	databaseResumePath         = "/api/v3/database/resume"
	databaseListPath           = "/api/v3/database/databases"
	databaseCreationStatusPath = "/api/v3/database/creation_status"
	databaseStatusPath         = "/api/v3/database/db_status"
	databaseActionPath         = "/api/v3/database/action"
	databaseSizePath           = "/api/v3/database/size"
	databaseResizeFlavorPath   = "/api/v3/database/resize_flavor"
	databaseCredentialsPath    = "/api/v3/database/database_credentials"
	databaseZabbixPath         = "/api/v3/database/zabbix_credentials"
	databaseLogsPath           = "/api/v3/database/logs"
	databaseChangePasswordPath = "/api/v3/database/changedbcredential"
	databaseExistingPath       = "/api/v3/database/existing_database"
)

type DatabaseCreateRequest struct {
	ProjectID        string `json:"project_id"`
	AvailabilityZone string `json:"availability_zone"`
	UserEmail        string `json:"user_email"`
	FlavorName       string `json:"flavor_name"`
	DatabaseName     string `json:"database_name"`
	DatabaseType     string `json:"database_type"`
	DatabaseVersion  string `json:"database_version"`
	DatabaseSize     int64  `json:"database_size"`
	InstallType      string `json:"install_type"`
	CustomerUserName string `json:"customer_user_name"`
	CustomerPassword string `json:"customer_user_password"`
	Zones            []Azs  `json:"zones"`
	ZabbixID         string `json:"zabbix_id,omitempty"`
	Encrypted        *bool  `json:"encrypted,omitempty"`
}

// Azs mirrors the Azs schema: one availability zone entry of a database
// deployment. The API requires the zones member of a create request to be an
// array of these objects, not a list of zone names.
type Azs struct {
	ProjectID        string `json:"project_id"`
	AvailabilityZone string `json:"availability_zone"`
	NoOfMasters      int64  `json:"no_of_masters"`
	NoOfSlaves       int64  `json:"no_of_slaves"`
	Origin           *bool  `json:"origin,omitempty"`
}

type Database struct {
	DatabaseID       string          `json:"database_id"`
	DatabaseName     string          `json:"database_name"`
	DatabaseType     string          `json:"database_type"`
	DatabaseVersion  string          `json:"database_version"`
	InstallationType string          `json:"installation_type"`
	InstallType      string          `json:"install_type"`
	ProjectID        string          `json:"project_id"`
	CreatedAt        string          `json:"created_at"`
	Status           string          `json:"status"`
	InstanceCount    int64           `json:"instance_count"`
	MasterCount      int64           `json:"number_of_masters"`
	SlaveCount       int64           `json:"number_of_slaves"`
	Encrypted        *bool           `json:"encrypted"`
	Instances        json.RawMessage `json:"instances"`
}

type DatabaseCreateResult struct {
	DatabaseID string
	TaskID     string
	Data       json.RawMessage
}

type DatabaseCredentials struct {
	Instances        json.RawMessage `json:"instances"`
	VolumeSize       int64           `json:"volume_size"`
	CustomerName     string          `json:"customer_name"`
	CustomerPassword string          `json:"customer_password"`
	DatabaseID       string          `json:"database_id"`
	DatabaseType     string          `json:"database_type"`
	InstallType      string          `json:"install_type"`
	ProjectID        string          `json:"project_id"`
	AvailabilityZone string          `json:"availability_zone"`
	Status           string          `json:"status"`
	DatabaseName     string          `json:"database_name"`
}

func (c *Client) CreateDatabase(ctx context.Context, request DatabaseCreateRequest) (*DatabaseCreateResult, error) {
	var raw []byte
	var response struct {
		Status  *bool           `json:"status"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := c.Do(ctx, Options{Method: http.MethodPost, Path: databasePath, Body: request, Result: &response, RawBody: &raw}); err != nil {
		return nil, err
	}
	if response.Status != nil && !*response.Status {
		return nil, &StatusError{Method: http.MethodPost, Path: databasePath, Message: response.Message}
	}
	return &DatabaseCreateResult{DatabaseID: findJSONString(response.Data, "database_id", "id"), TaskID: findJSONString(response.Data, "task_id"), Data: response.Data}, nil
}

func (c *Client) GetDatabase(ctx context.Context, projectID, availabilityZone, databaseID string) (*Database, error) {
	var response Database
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: databasePath, Query: databaseQuery(projectID, availabilityZone, databaseID), Result: &response}); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) ListDatabases(ctx context.Context, projectID, availabilityZone string) ([]Database, error) {
	var response []Database
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: databaseListPath, Query: fosQuery(projectID, availabilityZone), Result: &response}); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) DeleteDatabase(ctx context.Context, projectID, availabilityZone, databaseID string) error {
	return c.Do(ctx, Options{Method: http.MethodDelete, Path: databasePath, Query: databaseQuery(projectID, availabilityZone, databaseID)})
}

func (c *Client) ResumeDatabase(ctx context.Context, projectID, availabilityZone, databaseID string) error {
	return c.databaseGETMutation(ctx, databaseResumePath, databaseQuery(projectID, availabilityZone, databaseID))
}

func (c *Client) DatabaseAction(ctx context.Context, projectID, availabilityZone, databaseID, databaseType, action string) error {
	query := databaseQuery(projectID, availabilityZone, databaseID)
	query["db_type"], query["action"] = databaseType, action
	return c.databaseGETMutation(ctx, databaseActionPath, query)
}

func (c *Client) ResizeDatabase(ctx context.Context, projectID, availabilityZone, databaseID string, size int64) error {
	query := databaseQuery(projectID, availabilityZone, databaseID)
	query["new_size"] = strconv.FormatInt(size, 10)
	return c.databaseGETMutation(ctx, databaseSizePath, query)
}

func (c *Client) ResizeDatabaseFlavor(ctx context.Context, projectID, availabilityZone, databaseID, flavorID string) error {
	query := databaseQuery(projectID, availabilityZone, databaseID)
	query["flavor_id"] = flavorID
	return c.databaseGETMutation(ctx, databaseResizeFlavorPath, query)
}

func (c *Client) databaseGETMutation(ctx context.Context, path string, query map[string]string) error {
	return c.Do(ctx, Options{Method: http.MethodGet, Path: path, Query: query})
}

func (c *Client) ChangeDatabasePassword(ctx context.Context, projectID, availabilityZone, databaseID, databaseType, password string) error {
	body := map[string]string{"project_id": projectID, "availability_zone": availabilityZone, "database_id": databaseID, "db_type": databaseType, "new_password": password}
	return c.Do(ctx, Options{Method: http.MethodPost, Path: databaseChangePasswordPath, Body: body})
}

func (c *Client) GetDatabaseCreationStatus(ctx context.Context, projectID, availabilityZone, databaseID string) (json.RawMessage, error) {
	return c.databaseRawGET(ctx, databaseCreationStatusPath, databaseQuery(projectID, availabilityZone, databaseID))
}

func (c *Client) GetDatabaseStatus(ctx context.Context, projectID, availabilityZone, databaseID string, replicaCheck *bool) (json.RawMessage, error) {
	query := databaseQuery(projectID, availabilityZone, databaseID)
	if replicaCheck != nil {
		query["replica_check"] = strconv.FormatBool(*replicaCheck)
	}
	return c.databaseRawGET(ctx, databaseStatusPath, query)
}

func (c *Client) GetDatabaseCredentials(ctx context.Context, projectID, availabilityZone, databaseID string) (*DatabaseCredentials, error) {
	var response DatabaseCredentials
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: databaseCredentialsPath, Query: databaseQuery(projectID, availabilityZone, databaseID), Result: &response}); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *Client) GetDatabaseZabbixCredentials(ctx context.Context, projectID, availabilityZone, databaseID string) (json.RawMessage, error) {
	return c.databaseRawGET(ctx, databaseZabbixPath, databaseQuery(projectID, availabilityZone, databaseID))
}

func (c *Client) GetDatabaseLogs(ctx context.Context, projectID, availabilityZone, databaseID, databaseType string, lineCount int64) (json.RawMessage, error) {
	query := databaseQuery(projectID, availabilityZone, databaseID)
	query["db_type"], query["line_count"] = databaseType, strconv.FormatInt(lineCount, 10)
	return c.databaseRawGET(ctx, databaseLogsPath, query)
}

func (c *Client) databaseRawGET(ctx context.Context, path string, query map[string]string) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: path, Query: query, Result: &raw}); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) RegisterExistingDatabases(ctx context.Context, payload json.RawMessage) error {
	var body any
	if err := json.Unmarshal(payload, &body); err != nil {
		return err
	}
	return c.Do(ctx, Options{Method: http.MethodPost, Path: databaseExistingPath, Body: body})
}

func databaseQuery(projectID, availabilityZone, databaseID string) map[string]string {
	query := fosQuery(projectID, availabilityZone)
	query["database_id"] = databaseID
	return query
}

func findJSONString(raw json.RawMessage, keys ...string) string {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	var walk func(any) string
	walk = func(current any) string {
		switch typed := current.(type) {
		case map[string]any:
			for _, key := range keys {
				if found, ok := typed[key].(string); ok && found != "" {
					return found
				}
			}
			for _, nested := range typed {
				if found := walk(nested); found != "" {
					return found
				}
			}
		case []any:
			for _, nested := range typed {
				if found := walk(nested); found != "" {
					return found
				}
			}
		}
		return ""
	}
	return walk(value)
}
