/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package workflows

import (
	"reanahub/reana-client-go/client"
	"reanahub/reana-client-go/client/operations"
	"reanahub/reana-client-go/pkg/config"
	"reanahub/reana-client-go/pkg/validator"
)

// UpdateStatus updates the status of the specified workflow.
func UpdateStatus(
	api *client.API,
	token, workflow, status string,
	includeWorkspace, includeAllRuns bool,
) error {
	if err := validator.ValidateChoice(status, config.GetRunStatuses(true), "status"); err != nil {
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

	_, err := api.Operations.SetWorkflowStatus(deleteParams)
	if err != nil {
		return err
	}

	return nil
}

// GetStatus returns the status information of the specified workflow.
func GetStatus(api *client.API, token, workflow string) (*operations.GetWorkflowStatusOKBody, error) {
	getParams := operations.NewGetWorkflowStatusParams()
	getParams.SetAccessToken(&token)
	getParams.SetWorkflowIDOrName(workflow)

	resp, err := api.Operations.GetWorkflowStatus(getParams)
	if err != nil {
		return nil, err
	}

	return resp.GetPayload(), nil
}
