/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package workflows

import (
	"fmt"
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
	deleteParams.SetAccessToken(&token)
	deleteParams.SetWorkflowIDOrName(workflow)
	deleteParams.SetStatus(status)
	deleteParams.SetParameters(operations.SetWorkflowStatusBody{
		AllRuns:   includeAllRuns,
		Workspace: includeWorkspace,
	})

	api, err := client.ApiClient()
	if err != nil {
		return err
	}
	_, err = api.Operations.SetWorkflowStatus(deleteParams)
	if err != nil {
		return err
	}

	return nil
}

// GetStatus returns the status information of the specified workflow.
func GetStatus(token, workflow string) (*operations.GetWorkflowStatusOKBody, error) {
	getParams := operations.NewGetWorkflowStatusParams()
	getParams.SetAccessToken(&token)
	getParams.SetWorkflowIDOrName(workflow)

	api, err := client.ApiClient()
	if err != nil {
		return nil, err
	}
	resp, err := api.Operations.GetWorkflowStatus(getParams)
	if err != nil {
		return nil, err
	}

	return resp.GetPayload(), nil
}

// GetWorkflowSpecification returns the specification of the specified workflow.
func GetWorkflowSpecification(
	token, workflow string,
) (*operations.GetWorkflowSpecificationOKBody, error) {
	specParams := operations.NewGetWorkflowSpecificationParams()
	specParams.SetAccessToken(&token)
	specParams.SetWorkflowIDOrName(workflow)

	api, err := client.ApiClient()
	if err != nil {
		return nil, err
	}
	resp, err := api.Operations.GetWorkflowSpecification(specParams)
	if err != nil {
		return nil, err
	}

	return resp.GetPayload(), nil
}

// UploadFile uploads a file to the specified workflow.
func UploadFile(token, workflow, fileName string) (string, error) {
	if err := validator.ValidateFile(fileName); err != nil {
		return "", err
	}
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf(
			"file %s could not be uploaded: %s",
			fileName, err.Error(),
		)
	}
	uploadParams := operations.NewUploadFileParams()
	uploadParams.SetAccessToken(&token)
	uploadParams.SetWorkflowIDOrName(workflow)
	uploadParams.SetFileName(fileName)
	uploadParams.SetFile(string(fileData))

	api, err := client.ApiClient()
	if err != nil {
		return "", err
	}
	uploadResp, err := api.Operations.UploadFile(uploadParams)
	if err != nil {
		return "", err
	}
	return uploadResp.GetPayload().Message, nil
}
