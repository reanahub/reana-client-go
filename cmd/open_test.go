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
	"reanahub/reana-client-go/utils"
	"strings"
	"testing"
)

var openPathTemplate = "/api/workflows/%s/open/%s"

func TestOpen(t *testing.T) {
	successResponse := `{"path": "/test/jupyter"}`
	workflowName := "my_workflow"
	tests := map[string]TestCmdParams{
		"success default": {
			serverPath: fmt.Sprintf(
				openPathTemplate,
				workflowName,
				utils.InteractiveSessionTypes[0],
			),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName},
			expected: []string{
				"Interactive session opened successfully",
				"/test/jupyter?token=1234",
				"It could take several minutes to start the interactive session.",
			},
		},
		"success extra args": {
			serverPath:     fmt.Sprintf(openPathTemplate, workflowName, "jupyter"),
			serverResponse: successResponse,
			statusCode:     http.StatusOK,
			args:           []string{"-w", workflowName, "-i", "image", "jupyter"},
			expected: []string{
				"Interactive session opened successfully",
				"/test/jupyter?token=1234",
				"It could take several minutes to start the interactive session.",
			},
		},
		"invalid session type": {
			serverPath: fmt.Sprintf(openPathTemplate, workflowName, "invalid"),
			statusCode: http.StatusBadRequest,
			args:       []string{"-w", workflowName, "invalid"},
			expected: []string{
				fmt.Sprintf(
					"invalid value for 'interactive-session-type': 'invalid' is not part of '%s'",
					strings.Join(utils.InteractiveSessionTypes, "', '"),
				),
			},
			wantError: true,
		},
		"workflow already open": {
			serverPath:     fmt.Sprintf(openPathTemplate, workflowName, "jupyter"),
			serverResponse: `{"message": "Interactive session is already open"}`,
			statusCode:     http.StatusNotFound,
			args:           []string{"-w", workflowName},
			expected:       []string{"Interactive session is already open"},
			wantError:      true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "open"
			testCmdRun(t, params)
		})
	}
}
