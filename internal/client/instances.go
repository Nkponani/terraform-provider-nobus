// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Instance paths documented by the Nobus OpenAPI specification.
const (
	instancesPath      = "/api/v3/instance"
	instanceListPath   = "/api/v3/instance/list"
	instanceUpdatePath = "/api/v3/instance/update-server"
	instanceResizePath = "/api/v3/instance/resize"
	instanceShutdown   = "/api/v3/instance/shutdown"
)

// CreateInstanceRequest mirrors CreateInstanceRequestSchema.
//
// The API requires ProjectID, AvailabilityZone, Name, Image, Flavor and
// KeyName. Nics is always sent as "auto" because the deployed Nobus API
// expects the key even though the OpenAPI schema describes it as optional.
// NetworkID attaches the instance to an existing network and SecurityGroups
// defaults to ["default"] when omitted.
type CreateInstanceRequest struct {
	ProjectID        string   `json:"project_id"`
	AvailabilityZone string   `json:"availability_zone"`
	Name             string   `json:"name"`
	Image            string   `json:"image"`
	Flavor           string   `json:"flavor"`
	KeyName          string   `json:"key_name"`
	Nics             string   `json:"nics"`
	NetworkID        *string  `json:"network_id,omitempty"`
	VolumeSize       *int64   `json:"volume_size,omitempty"`
	SecurityGroups   []string `json:"security_groups,omitempty"`
	AdminPass        *string  `json:"admin_pass,omitempty"`
	Userdata         *string  `json:"userdata,omitempty"`
}

// UpdateInstanceRequest mirrors UpdateInstanceRequestSchema.
type UpdateInstanceRequest struct {
	ProjectID        string  `json:"project_id"`
	AvailabilityZone string  `json:"availability_zone"`
	ServerID         string  `json:"server_id"`
	Name             *string `json:"name,omitempty"`
	Description      *string `json:"description,omitempty"`
	Hostname         *string `json:"hostname,omitempty"`
}

// ResizeInstanceRequest mirrors ResizeInstanceRequestSchema.
type ResizeInstanceRequest struct {
	ProjectID        string `json:"project_id"`
	AvailabilityZone string `json:"availability_zone"`
	ServerID         string `json:"server_id"`
	FlavorID         string `json:"flavor_id"`
}

// InstanceVolume mirrors InstanceVolumeSchema.
type InstanceVolume struct {
	ID string `json:"id"`
}

// NIC mirrors NICSchema.
type NIC struct {
	NetID     *string `json:"net-id,omitempty"`
	PortID    *string `json:"port-id,omitempty"`
	V4FixedIP *string `json:"v4-fixed-ip,omitempty"`
	V6FixedIP *string `json:"v6-fixed-ip,omitempty"`
	NoFixedIP *bool   `json:"no-fixed-ip,omitempty"`
	Tag       *string `json:"tag,omitempty"`
}

// Security mirrors SecurityGroupOut schema.
type Security struct {
	ID                 string         `json:"id"`
	Name               *string        `json:"name,omitempty"`
	Description        *string        `json:"description,omitempty"`
	TenantID           *string        `json:"tenant_id,omitempty"`
	ProjectID          *string        `json:"project_id,omitempty"`
	Tags               []string       `json:"tags,omitempty"`
	Extra              map[string]any `json:"extra,omitempty"`
	SecurityGroupRules []any          `json:"security_group_rules,omitempty"`
}

// Instance mirrors InstanceSchema.
//
// Flavor and Image are returned as opaque objects and Addresses as an opaque
// map, so they are surfaced as raw JSON.
type Instance struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Status         *string          `json:"status"`
	Description    *string          `json:"description"`
	SecurityGroups []map[string]any `json:"security_groups"`
	KeyName        *string          `json:"key_name"`
	Created        *string          `json:"created"`
	Updated        *string          `json:"updated"`
	Addresses      map[string]any   `json:"addresses"`
	Flavor         map[string]any   `json:"flavor"`
	Image          map[string]any   `json:"image"`
	Volumes        []InstanceVolume `json:"volumes"`
}

type instanceResponse struct {
	Status bool     `json:"status"`
	Data   Instance `json:"data"`
}

type instanceListResponse struct {
	Status bool       `json:"status"`
	Data   []Instance `json:"data"`
}

type actionMessageResponse struct {
	Status bool   `json:"status"`
	Data   string `json:"data"`
}

