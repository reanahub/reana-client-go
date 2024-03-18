/*
This file is part of REANA.
Copyright (C) 2023 CERN.

REANA is free software; you can redistribute it and/or modify it under the terms
of the MIT License; see LICENSE file for more details.
*/

package cmd

import (
	"fmt"
	"net/http"
	"testing"
)

var shareAddPathTemplate = "/api/workflows/%s/share"

func TestShareAdd(t *testing.T) {
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareAddPathTemplate, workflowName): {
					statusCode: http.StatusOK,
				},
			},
			args: []string{"-w", workflowName, "--user", "bob@cern.ch"},
			expected: []string{
				"my_workflow is now read-only shared with bob@cern.ch",
			},
		},
		"with message and valid-until": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareAddPathTemplate, workflowName): {
					statusCode: http.StatusOK,
				},
			},
			args: []string{
				"-w", workflowName,
				"--user", "bob@cern.ch",
				"--message", "Please review my analysis",
				"--valid-until", "2024-12-31",
			},
			expected: []string{
				"my_workflow is now read-only shared with bob@cern.ch",
			},
		},
		"invalid workflow": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(shareAddPathTemplate, "invalid"): {
					statusCode:   http.StatusNotFound,
					responseFile: "common_invalid_workflow.json",
				},
			},
			args: []string{
				"-w", "invalid",
				"--user", "bob@cern.ch",
			},
			expected: []string{
				"REANA_WORKON is set to invalid, but that workflow does not exist.",
			},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "share-add"
			testCmdRun(t, params)
		})
	}
}
