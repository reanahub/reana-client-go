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

var stopPathTemplate = "/api/workflows/%s/status"

func TestStop(t *testing.T) {
	workflowName := "my_workflow"

	tests := map[string]TestCmdParams{
		"default": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(stopPathTemplate, workflowName): {
					statusCode:   http.StatusOK,
					responseFile: "stop_success.json",
				},
			},
			args: []string{"-w", workflowName, "--force"},
			expected: []string{
				"my_workflow has been stopped",
			},
		},
		"graceful stop error": {
			serverResponses: nil,
			args:            []string{"-w", workflowName},
			expected: []string{
				"graceful stop not implemented yet",
			},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "stop"
			testCmdRun(t, params)
		})
	}
}
