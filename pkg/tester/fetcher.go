/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package tester

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/reanahub/reana-client-go/client"
	"github.com/reanahub/reana-client-go/client/operations"
)

// APIFetcher retrieves workflow test data using the REANA REST API.
type APIFetcher struct {
	api *client.AuthenticatedClient
}

// NewAPIFetcher creates a workflow test data fetcher.
func NewAPIFetcher(api *client.AuthenticatedClient) *APIFetcher {
	return &APIFetcher{api: api}
}

// Status returns the status and run timestamps of a workflow.
func (f *APIFetcher) Status(workflow string) (WorkflowStatus, error) {
	params := operations.NewGetWorkflowStatusParams()
	params.SetWorkflowIDOrName(workflow)
	response, err := f.api.Operations.GetWorkflowStatus(params, nil)
	if err != nil {
		return WorkflowStatus{}, err
	}
	payload := response.GetPayload()
	if payload == nil {
		return WorkflowStatus{}, errors.New("workflow status response is empty")
	}
	status := WorkflowStatus{Name: payload.Name, Status: payload.Status}
	if payload.Progress != nil {
		if payload.Progress.RunStartedAt != nil {
			status.RunStartedAt = *payload.Progress.RunStartedAt
		}
		if payload.Progress.RunFinishedAt != nil {
			status.RunFinishedAt = *payload.Progress.RunFinishedAt
		}
	}
	return status, nil
}

// Specification returns outputs and workflow test files from the specification.
func (f *APIFetcher) Specification(
	workflow string,
) (WorkflowSpecification, error) {
	params := operations.NewGetWorkflowSpecificationParams()
	params.SetWorkflowIDOrName(workflow)
	response, err := f.api.Operations.GetWorkflowSpecification(params, nil)
	if err != nil {
		return WorkflowSpecification{}, err
	}
	payload := response.GetPayload()
	if payload == nil || payload.Specification == nil {
		return WorkflowSpecification{}, errors.New(
			"workflow specification response is empty",
		)
	}
	result := WorkflowSpecification{}
	if payload.Specification.Outputs != nil {
		result.OutputFiles = payload.Specification.Outputs.Files
		result.OutputDirectories = payload.Specification.Outputs.Directories
	}
	if payload.Specification.Tests != nil {
		result.TestFiles = payload.Specification.Tests.Files
	}
	return result, nil
}

// Files returns workspace files, optionally restricted by exact file name.
func (f *APIFetcher) Files(
	workflow,
	fileName string,
) ([]FileInfo, error) {
	params := operations.NewGetFilesParams()
	params.SetWorkflowIDOrName(workflow)
	if fileName != "" {
		params.SetFileName(&fileName)
	}
	response, err := f.api.Operations.GetFiles(params, nil)
	if err != nil {
		return nil, err
	}
	payload := response.GetPayload()
	if payload == nil {
		return nil, errors.New("workspace file response is empty")
	}
	files := make([]FileInfo, 0, len(payload.Items))
	for _, item := range payload.Items {
		if item == nil {
			continue
		}
		file := FileInfo{Name: item.Name}
		if item.Size != nil {
			file.Size = item.Size.Raw
			file.HumanReadable = item.Size.HumanReadable
		}
		files = append(files, file)
	}
	return files, nil
}

// DiskUsage returns workspace disk usage information.
func (f *APIFetcher) DiskUsage(
	workflow string,
	summarize bool,
) ([]FileInfo, error) {
	params := operations.NewGetWorkflowDiskUsageParams()
	params.SetWorkflowIDOrName(workflow)
	params.SetParameters(operations.GetWorkflowDiskUsageBody{
		Summarize: summarize,
	})
	response, err := f.api.Operations.GetWorkflowDiskUsage(params, nil)
	if err != nil {
		return nil, err
	}
	payload := response.GetPayload()
	if payload == nil {
		return nil, errors.New("workspace disk usage response is empty")
	}
	files := make([]FileInfo, 0, len(payload.DiskUsageInfo))
	for _, item := range payload.DiskUsageInfo {
		if item == nil {
			continue
		}
		file := FileInfo{Name: item.Name}
		if item.Size != nil {
			file.Size = item.Size.Raw
			file.HumanReadable = item.Size.HumanReadable
		}
		files = append(files, file)
	}
	return files, nil
}

// Logs returns engine and job logs, optionally restricted to workflow steps.
func (f *APIFetcher) Logs(
	workflow string,
	steps []string,
) (WorkflowLogs, error) {
	params := operations.NewGetWorkflowLogsParams()
	params.SetWorkflowIDOrName(workflow)
	if len(steps) > 0 {
		params.SetSteps(steps)
	}
	response, err := f.api.Operations.GetWorkflowLogs(params, nil)
	if err != nil {
		return WorkflowLogs{}, err
	}
	payload := response.GetPayload()
	if payload == nil {
		return WorkflowLogs{}, errors.New("workflow logs response is empty")
	}
	var logs WorkflowLogs
	if err := json.Unmarshal([]byte(payload.Logs), &logs); err != nil {
		return WorkflowLogs{}, fmt.Errorf(
			"could not decode workflow logs: %w",
			err,
		)
	}
	return logs, nil
}

// Download returns raw workspace file content.
func (f *APIFetcher) Download(
	workflow,
	fileName string,
) (DownloadedFile, error) {
	var content bytes.Buffer
	params := operations.NewDownloadFileParams()
	params.SetWorkflowIDOrName(workflow)
	params.SetFileName(fileName)
	response, err := f.api.Operations.DownloadFile(params, nil, &content)
	if err != nil {
		return DownloadedFile{}, err
	}
	return DownloadedFile{
		Content:   content.Bytes(),
		IsArchive: response.ContentType == "application/zip",
	}, nil
}
