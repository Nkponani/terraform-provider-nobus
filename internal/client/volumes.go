// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Volume paths documented by the Nobus OpenAPI specification.
const (
	volumesPath  = "/api/v3/volume/"
	volumeList   = "/api/v3/volume/list"
	volumeExtend = "/api/v3/volume/extend"
	volumeAttach = "/api/v3/volume/attach"
	volumeDetach = "/api/v3/volume/detach"
)

// DefaultAvailabilityZone is the availability zone the API defaults to when
// the caller omits it, as documented on the volume, image, flavor and keypair
// read operations.
const DefaultAvailabilityZone = "nobus-wa-az1"

// VolumeState mirrors the VolumeStatus enum documented by the API.
type VolumeState string

const (
	VolumeStateCreating       VolumeState = "creating"
	VolumeStateAvailable      VolumeState = "available"
	VolumeStateReserved       VolumeState = "reserved"
	VolumeStateAttaching      VolumeState = "attaching"
	VolumeStateDetaching      VolumeState = "detaching"
	VolumeStateInUse          VolumeState = "in-use"
	VolumeStateMaintenance    VolumeState = "maintenance"
	VolumeStateDeleting       VolumeState = "deleting"
	VolumeStateAwaitingXfer   VolumeState = "awaiting-transfer"
	VolumeStateError          VolumeState = "error"
	VolumeStateErrorDeleting  VolumeState = "error_deleting"
	VolumeStateBackingUp      VolumeState = "backing-up"
	VolumeStateRestoring      VolumeState = "restoring-backup"
	VolumeStateErrorRestore   VolumeState = "error_restoring"
	VolumeStateErrorExtending VolumeState = "error_extending"
)

// volumeFailureStates are the states from which a volume will never recover on
// its own; a waiter must abort as soon as one of these is observed.
var volumeFailureStates = map[VolumeState]bool{
	VolumeStateError:          true,
	VolumeStateErrorDeleting:  true,
	VolumeStateErrorRestore:   true,
	VolumeStateErrorExtending: true,
}

// volumeReadyStates are the states a volume may rest in.
var volumeReadyStates = map[VolumeState]bool{
	VolumeStateAvailable: true,
	VolumeStateInUse:     true,
	VolumeStateReserved:  true,
}

