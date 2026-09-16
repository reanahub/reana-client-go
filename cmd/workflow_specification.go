/*
This file is part of REANA.
Copyright (C) 2026 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"errors"

	"github.com/reanahub/reana-client-go/client/operations"
)

func workflowSpecification(
	response *operations.GetWorkflowSpecificationOKBody,
) (*operations.GetWorkflowSpecificationOKBodySpecification, error) {
	if response == nil {
		return nil, errors.New("workflow specification response is empty")
	}
	if response.Specification == nil {
		return nil, errors.New(
			"workflow specification response is missing specification",
		)
	}
	return response.Specification, nil
}

func workflowInputs(
	response *operations.GetWorkflowSpecificationOKBody,
) ([]string, []string, error) {
	specification, err := workflowSpecification(response)
	if err != nil {
		return nil, nil, err
	}
	if specification.Inputs == nil {
		return nil, nil, nil
	}
	return specification.Inputs.Files, specification.Inputs.Directories, nil
}
