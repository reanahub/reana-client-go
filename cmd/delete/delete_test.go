/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package delete

import (
	"fmt"
	"net/http"
	. "reanahub/reana-client-go/cmd/internal"
	"testing"
)

var deletePathTemplate = "/api/workflows/%s/status"

func TestDelete(t *testing.T) {
	workflowName := "my_workflow"

	tests := map[string]TestCmdParams{
		"default": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(deletePathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName},
			Expected: []string{
				"my_workflow has been deleted",
			},
		},
		"include workspace": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(deletePathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "--include-workspace"},
			Expected: []string{
				"my_workflow has been deleted",
			},
		},
		"include all runs": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(deletePathTemplate, workflowName): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", workflowName, "--include-all-runs"},
			Expected: []string{
				"All workflows named 'my_workflow' have been deleted",
			},
		},
		"include all runs complete name": {
			ServerResponses: map[string]ServerResponse{
				fmt.Sprintf(deletePathTemplate, "my_workflow.10"): {
					StatusCode:   http.StatusOK,
					ResponseFile: "testdata/success.json",
				},
			},
			Args: []string{"-w", "my_workflow.10", "--include-all-runs"},
			Expected: []string{
				"All workflows named 'my_workflow' have been deleted",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.Cmd = NewCmd()
			TestCmdRun(t, params)
		})
	}
}
