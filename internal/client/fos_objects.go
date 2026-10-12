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

	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	contentType := multipartWriter.FormDataContentType()
	writeErr := make(chan error, 1)
	go func() {
		var streamErr error
		defer func() {
			if closeErr := multipartWriter.Close(); streamErr == nil {
				streamErr = closeErr
			}
			if streamErr != nil {
				_ = writer.CloseWithError(streamErr)
			} else {
				_ = writer.Close()
			}
			writeErr <- streamErr
		}()

		payloadPart, partErr := multipartWriter.CreateFormField("payload")
		if partErr != nil {
			streamErr = partErr
			return
		}
		payload := map[string]string{"project_id": projectID, "availability_zone": availabilityZone, "container_name": containerName}
		if partErr = json.NewEncoder(payloadPart).Encode(payload); partErr != nil {
			streamErr = partErr
			return
		}
		filePart, partErr := multipartWriter.CreateFormFile("file_data", objectName)
		if partErr != nil {
			streamErr = partErr
			return
		}
		_, streamErr = io.Copy(filePart, file)
	}()

	var raw []byte
	var response fosMessageResponse
	requestErr := c.Do(ctx, Options{Method: http.MethodPost, Path: fosObjectUploadPath, BodyReader: reader, ContentType: contentType, Result: &response, RawBody: &raw})
	if requestErr != nil {
		_ = reader.CloseWithError(requestErr)
	}
	streamErr := <-writeErr
	if requestErr != nil {
		return requestErr
	}
	if streamErr != nil {
		return fmt.Errorf("encoding FOS object upload: %w", streamErr)
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
