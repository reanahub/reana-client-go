/*
This file is part of REANA.
Copyright (C) 2022, 2025, 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/validator"
)

// UpdateStatus updates the status of the specified workflow.
func UpdateStatus(
	token, workflow, status string,
	includeWorkspace, includeAllRuns bool,
) error {
	if err := validator.ValidateChoice(status, config.UpdateStatusActions, "status"); err != nil {
		return err
	}

	deleteParams := operations.NewSetWorkflowStatusParams()
	deleteParams.SetWorkflowIDOrName(workflow)
	deleteParams.SetStatus(status)
	deleteParams.SetParameters(operations.SetWorkflowStatusBody{
		AllRuns:   includeAllRuns,
		Workspace: includeWorkspace,
	})

	api, err := client.ApiClient(token)
	if err != nil {
		return err
	}
	_, err = api.Operations.SetWorkflowStatus(deleteParams, nil)
	if err != nil {
		return err
	}

	return nil
}

// GetStatus returns the status information of the specified workflow.
func GetStatus(
	token, workflow string,
) (*operations.GetWorkflowStatusOKBody, error) {
	getParams := operations.NewGetWorkflowStatusParams()
	getParams.SetWorkflowIDOrName(workflow)

	api, err := client.ApiClient(token)
	if err != nil {
		return nil, err
	}
	resp, err := api.Operations.GetWorkflowStatus(getParams, nil)
	if err != nil {
		return nil, err
	}

	return resp.GetPayload(), nil
}

// ListRuns returns a page of batch workflow runs and the total number matching
// the given workflow name and statuses.
func ListRuns(
	token, workflow string,
	statuses []string,
	page, size int64,
) ([]*operations.GetWorkflowsOKBodyItemsItems0, int64, error) {
	listParams := operations.NewGetWorkflowsParams()
	listParams.SetType("batch")
	listParams.SetWorkflowIDOrName(&workflow)
	listParams.SetStatus(statuses)
	listParams.SetPage(&page)
	listParams.SetSize(&size)

	api, err := client.ApiClient(token)
	if err != nil {
		return nil, 0, err
	}
	resp, err := api.Operations.GetWorkflows(listParams, nil)
	if err != nil {
		return nil, 0, err
	}
	if resp.GetPayload() == nil {
		return nil, 0, errors.New("workflow list response is empty")
	}

	return resp.GetPayload().Items, resp.GetPayload().Total, nil
}

// GetWorkflowSpecification returns the specification of the specified workflow.
func GetWorkflowSpecification(
	token, workflow string,
) (*operations.GetWorkflowSpecificationOKBody, error) {
	specParams := operations.NewGetWorkflowSpecificationParams()
	specParams.SetWorkflowIDOrName(workflow)

	api, err := client.ApiClient(token)
	if err != nil {
		return nil, err
	}
	resp, err := api.Operations.GetWorkflowSpecification(specParams, nil)
	if err != nil {
		return nil, err
	}

	return resp.GetPayload(), nil
}

// UploadFile uploads a file to the specified workflow.
func UploadFile(token, workflow, fileName string) (string, error) {
	return UploadFileAs(token, workflow, fileName, fileName)
}

// UploadFileAs uploads a local file under an explicit workspace path.
func UploadFileAs(
	token, workflow, localFileName, workspaceFileName string,
) (string, error) {
	// Preserve local validation before constructing a client so missing files
	// fail independently of REANA_SERVER_URL configuration.
	if err := validator.ValidateFile(localFileName); err != nil {
		return "", err
	}
	httpClient, serverURL, err := client.StreamingHTTPClient()
	if err != nil {
		return "", err
	}
	return uploadFileAs(httpClient, serverURL, token, workflow, localFileName,
		workspaceFileName)
}

func uploadFileAs(
	httpClient *http.Client,
	serverURL *url.URL,
	token, workflow, localFileName, workspaceFileName string,
) (string, error) {
	if err := validator.ValidateFile(localFileName); err != nil {
		return "", err
	}
	file, err := os.Open(localFileName)
	if err != nil {
		return "", fmt.Errorf(
			"file %s could not be uploaded: %s",
			localFileName, err.Error(),
		)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf(
			"file %s could not be inspected: %w",
			localFileName,
			err,
		)
	}

	// url.PathEscape encodes "/" and other reserved characters within the
	// workflow name, unlike setting url.URL.Path directly with an
	// unescaped value -- a workflow name containing ".." could otherwise
	// retarget ResolveReference's dot-segment resolution outside
	// /api/workflows/ entirely.
	reference, err := url.Parse(
		fmt.Sprintf("/api/workflows/%s/workspace", url.PathEscape(workflow)),
	)
	if err != nil {
		return "", fmt.Errorf("could not build workspace URL: %w", err)
	}
	endpoint := serverURL.ResolveReference(reference)
	query := endpoint.Query()
	query.Set("file_name", workspaceFileName)
	endpoint.RawQuery = query.Encode()
	var body io.Reader = http.NoBody
	if info.Size() > 0 {
		body = io.LimitReader(file, info.Size())
	}
	request, err := http.NewRequest(http.MethodPost, endpoint.String(), body)
	if err != nil {
		return "", err
	}
	request.ContentLength = info.Size()
	request.Header.Set("Content-Type", "application/octet-stream")
	accessToken, err := client.AccessToken(context.Background(), token)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("could not read upload response: %w", err)
	}
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return "", fmt.Errorf("could not decode upload response: %w", err)
	}
	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("%s", payload.Message)
	}
	return payload.Message, nil
}

// DownloadFile downloads a file of the specified workflow.
func DownloadFile(
	token, workflow, fileName string,
) (string, *bytes.Buffer, bool, error) {
	fileBuf := new(bytes.Buffer)
	downloadParams := operations.NewDownloadFileParams()
	downloadParams.SetWorkflowIDOrName(workflow)
	downloadParams.SetFileName(fileName)

	api, err := client.ApiClient(token)
	if err != nil {
		return "", nil, false, err
	}
	downloadResp, err := api.Operations.DownloadFile(
		downloadParams,
		nil,
		fileBuf,
	)
	if err != nil {
		return "", nil, false, err
	}

	// parse Content-Disposition header to extract a filename
	_, params, err := mime.ParseMediaType(downloadResp.ContentDisposition)
	if err != nil {
		return "", nil, false, err
	}
	name := "downloaded_file"
	if val, ok := params["filename"]; ok {
		name = val
	}

	// a zip archive is downloaded if multiple files are requested
	multipleFilesZipped := downloadResp.ContentType == "application/zip"

	return name, fileBuf, multipleFilesZipped, nil
}
