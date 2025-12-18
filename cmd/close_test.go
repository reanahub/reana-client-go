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

var closePathTemplate = "/api/workflows/%s/close/"

func TestClose(t *testing.T) {
	tests := map[string]TestCmdParams{
		"success": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(closePathTemplate, "my_workflow"): {
					statusCode:   http.StatusOK,
					responseFile: "common_empty.json",
				},
			},
			args: []string{"-w", "my_workflow"},
			expected: []string{
				"Interactive session for workflow my_workflow was successfully closed",
			},
		},
		"error": {
			serverResponses: map[string]ServerResponse{
				fmt.Sprintf(closePathTemplate, "my_workflow"): {
					statusCode:   http.StatusNotFound,
					responseFile: "close_no_open.json",
				},
			},
			args: []string{"-w", "my_workflow"},
			expected: []string{
				"Workflow - my_workflow has no open interactive session.",
			},
			wantError: true,
		},
	}

	for name, params := range tests {
		t.Run(name, func(t *testing.T) {
			params.cmd = "close"
			testCmdRun(t, params)
		})
	}
}
