/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"

	"reanahub/reana-client-go/client/operations"
)

func workflowProgress(
	response *operations.GetWorkflowStatusOKBody,
) (*operations.GetWorkflowStatusOKBodyProgress, error) {
	if response == nil {
		return nil, errors.New("workflow status response is empty")
	}
	if response.Progress == nil {
		return &operations.GetWorkflowStatusOKBodyProgress{}, nil
	}
	return response.Progress, nil
}