// CreateVolumeRequest mirrors VolumeCreateRequestSchema.
//
// Required by the API: Size, AvailabilityZone and ProjectID.
type CreateVolumeRequest struct {
	ProjectID        string            `json:"project_id"`
	AvailabilityZone string            `json:"availability_zone"`
	Size             int64             `json:"size"`
	Name             *string           `json:"name,omitempty"`
	Description      *string           `json:"description,omitempty"`
	VolumeType       *string           `json:"volume_type,omitempty"`
	SnapshotID       *string           `json:"snapshot_id,omitempty"`
	SourceVolID      *string           `json:"source_volid,omitempty"`
	ImageRef         *string           `json:"image_ref,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	Encrypted        *bool             `json:"encrypted,omitempty"`
}

// FlexBool decodes a JSON boolean that the Nobus API may serialise either as a
// real boolean or as a string ("true"/"false"). Volume reads return bootable as
// a string even though volume creates accept a boolean, so this type is used to
// tolerate both encodings.
type FlexBool bool

func (b *FlexBool) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*b = false
			return nil
		}
		parsed, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("cannot parse %q as a boolean: %w", s, err)
		}
		*b = FlexBool(parsed)
		return nil
	}
	var raw bool
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return err
	}
	*b = FlexBool(raw)
	return nil
}

// BoolPtr returns the value as a *bool, or nil when the receiver is nil.
func (b *FlexBool) BoolPtr() *bool {
	if b == nil {
		return nil
	}
	converted := bool(*b)
	return &converted
}

// VolumeAttachment mirrors VolumeAttachmentSchema.
type VolumeAttachment struct {
	ID           *string `json:"id"`
	AttachmentID *string `json:"attachment_id"`
	ServerID     *string `json:"server_id"`
	HostName     *string `json:"host_name"`
	Device       *string `json:"device"`
	AttachedAt   *string `json:"attached_at"`
}

// Volume mirrors VolumeSchema.
type Volume struct {
	ID               string             `json:"id"`
	Name             *string            `json:"name"`
	Description      *string            `json:"description"`
	Size             int64              `json:"size"`
	Status           *VolumeState       `json:"status"`
	CreatedAt        *string            `json:"created_at"`
	UpdatedAt        *string            `json:"updated_at"`
	Bootable         *FlexBool          `json:"bootable"`
	Encrypted        *bool              `json:"encrypted"`
	VolumeType       *string            `json:"volume_type"`
	Metadata         map[string]string  `json:"metadata"`
	Attachments      []VolumeAttachment `json:"attachments"`
	AvailabilityZone *string            `json:"availability_zone"`
	TenantID         *string            `json:"tenant_id"`
	ProjectID        *string            `json:"project_id"`
	UserID           *string            `json:"user_id"`
	SourceVolID      *string            `json:"source_volid"`
	SnapshotID       *string            `json:"snapshot_id"`
	Multiattach      *bool              `json:"multiattach"`
}

// UpdateVolumeRequest mirrors VolumeUpdateRequestSchema. Only name,
// description and metadata are mutable after creation.
type UpdateVolumeRequest struct {
	ProjectID        string            `json:"project_id"`
	AvailabilityZone string            `json:"availability_zone"`
	Name             *string           `json:"name,omitempty"`
	Description      *string           `json:"description,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// ExtendVolumeRequest mirrors ExtendVolumeRequestSchema. Volumes can only grow.
type ExtendVolumeRequest struct {
	ProjectID        string `json:"project_id"`
	AvailabilityZone string `json:"availability_zone"`
	VolumeID         string `json:"volume_id"`
	NewSize          int64  `json:"new_size"`
}

// AttachVolumeRequest mirrors AttachVolumeRequestSchema.
type AttachVolumeRequest struct {
	ProjectID        string `json:"project_id"`
	AvailabilityZone string `json:"availability_zone"`
	ServerID         string `json:"server_id"`
	VolumeID         string `json:"volume_id"`
}

// volumeResponse is the documented response of a volume create.
type volumeResponse struct {
	Status bool   `json:"status"`
	Data   Volume `json:"data"`
}

// genericVolumeResult decodes GenericVolumeResponseSchema, whose "data" member
// is documented as a single volume, a list of volumes or a string. Only the
// single volume form is meaningful for a read by ID.
type genericVolumeResult struct {
	Status bool            `json:"status"`
	Volume *Volume         `json:"-"`
	raw    json.RawMessage `json:"-"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *genericVolumeResult) UnmarshalJSON(data []byte) error {
	var envelope struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}

	r.Status = envelope.Status
	r.raw = envelope.Data
	if len(envelope.Data) == 0 {
		return nil
	}

	// The API answers a read by id with a single object once the record has
	// settled, but wraps the same volume in a one-element array while it is
	// fresh from a create, so accept both shapes.
	var single Volume
	if err := json.Unmarshal(envelope.Data, &single); err == nil && single.ID != "" {
		r.Volume = &single
		return nil
	}
	var many []Volume
	if err := json.Unmarshal(envelope.Data, &many); err == nil && len(many) > 0 {
		r.Volume = &many[0]
	}
	return nil
}

type volumeListResponse struct {
	Status bool     `json:"status"`
	Data   []Volume `json:"data"`
}

// CreateVolume creates a volume.
func (c *Client) CreateVolume(ctx context.Context, req CreateVolumeRequest) (*Volume, error) {
	var out volumeResponse
	var raw []byte
	if err := c.Do(ctx, Options{
		Method:  http.MethodPost,
		Path:    volumesPath,
		Body:    req,
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodPost, volumesPath, out.Status, raw); err != nil {
		return nil, err
	}
	if out.Data.ID == "" {
		return nil, &StatusError{
			Method:  http.MethodPost,
			Path:    volumesPath,
			Message: "the API did not return a volume id",
		}
	}

	return &out.Data, nil
}

// GetVolume reads a single volume.
func (c *Client) GetVolume(ctx context.Context, projectID, availabilityZone, volumeID string) (*Volume, error) {
	var out genericVolumeResult
	var raw []byte
	err := c.Do(ctx, Options{
		Method: http.MethodGet,
		Path:   volumesPath,
		Query: map[string]string{
			"project_id":        projectID,
			"availability_zone": availabilityZone,
			"volume_id":         volumeID,
		},
		Result:  &out,
		RawBody: &raw,
	})
	if err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, volumesPath, out.Status, raw); err != nil {
		return nil, err
	}
	if out.Volume == nil {
		return nil, &StatusError{
			Method:  http.MethodGet,
			Path:    volumesPath,
			Message: "the API did not return a volume for the requested id",
		}
	}

	return out.Volume, nil
}

// UpdateVolume updates the mutable attributes of a volume.
func (c *Client) UpdateVolume(ctx context.Context, req UpdateVolumeRequest) error {
	var out struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	var raw []byte
	if err := c.Do(ctx, Options{
		Method:  http.MethodPut,
		Path:    volumesPath,
		Body:    req,
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return err
	}

	return CheckStatus(http.MethodPut, volumesPath, out.Status, raw)
}

// ExtendVolume grows a volume to newSize.
func (c *Client) ExtendVolume(ctx context.Context, req ExtendVolumeRequest) error {
	var out struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	var raw []byte
	if err := c.Do(ctx, Options{
		Method:  http.MethodPost,
		Path:    volumeExtend,
		Body:    req,
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return err
	}

	return CheckStatus(http.MethodPost, volumeExtend, out.Status, raw)
}

// DeleteVolume deletes a volume. Callers should treat a not-found error as a
// successful deletion.
func (c *Client) DeleteVolume(ctx context.Context, projectID, availabilityZone, volumeID string) error {
	var out struct {
		Status bool `json:"status"`
	}
	var raw []byte
	if err := c.Do(ctx, Options{
		Method: http.MethodDelete,
		Path:   volumesPath,
		Query: map[string]string{
			"project_id":        projectID,
			"availability_zone": availabilityZone,
			"volume_id":         volumeID,
		},
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return err
	}

	return CheckStatus(http.MethodDelete, volumesPath, out.Status, raw)
}

// AttachVolume attaches a volume to an instance.
func (c *Client) AttachVolume(ctx context.Context, req AttachVolumeRequest) error {
	var out struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	var raw []byte
	if err := c.Do(ctx, Options{
		Method:  http.MethodPost,
		Path:    volumeAttach,
		Body:    req,
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return err
	}

	return CheckStatus(http.MethodPost, volumeAttach, out.Status, raw)
}

// DetachVolume detaches a volume from an instance.
func (c *Client) DetachVolume(ctx context.Context, req AttachVolumeRequest) error {
	var out struct {
		Status bool            `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	var raw []byte
	if err := c.Do(ctx, Options{
		Method:  http.MethodPost,
		Path:    volumeDetach,
		Body:    req,
		Result:  &out,
		RawBody: &raw,
	}); err != nil {
		return err
	}

	return CheckStatus(http.MethodPost, volumeDetach, out.Status, raw)
}

// ListVolumes returns one page of volumes.
func (c *Client) ListVolumes(ctx context.Context, projectID, availabilityZone string, limit int, marker string) ([]Volume, error) {
	query := map[string]string{
		"project_id":        projectID,
		"availability_zone": availabilityZone,
		"marker":            marker,
	}
	if limit > 0 {
		query["limit"] = strconv.Itoa(limit)
	}

	var out volumeListResponse
	var raw []byte
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: volumeList, Query: query, Result: &out, RawBody: &raw}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, volumeList, out.Status, raw); err != nil {
		return nil, err
	}

	return out.Data, nil
}