// InstanceActionRequest is the union of the request fields accepted by the
// documented one-shot instance action endpoints. Unused fields are omitted.
type InstanceActionRequest struct {
	ProjectID         string   `json:"project_id"`
	AvailabilityZone  string   `json:"availability_zone"`
	ServerID          string   `json:"server_id"`
	RebootType        *string  `json:"reboot_type,omitempty"`
	FlavorID          *string  `json:"flavor_id,omitempty"`
	PortID            *string  `json:"port_id,omitempty"`
	NetworkID         *string  `json:"net_id,omitempty"`
	FixedIP           *string  `json:"fixed_ip,omitempty"`
	SecurityGroupName *string  `json:"security_group_name,omitempty"`
	ImageName         *string  `json:"image_name,omitempty"`
	NewPassword       *string  `json:"new_password,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Tag               *string  `json:"tag,omitempty"`
}

var instanceActionPaths = map[string]string{
	"shutdown":              "/api/v3/instance/shutdown",
	"reboot":                "/api/v3/instance/reboot-server",
	"resize":                "/api/v3/instance/resize",
	"confirm_resize":        "/api/v3/instance/confirm-resize",
	"revert_resize":         "/api/v3/instance/revert-resize",
	"suspend":               "/api/v3/instance/suspend-instance",
	"resume":                "/api/v3/instance/resume-instance",
	"lock":                  "/api/v3/instance/lock-instance",
	"unlock":                "/api/v3/instance/unlock-instance",
	"attach_interface":      "/api/v3/instance/attach-network-interface",
	"detach_interface":      "/api/v3/instance/detach-network-interface",
	"attach_security_group": "/api/v3/instance/attach-security-group",
	"remove_security_group": "/api/v3/instance/remove-security-group",
	"create_image":          "/api/v3/instance/create-server-image",
	"change_password":       "/api/v3/instance/change-server-password",
	"clear_password":        "/api/v3/instance/clear-password",
	"set_tags":              "/api/v3/instance/set-tags",
	"delete_tag":            "/api/v3/instance/delete-tags",
	"console_output":        "/api/v3/instance/get-console-output",
	"console_url":           "/api/v3/instance/get-console-url",
}

// InstanceAction invokes a documented one-shot compute operation and returns
// the API's data member as JSON for inspection in Terraform state.
func (c *Client) InstanceAction(ctx context.Context, action string, req InstanceActionRequest) (json.RawMessage, error) {
	endpoint, ok := instanceActionPaths[action]
	if !ok {
		return nil, fmt.Errorf("nobus: unsupported instance action %q", action)
	}
	var out struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	var raw []byte
	if err := c.Do(ctx, Options{Method: http.MethodPost, Path: endpoint, Body: req, Result: &out, RawBody: &raw}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodPost, endpoint, out.Status, raw); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// CreateInstance creates an instance.
func (c *Client) CreateInstance(ctx context.Context, req CreateInstanceRequest) (*Instance, error) {
	if req.Nics == "" {
		req.Nics = "auto"
	}

	var out instanceResponse
	var raw []byte
	if err := c.Do(ctx, Options{
		Method:  http.MethodPost,
		Path:    instancesPath,
		Body:    req,
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodPost, instancesPath, out.Status, raw); err != nil {
		return nil, err
	}
	if out.Data.ID == "" {
		return nil, &StatusError{
			Method:  http.MethodPost,
			Path:    instancesPath,
			Message: "the API did not return an instance id",
		}
	}

	return &out.Data, nil
}

// GetInstance reads a single instance.
func (c *Client) GetInstance(ctx context.Context, projectID, availabilityZone, serverID string) (*Instance, error) {
	var out instanceResponse
	var raw []byte
	err := c.Do(ctx, Options{
		Method: http.MethodGet,
		Path:   instancesPath,
		Query: map[string]string{
			"project_id":        projectID,
			"availability_zone": availabilityZone,
			"server_id":         serverID,
		},
		Result:  &out,
		RawBody: &raw,
	})
	if err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, instancesPath, out.Status, raw); err != nil {
		return nil, err
	}
	if out.Data.ID == "" {
		return nil, &StatusError{
			Method:  http.MethodGet,
			Path:    instancesPath,
			Message: "the API did not return an instance for the requested id",
		}
	}

	return &out.Data, nil
}

// UpdateInstance applies a partial update to an instance. The API supports
// changing the name, description and hostname only.
func (c *Client) UpdateInstance(ctx context.Context, req UpdateInstanceRequest) error {
	var out actionMessageResponse
	var raw []byte
	if err := c.Do(ctx, Options{
		Method:  http.MethodPost,
		Path:    instanceUpdatePath,
		Body:    req,
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return err
	}

	return CheckStatus(http.MethodPost, instanceUpdatePath, out.Status, raw)
}

// ResizeInstance changes the flavor of an instance.
func (c *Client) ResizeInstance(ctx context.Context, req ResizeInstanceRequest) error {
	var out actionMessageResponse
	var raw []byte
	if err := c.Do(ctx, Options{
		Method:  http.MethodPost,
		Path:    instanceResizePath,
		Body:    req,
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return err
	}

	return CheckStatus(http.MethodPost, instanceResizePath, out.Status, raw)
}

// DeleteInstance deletes an instance. When deleteVolumes is true the boot
// volume created with the instance is removed as well, which is the documented
// default for the delete_volume query parameter.
func (c *Client) DeleteInstance(ctx context.Context, projectID, availabilityZone, serverID string, deleteVolumes bool) error {
	var out actionMessageResponse
	var raw []byte
	if err := c.Do(ctx, Options{
		Method: http.MethodDelete,
		Path:   instancesPath,
		Query: map[string]string{
			"project_id":        projectID,
			"availability_zone": availabilityZone,
			"server_id":         serverID,
			"delete_volume":     strconv.FormatBool(deleteVolumes),
		},
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return err
	}

	return CheckStatus(http.MethodDelete, instancesPath, out.Status, raw)
}

// ListInstances returns one page of instances using the documented limit and
// marker pagination parameters.
func (c *Client) ListInstances(ctx context.Context, projectID, availabilityZone string, limit int, marker string) ([]Instance, error) {
	query := map[string]string{
		"project_id":        projectID,
		"availability_zone": availabilityZone,
		"marker":            marker,
	}
	if limit > 0 {
		query["limit"] = strconv.Itoa(limit)
	}

	var out instanceListResponse
	var raw []byte
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: instanceListPath, Query: query, Result: &out, RawBody: &raw}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, instanceListPath, out.Status, raw); err != nil {
		return nil, err
	}

	return out.Data, nil
}

// InstanceStateSuccess lists the instance states treated as ready.
//
// InstanceSchema.status is documented only as a free-form string, with no enum
// in the specification, so the accepted values cannot be derived from the
// specification alone. The API is known to report the provisioned state as
// "ACTIVE"; the equivalent lower case form is accepted as well.
var InstanceStateSuccess = []string{"ACTIVE", "active"}

// instanceFailureStates are states from which an instance never recovers.
var instanceFailureStates = map[string]bool{
	"ERROR":  true,
	"error":  true,
	"FAILED": true,
	"failed": true,
}

// WaitForInstanceReady polls the instance until its status is one of
// targetStates, an unrecoverable state is observed or ctx is done.
//
// Polling is driven purely by the status reported by the API; there is no fixed
// sleep. Because InstanceSchema.status has no documented enum, callers can
// override targetStates through the resource's instance_ready_states
// attribute when a Nobus deployment reports different values.
func (c *Client) WaitForInstanceReady(
	ctx context.Context,
	projectID, availabilityZone, serverID string,
	targetStates []string,
	interval time.Duration,
) (*Instance, error) {
	if len(targetStates) == 0 {
		targetStates = InstanceStateSuccess
	}
	desired := make(map[string]bool, len(targetStates))
	for _, state := range targetStates {
		desired[state] = true
	}

	var last *Instance
	lastState := ""

	err := WaitFor(ctx, interval, func(ctx context.Context) (bool, error) {
		instance, err := c.GetInstance(ctx, projectID, availabilityZone, serverID)
		if err != nil {
			if IsNotFound(err) {
				return false, fmt.Errorf("nobus: instance %q disappeared while waiting for it to become ready: %w", serverID, err)
			}
			return false, err
		}

		last = instance
		if instance.Status != nil {
			lastState = *instance.Status
		}

		if instanceFailureStates[lastState] {
			return false, fmt.Errorf("nobus: instance %q entered the %q state", serverID, lastState)
		}

		return desired[lastState], nil
	})
	if err != nil {
		if last != nil {
			return nil, fmt.Errorf("%w (last known status: %q)", err, lastState)
		}
		return nil, err
	}

	return last, nil
}

// WaitForInstanceDeleted polls until the instance is no longer returned.
func (c *Client) WaitForInstanceDeleted(
	ctx context.Context,
	projectID, availabilityZone, serverID string,
	interval time.Duration,
) error {
	return WaitFor(ctx, interval, func(ctx context.Context) (bool, error) {
		instance, err := c.GetInstance(ctx, projectID, availabilityZone, serverID)
		if err != nil {
			if IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		if instance == nil {
			return true, nil
		}
		return false, nil
	})
}

// InstanceExists reports whether an instance is still present.
func (c *Client) InstanceExists(ctx context.Context, projectID, availabilityZone, serverID string) (bool, error) {
	_, err := c.GetInstance(ctx, projectID, availabilityZone, serverID)
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// WaitForVolumeAttached polls until a volume reports the given attachment state.
func (c *Client) WaitForVolumeAttached(
	ctx context.Context,
	projectID, availabilityZone, volumeID string,
	attached bool,
	interval time.Duration,
) (*Volume, error) {
	var last *Volume
	lastState := VolumeState("")

	err := WaitFor(ctx, interval, func(ctx context.Context) (bool, error) {
		volume, err := c.GetVolume(ctx, projectID, availabilityZone, volumeID)
		if err != nil {
			if IsNotFound(err) {
				return false, fmt.Errorf("nobus: volume %q disappeared while waiting for it to be attached: %w", volumeID, err)
			}
			return false, err
		}

		last = volume
		if volume.Status != nil {
			lastState = *volume.Status
		}
		if volumeFailureStates[lastState] {
			return false, fmt.Errorf("nobus: volume %q entered the %q state", volumeID, lastState)
		}

		switch {
		case attached && lastState == VolumeStateInUse:
			return true, nil
		case !attached && lastState != VolumeStateInUse && volumeReadyStates[lastState]:
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		if last != nil {
			return nil, fmt.Errorf("%w (last known status: %q)", err, lastState)
		}
		return nil, err
	}

	return last, nil
}

// CreateInstanceFromVolume creates an instance from an existing volume.
// CreateInstanceFromVolumeRequest mirrors CreateInstanceFromVolumeRequestSchema.
type CreateInstanceFromVolumeRequest struct {
	ProjectID        string    `json:"project_id"`
	AvailabilityZone string    `json:"availability_zone"`
	Name             string    `json:"name"`
	Image            string    `json:"image"`
	Flavor           string    `json:"flavor"`
	VolumeSize       *int64    `json:"volume_size,omitempty"`
	NetworkID        *string   `json:"network_id,omitempty"`
	KeyName          string    `json:"key_name"`
	SecurityGroups   *[]string `json:"security_groups,omitempty"`
	Nics             *[]NIC    `json:"nics,omitempty"`
	Userdata         *string   `json:"userdata,omitempty"`
	Tags             *[]any    `json:"tags,omitempty"`
	VolumeID         string    `json:"volume_id"`
}

// InstanceNetworkInterface mirrors NetworkInterfaceSchema for instance interfaces.
type InstanceNetworkInterface struct {
	NetworkID string `json:"network_id"`
	PortID    string `json:"port_id"`
	MacAddr   string `json:"mac_addr"`
}

// InstanceDiagnostics mirrors ServerDiagnosticsSchema.
type InstanceDiagnostics struct {
	Disk     map[string]any `json:"disk"`
	CPU      map[string]any `json:"cpu"`
	Memory   map[string]any `json:"memory"`
	Network  map[string]any `json:"network"`
	General  map[string]any `json:"general"`
	Platform map[string]any `json:"platform"`
}

// InstanceTopology mirrors TopologySchema.
type InstanceTopology struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Devices map[string]any   `json:"devices"`
	Links   []map[string]any `json:"links"`
}

type instanceNetworkInterfacesResponse struct {
	Status bool                       `json:"status"`
	Data   []InstanceNetworkInterface `json:"data"`
}

type instanceDiagnosticsResponse struct {
	Status bool                `json:"status"`
	Data   InstanceDiagnostics `json:"data"`
}

type instanceSecurityGroupsResponse struct {
	Status bool       `json:"status"`
	Data   []Security `json:"data"`
}

type instanceTagsResponse struct {
	Status bool     `json:"status"`
	Data   []string `json:"data"`
}

type instanceTopologyResponse struct {
	Status bool             `json:"status"`
	Data   InstanceTopology `json:"data"`
}

func (c *Client) CreateInstanceFromVolume(ctx context.Context, req CreateInstanceFromVolumeRequest) (*Instance, error) {
	if req.SecurityGroups == nil || len(*req.SecurityGroups) == 0 {
		sg := []string{"default"}
		req.SecurityGroups = &sg
	}
	if req.Nics == nil {
		empty := []NIC{}
		req.Nics = &empty
	}

	var out instanceResponse
	var raw []byte
	if err := c.Do(ctx, Options{
		Method:  http.MethodPost,
		Path:    "/api/v3/instance/from-volume",
		Body:    req,
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodPost, "/api/v3/instance/from-volume", out.Status, raw); err != nil {
		return nil, err
	}
	if out.Data.ID == "" {
		return nil, &StatusError{
			Method:  http.MethodPost,
			Path:    "/api/v3/instance/from-volume",
			Message: "the API did not return an instance id",
		}
	}
	return &out.Data, nil
}

// GetInstanceNetworkInterfaces lists network interfaces attached to an instance.
func (c *Client) GetInstanceNetworkInterfaces(ctx context.Context, projectID, availabilityZone, serverID string) ([]InstanceNetworkInterface, error) {
	var out instanceNetworkInterfacesResponse
	var raw []byte
	err := c.Do(ctx, Options{
		Method: http.MethodGet,
		Path:   "/api/v3/instance/network-interfaces",
		Query: map[string]string{
			"project_id":        projectID,
			"availability_zone": availabilityZone,
			"server_id":         serverID,
		},
		Result:  &out,
		RawBody: &raw,
	})
	if err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, "/api/v3/instance/network-interfaces", out.Status, raw); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetInstanceDiagnostics retrieves server diagnostics.
func (c *Client) GetInstanceDiagnostics(ctx context.Context, projectID, availabilityZone, serverID string) (*InstanceDiagnostics, error) {
	var out instanceDiagnosticsResponse
	var raw []byte
	err := c.Do(ctx, Options{
		Method: http.MethodGet,
		Path:   "/api/v3/instance/diagnostics",
		Query: map[string]string{
			"project_id":        projectID,
			"availability_zone": availabilityZone,
			"server_id":         serverID,
		},
		Result:  &out,
		RawBody: &raw,
	})
	if err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, "/api/v3/instance/diagnostics", out.Status, raw); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

// GetInstanceSecurityGroups lists security groups attached to an instance.
func (c *Client) GetInstanceSecurityGroups(ctx context.Context, projectID, availabilityZone, serverID string) ([]Security, error) {
	var out instanceSecurityGroupsResponse
	var raw []byte
	err := c.Do(ctx, Options{
		Method: http.MethodGet,
		Path:   "/api/v3/instance/security-groups",
		Query: map[string]string{
			"project_id":        projectID,
			"availability_zone": availabilityZone,
			"server_id":         serverID,
		},
		Result:  &out,
		RawBody: &raw,
	})
	if err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, "/api/v3/instance/security-groups", out.Status, raw); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetInstanceTags lists tags attached to an instance.
func (c *Client) GetInstanceTags(ctx context.Context, projectID, availabilityZone, serverID string) ([]string, error) {
	var out instanceTagsResponse
	var raw []byte
	err := c.Do(ctx, Options{
		Method: http.MethodGet,
		Path:   "/api/v3/instance/tags",
		Query: map[string]string{
			"project_id":        projectID,
			"availability_zone": availabilityZone,
			"server_id":         serverID,
		},
		Result:  &out,
		RawBody: &raw,
	})
	if err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, "/api/v3/instance/tags", out.Status, raw); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetInstanceTopology retrieves instance topology.
func (c *Client) GetInstanceTopology(ctx context.Context, projectID, availabilityZone, serverID string) (*InstanceTopology, error) {
	var out instanceTopologyResponse
	var raw []byte
	err := c.Do(ctx, Options{
		Method: http.MethodGet,
		Path:   "/api/v3/instance/topology",
		Query: map[string]string{
			"project_id":        projectID,
			"availability_zone": availabilityZone,
			"server_id":         serverID,
		},
		Result:  &out,
		RawBody: &raw,
	})
	if err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, "/api/v3/instance/topology", out.Status, raw); err != nil {
		return nil, err
	}
	return &out.Data, nil
}
