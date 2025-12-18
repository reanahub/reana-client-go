/*
This file is part of REANA.
Copyright (C) 2023 CERN.

REANA is free software; you can redistribute it and/or modify it
under the terms of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"net/http"
	"testing"
)

var prunePathTemplate = "/api/workflows/%s/prune"

func TestPrune(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(prunePathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "prune_success.json",
				},
			},
			args: []string{"-w", workflowName},
			expected: []string{
				"The workspace has been correctly pruned.",
			},
		},
		"include inputs and outputs": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(prunePathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "prune_success.json",
				},
			},
			args: []string{
				"-w",
				workflowName,
				"--include-inputs",
				"--include-outputs",
			},
			expected: []string{
				"The workspace has been correctly pruned.",
			},
		},
		"invalid workflow": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(prunePathTemplate, "invalid"): {
					statusCode:   http.StatusNotFound,
					responseFile: "common_invalid_workflow.json",
				},
			},
			args:      []string{"-w", "invalid", "--include-inputs"},
			wantError: true,
			expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "prune"
			testCmdRun(t, params)
		})
	}
}