// WaitForVolumeReady polls the volume until it reaches one of the given target
// states, an unrecoverable error state or ctx is done.
//
// It is driven entirely by the status reported by the API; there are no fixed
// sleeps. When targetStates is empty a volume is considered ready once it is
// available, in use or reserved.
func (c *Client) WaitForVolumeReady(
	ctx context.Context,
	projectID, availabilityZone, volumeID string,
	targetStates []VolumeState,
	interval time.Duration,
) (*Volume, error) {
	var last *Volume
	var lastState VolumeState

	err := WaitFor(ctx, interval, func(ctx context.Context) (bool, error) {
		volume, err := c.GetVolume(ctx, projectID, availabilityZone, volumeID)
		if err != nil {
			// A volume that disappears while we wait was never created or has
			// already been removed.
			if IsNotFound(err) {
				return false, fmt.Errorf("nobus: volume %q disappeared while waiting for it to become ready: %w", volumeID, err)
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

		desired := volumeReadyStates
		if len(targetStates) > 0 {
			desired = make(map[VolumeState]bool, len(targetStates))
			for _, state := range targetStates {
				desired[state] = true
			}
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

// WaitForVolumeDeleted polls until the volume is no longer returned by the API.
func (c *Client) WaitForVolumeDeleted(
	ctx context.Context,
	projectID, availabilityZone, volumeID string,
	interval time.Duration,
) error {
	return WaitFor(ctx, interval, func(ctx context.Context) (bool, error) {
		volume, err := c.GetVolume(ctx, projectID, availabilityZone, volumeID)
		if err != nil {
			if IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		if volume == nil {
			return true, nil
		}
		// A delete that failed server-side never resolves on its own, so report
		// the terminal state instead of polling until the context deadline.
		if volume.Status != nil && volumeFailureStates[*volume.Status] {
			return false, fmt.Errorf(
				"nobus: volume %q entered the %q state while waiting for deletion",
				volumeID, *volume.Status,
			)
		}
		return false, nil
	})
}

// VolumeExists reports whether a volume is still present.
func (c *Client) VolumeExists(ctx context.Context, projectID, availabilityZone, volumeID string) (bool, error) {
	_, err := c.GetVolume(ctx, projectID, availabilityZone, volumeID)
	if err != nil {
		if IsNotFound(err) {
			return false, nil
		}
		var statusErr *StatusError
		if errors.As(err, &statusErr) && statusErr.IsNotFound() {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
