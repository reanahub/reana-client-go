/*
This file is part of REANA.
Copyright (C) 2022 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"net/http"
	"testing"
)

var deletePathTemplate = "/api/workflows/%s/status"

func TestDelete(t *testing.T) {
	workflowName := "my_workflow"

	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(deletePathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "delete_success.json",
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"my_workflow has been deleted",
			},
		},
		"include workspace": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(deletePathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "delete_success.json",
				},
			},
			args: []string{"-w", workflowName, "--include-workspace"},
			expected: []string{
				"my_workflow has been deleted",
			},
		},
		"include all runs": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(deletePathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "delete_success.json",
				},
			},
			args: []string{"-w", workflowName, "--include-all-runs"},
			expected: []string{
				"All workflows named 'my_workflow' have been deleted",
			},
		},
		"include all runs complete name": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(deletePathTemplate, "my_workflow.10"): {
					statusCode:   http.StatusOK,
					responseFile: "delete_success.json",
				},
			},
			args: []string{"-w", "my_workflow.10", "--include-all-runs"},
			expected: []string{
				"All workflows named 'my_workflow' have been deleted",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "delete"
			testCmdRun(t, params)
		})
	}
}
