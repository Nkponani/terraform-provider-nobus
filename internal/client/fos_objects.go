// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

type FOSObject struct {
	Name         string  `json:"name"`
	Bytes        int64   `json:"bytes"`
	ContentType  *string `json:"content_type"`
	LastModified *string `json:"last_modified"`
	ETag         *string `json:"etag"`
	Subdirectory *string `json:"subdir"`
}

func (c *Client) UploadFOSObject(ctx context.Context, projectID, availabilityZone, containerName, objectName, sourcePath string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("opening FOS upload source %q: %w", sourcePath, err)
	}
	defer func() { _ = file.Close() }()

	staged, err := os.CreateTemp("", "nobus-fos-upload-*")
	if err != nil {
		return fmt.Errorf("staging FOS object upload: %w", err)
	}
	defer func() {
		_ = staged.Close()
		_ = os.Remove(staged.Name())
	}()

	multipartWriter := multipart.NewWriter(staged)
	contentType := multipartWriter.FormDataContentType()

	payloadPart, err := multipartWriter.CreateFormField("payload")
	if err != nil {
		return fmt.Errorf("encoding FOS object upload: %w", err)
	}
	payload := map[string]string{"project_id": projectID, "availability_zone": availabilityZone, "container_name": containerName}
	if err := json.NewEncoder(payloadPart).Encode(payload); err != nil {
		return fmt.Errorf("encoding FOS object upload: %w", err)
	}
	filePart, err := multipartWriter.CreateFormFile("file_data", objectName)
	if err != nil {
		return fmt.Errorf("encoding FOS object upload: %w", err)
	}
	if _, err := io.Copy(filePart, file); err != nil {
		return fmt.Errorf("reading FOS upload source %q: %w", sourcePath, err)
	}
	if err := multipartWriter.Close(); err != nil {
		return fmt.Errorf("encoding FOS object upload: %w", err)
	}

	info, err := staged.Stat()
	if err != nil {
		return fmt.Errorf("sizing FOS object upload: %w", err)
	}
	if _, err := staged.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewinding FOS object upload: %w", err)
	}

	var raw []byte
	var response fosMessageResponse
	if err := c.Do(ctx, Options{
		Method:        http.MethodPost,
		Path:          fosObjectUploadPath,
		BodyReader:    staged,
		ContentLength: info.Size(),
		ContentType:   contentType,
		Result:        &response,
		RawBody:       &raw,
	}); err != nil {
		return err
	}
	return checkFOSMessage(http.MethodPost, fosObjectUploadPath, response, raw)
}

func (c *Client) DeleteFOSObject(ctx context.Context, projectID, availabilityZone, containerName, objectName string) error {
	query := fosQuery(projectID, availabilityZone)
	query["container_name"] = containerName
	query["file_name"] = objectName
	var raw []byte
	var response fosMessageResponse
	if err := c.Do(ctx, Options{Method: http.MethodDelete, Path: fosObjectPath, Query: query, Result: &response, RawBody: &raw}); err != nil {
		return err
	}
	return checkFOSMessage(http.MethodDelete, fosObjectPath, response, raw)
}

func (c *Client) ListFOSObjects(ctx context.Context, projectID, availabilityZone, containerName string, options FOSListOptions) ([]FOSObject, error) {
	query := fosQuery(projectID, availabilityZone)
	query["container_name"] = containerName
	addFOSListOptions(query, options, true)
	var raw []byte
	var response struct {
		Status    bool        `json:"status"`
		Container string      `json:"container"`
		Objects   []FOSObject `json:"objects"`
		Count     int64       `json:"count"`
	}
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: fosObjectsPath, Query: query, Result: &response, RawBody: &raw}); err != nil {
		return nil, err
	}
	if err := CheckStatus(http.MethodGet, fosObjectsPath, response.Status, raw); err != nil {
		return nil, err
	}
	return response.Objects, nil
}

func (c *Client) DownloadFOSObject(ctx context.Context, projectID, availabilityZone, containerName, objectName string) ([]byte, error) {
	query := fosQuery(projectID, availabilityZone)
	query["container_name"] = containerName
	query["file_name"] = objectName
	var raw []byte
	if err := c.Do(ctx, Options{Method: http.MethodGet, Path: fosObjectDownloadPath, Query: query, Accept: "application/octet-stream", RawBody: &raw}); err != nil {
		return nil, err
	}
	return bytes.Clone(raw), nil
}
